package source

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"helm.sh/helm/v3/pkg/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type downloadManager interface {
	Download(context.Context, *http.Client) ([]byte, error)
}

type httpDownloader struct {
	url string
}

type ociDownloader struct {
	url string
}

// newDowloadManager accepts a chart URL, detects protocol and
// returns a suitable download manager. It checks if the URL
// scheme is "oci", if not "http" protocol is assumed.
func newDownloadManager(chart string) downloadManager {
	uri := strings.SplitN(chart, "://", 2)

	if len(uri) > 1 {
		var proto, cleanURL string = uri[0], uri[1]

		if proto == "oci" {
			return &ociDownloader{
				url: cleanURL,
			}
		}
	}

	return &httpDownloader{
		url: chart,
	}
}

func (d *httpDownloader) Download(ctx context.Context, httpClient *http.Client) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, d.url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting repsonse; %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error downloading helm chart '%s': got status code '%d'", filepath.Base(d.url), resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (d *ociDownloader) Download(ctx context.Context, httpClient *http.Client) ([]byte, error) {
	re := regexp.MustCompile(`^(?P<host>[a-zA-Z0-9\:\-\.]+)\/.*$`)
	matches := re.FindStringSubmatch(d.url)
	var host string = matches[re.SubexpIndex("host")]

	client, err := registry.NewClient(registry.ClientOptHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	if err := client.Login(host, registry.LoginOptInsecure(true)); err != nil {
		return nil, err
	}

	res, err := client.Pull(chartNameConverter(d.url), registry.PullOptWithChart(true))
	if err != nil {
		return nil, err
	}

	return res.Chart.DescriptorPullSummary.Data, nil
}

// chartNameConverter accepts a helm compatible OCI chart URL
// and return a chart name in the format <chart-name>:<version>
// Input URL is returned if an invalid URL
func chartNameConverter(chartURL string) string {
	re := regexp.MustCompile(`(?P<base>.*)-(?P<ver>[0-9\.]+).tgz`)
	match := re.FindStringSubmatch(chartURL)
	if len(match) == 0 {
		return chartURL
	}
	var baseURL string = match[re.SubexpIndex("base")]
	var version string = match[re.SubexpIndex("ver")]
	return fmt.Sprintf("%s:%s", baseURL, version)
}

func kubeClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("",
			filepath.Join(os.Getenv("HOME"), ".kube", "config"),
		)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func httpClientWithCA(ctx context.Context, certificate, namespace string) (*http.Client, error) {
	caPool, err := x509.SystemCertPool()
	if err != nil {
		return &http.Client{}, fmt.Errorf("system certificate pool; %w", err)
	}

	// Get a kubernetes
	client, err := kubeClient()
	if err != nil {
		return &http.Client{}, err
	}

	// Retrieve the contents of the tls secret associated with cert-manager olmv1-ca certificate object
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, certificate, metav1.GetOptions{})
	if err != nil {
		return &http.Client{}, err
	}

	// Create a PEM certificate using the contents of the tls secret
	pem := []byte{}
	for _, v := range secret.Data {
		pem = append(pem[:], v[:]...)
	}

	// Append the PEM certificate to the system certificate pool
	if ok := caPool.AppendCertsFromPEM(pem); !ok {
		return nil, errors.New("error creating PEM encoded certificate")
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: err != nil,
				RootCAs:            caPool,
			},
		},
	}, nil
}
