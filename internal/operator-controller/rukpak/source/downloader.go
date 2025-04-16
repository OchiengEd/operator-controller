package source

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/registry"
)

type HelmDownloader interface {
	download(ctx context.Context, url string) (*PullResponse, error)
}

var _ HelmDownloader = &httpDownloader{}

var _ HelmDownloader = &ociDownloader{}

type ociDownloader struct {
	*http.Client
}

type httpDownloader struct {
	*http.Client
}

type PullResponse struct {
	// chart represents a helm chart object
	chart *chart.Chart
	// chart archive as a []byte slice
	raw []byte
	// filename represents the name of the chart
	filename string
}

func (r *PullResponse) FileName() string {
	return r.filename
}

func (r *PullResponse) Chart() *chart.Chart {
	return r.chart
}

func (r *PullResponse) WriteTo(path string) (string, error) {
	return chartutil.Save(r.chart, path)
}

func (r *PullResponse) Bytes() []byte {
	return r.raw
}

func (r *PullResponse) IsChart() bool {
	return r.chart != nil &&
		r.chart.Metadata != nil &&
		func() bool {
			err := r.chart.Metadata.Validate()
			return (err == nil)
		}()
}

func (i *TarGZ) PullChart(ctx context.Context, chart string) (*PullResponse, error) {
	// Get http client with the OLMv1 CA certificate
	client, err := httpClientWithCustomCA()
	if err != nil {
		return nil, err
	}

	addr, err := url.Parse(chart)
	if err != nil {
		return nil, fmt.Errorf("invalid helm chart uri")
	}

	// If URL contains a scheme and the scheme starts
	//  with 'HTTP', then it would need a HTTP downloader
	if addr.Scheme != "" &&
		strings.HasPrefix(addr.Scheme, "http") {
		return (&httpDownloader{
			Client: client,
		}).download(ctx, chart)
	}

	return (&ociDownloader{
		Client: client,
	}).download(ctx, chart)
}

func (d *httpDownloader) download(_ context.Context, url string) (*PullResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/octet-stream")

	filename := filepath.Base(url)
	resp, err := d.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http response")
	}

	chart, err := loader.LoadArchive(resp.Body)
	if err != nil {
		return nil, err
	}

	return &PullResponse{chart, raw, filename}, nil
}

func (d *ociDownloader) download(_ context.Context, url string) (*PullResponse, error) {
	re := regexp.MustCompile(`^(?P<host>[a-zA-Z0-9\:\-\.]+)\/.*$`)
	matches := re.FindStringSubmatch(url)
	host := matches[re.SubexpIndex("host")]

	client, err := registry.NewClient(
		registry.ClientOptEnableCache(true),
		registry.ClientOptHTTPClient(d.Client),
	)
	if err != nil {
		return nil, err
	}

	if err := client.Login(host, registry.LoginOptInsecure(true)); err != nil {
		return nil, err
	}

	res, err := client.Pull(url,
		registry.PullOptWithChart(true),
		registry.PullOptWithProv(false),
	)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf(
		"%s-%s.tgz",
		res.Chart.Meta.Name,
		res.Chart.Meta.Version,
	)

	raw := res.Chart.Data
	data := bytes.NewBuffer(res.Chart.Data)
	chart, err := loader.LoadArchive(data)
	if err != nil {
		return nil, err
	}

	return &PullResponse{chart, raw, filename}, nil
}

func httpClientWithCustomCA() (*http.Client, error) {
	httpClient := &http.Client{}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return httpClient, fmt.Errorf("create certificate pool; %w", err)
	}

	// Read and append OLM CA certificate to certificate pool
	caFileName := "/var/certs/olm-ca.crt"
	f, err := os.Open(caFileName)
	if err != nil {
		return httpClient, fmt.Errorf("open CA cert at %s; %v", caFileName, err)
	}
	defer f.Close()

	pem, err := io.ReadAll(f)
	if err != nil {
		return httpClient, fmt.Errorf("reading OLM CA certificate; %v", err)
	}

	if !certPool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("error appending provided CA certificate")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            certPool,
			InsecureSkipVerify: false,
		},
		Proxy: http.ProxyFromEnvironment,
	}
	httpClient.Transport = transport

	return httpClient, nil
}
