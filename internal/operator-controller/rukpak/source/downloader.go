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
	"path/filepath"
	"regexp"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	tlsSecret := &corev1.Secret{}
	if err := i.Get(ctx,
		types.NamespacedName{
			Name:      "registry-cert",
			Namespace: "olmv1-system",
		},
		tlsSecret); err != nil {
		return nil, err
	}

	// Get http client with the OLMv1 CA certificate
	client, err := httpClientWithCustomCA(tlsSecret)
	if err != nil {
		return nil, err
	}

	addr, err := url.Parse(chart)
	if err != nil {
		return nil, fmt.Errorf("invalid helm chart uri")
	}

	// OCI urls do not have a scheme and will return
	// an empty scheme and host
	if addr.Scheme != "" && addr.Host != "" {
		return (&httpDownloader{
			Client: client,
		}).download(ctx, chart)
	}

	return (&ociDownloader{
		Client: client,
	}).download(ctx, chart)
}

func (d *httpDownloader) download(ctx context.Context, url string) (*PullResponse, error) {
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

func (d *ociDownloader) download(ctx context.Context, url string) (*PullResponse, error) {
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

func httpClientWithCustomCA(tlsSecret *corev1.Secret) (*http.Client, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	if _, ok := tlsSecret.Data["tls.crt"]; !ok {
		return httpClient, fmt.Errorf("kubernetes tls secret not found")
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return httpClient, fmt.Errorf("x509 certificate pool; %w", err)
	}

	if !certPool.AppendCertsFromPEM(tlsSecret.Data["ca.crt"]) {
		return nil, fmt.Errorf("error appending provided CA certificate")
	}

	// Create default http.Client if no tls certificate was created
	certPEM := bytes.TrimRight(tlsSecret.Data["tls.crt"], "\n")
	keyPEM := bytes.TrimRight(tlsSecret.Data["tls.key"], "\n")

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return httpClient, fmt.Errorf("create x509 tls certificate; %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            certPool,
			InsecureSkipVerify: false,
		},
		Proxy: http.ProxyFromEnvironment,
	}
	httpClient.Transport = transport

	return httpClient, nil
}
