package image

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"
	"helm.sh/helm/v3/pkg/registry"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func hasChart(imgCloser types.ImageCloser) bool {
	config := imgCloser.ConfigInfo()
	return config.MediaType == registry.ConfigMediaType
}

func pullChart(ctx context.Context, src reference.Named) (fs.FS, time.Time, error) {
	logger := log.FromContext(ctx, "ref", src.String())

	logger.Info("pulling helm chart", "ref", src.String())
	httpClient, err := httpClientWithCustomCA()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("creating custom http client; %+v\n", err)
	}

	client, err := registry.NewClient(
		registry.ClientOptEnableCache(true),
		registry.ClientOptHTTPClient(httpClient),
	)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("create helm oci client; %+v\n", err)
	}

	res, err := client.Pull(src.String(),
		registry.PullOptWithChart(true),
		registry.PullOptWithProv(false),
	)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("pull helm chart; %+v\n", err)
	}

	filename := fmt.Sprintf(
		"%s-%s.tgz",
		res.Chart.Meta.Name,
		res.Chart.Meta.Version,
	)

	raw := res.Chart.Data
	if err := os.MkdirAll("/var/cache/charts", 0750); err != nil {
		return nil, time.Time{}, fmt.Errorf("create helm chart cache; %+v\n", err)
	}

	f, err := os.Create(filepath.Join("/var/cache/charts/", filename))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("open helm chart file; %+v\n", err)
	}
	defer f.Close()

	_, err = f.Write(raw)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("writing helm chart; %+v\n", err)
	}

	return os.DirFS("/var/cache/charts"), time.Time{}, nil
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
