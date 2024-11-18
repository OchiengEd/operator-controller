package source

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TarGZ struct {
	BaseCachePath string
}

// downloader uses a custom HTTP client that accepts a URL and returns a HTTP response and an error
// The client would be aware of know user-generated CA certificates
func downloader(ctx context.Context, url string) (*http.Response, error) {
	// Get a kubernetes client
	client, err := kubeClient()
	if err != nil {
		return nil, fmt.Errorf("getting k8s client; %w", err)
	}

	// Retrieve the contents of the tls secret associated with cert-manager olmv1-ca certificate object
	secret, err := client.CoreV1().Secrets("olmv1-system").Get(ctx, "olmv1-cert", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieving ca cert secret; %w", err)
	}

	// Create a PEM certificate using the contents of the tls secret
	caPEM := []byte{}
	for _, v := range secret.Data {
		caPEM = append(caPEM[:], v[:]...)
	}

	// Append the PEM certificate to a new certificate bool
	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caPEM); !ok {
		return nil, errors.New("error creating PEM encoded certificate")
	}

	// Create a custom HTTP client that will use the new certificate pool
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				RootCAs:            caPool,
			},
		},
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)

	return httpClient.Do(req)
}

func kubeClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func (i *TarGZ) Unpack(ctx context.Context, bundle *BundleSource) (*Result, error) {
	l := log.FromContext(ctx)

	if bundle.Image == nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error parsing bundle, bundle %s has a nil image source", bundle.Name))
	}

	// Parse the URL
	parsedURL, err := url.Parse(bundle.Image.Ref)
	if err != nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error downloading bundle '%s': %v", bundle.Name, err))
	}
	fileName := path.Base(parsedURL.Path)

	// Download the .tgz file
	resp, err := downloader(ctx, bundle.Image.Ref)
	if err != nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error downloading bundle '%s': %v", bundle.Name, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, reconcile.TerminalError(fmt.Errorf("error downloading bundle '%s': got status code '%d'", bundle.Name, resp.StatusCode))
	}

	// Open a gzip reader
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error unpacking bundle '%s': %v", bundle.Name, err))
	}
	defer gzReader.Close()

	unpackDir := path.Join(i.BaseCachePath, bundle.Name, fileName)
	err = os.MkdirAll(unpackDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}

	// Open a tar reader
	tarReader := tar.NewReader(gzReader)
	topLevelDir := ""
	// Extract tar contents
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, reconcile.TerminalError(fmt.Errorf("error unpaking bundle '%s': %v", bundle.Name, err))
		}

		// On the first entry, capture the top-level directory
		if topLevelDir == "" {
			topLevelDir = strings.Split(header.Name, "/")[0]
		}

		// Strip the top-level directory from the path
		relativePath := strings.TrimPrefix(header.Name, topLevelDir+"/")

		if relativePath == "" {
			// Skip the top-level directory itself
			continue
		}

		// Construct the target file path
		targetPath := filepath.Join(unpackDir, relativePath)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return nil, reconcile.TerminalError(fmt.Errorf("error unpacking bundle '%s': %v", bundle.Name, err))
			}
		case tar.TypeReg:
			// Ensure the directory for the file exists
			if err := os.MkdirAll(filepath.Dir(targetPath), os.FileMode(0700)); err != nil {
				return nil, reconcile.TerminalError(fmt.Errorf("error unpacking bundle '%s': %v", bundle.Name, err))
			}

			// Create a file
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return nil, reconcile.TerminalError(fmt.Errorf("error unpacking bundle '%s': %v", bundle.Name, err))
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return nil, reconcile.TerminalError(fmt.Errorf("error unpacking bundle '%s': %v", bundle.Name, err))
			}
			outFile.Close()
		default:
			// Handle other types of files if necessary (e.g., symlinks, etc.)
			l.V(2).Info("Skipping unsupported file type in tar: %s\n", header.Name)
		}
	}

	return successHelmUnpackResult(bundle.Name, unpackDir, bundle.Image.Ref), nil
}

func successHelmUnpackResult(bundleName, unpackPath string, chartgz string) *Result {
	return &Result{
		Bundle:         os.DirFS(unpackPath),
		ResolvedSource: &BundleSource{Type: SourceTypeImage, Name: bundleName, Image: &ImageSource{Ref: chartgz}},
		State:          StateUnpacked,
		Message:        fmt.Sprintf("unpacked %q successfully", chartgz),
	}
}

func (i *TarGZ) Cleanup(_ context.Context, bundle *BundleSource) error {
	return deleteRecursive(i.bundlePath(bundle.Name))
}

func (i *TarGZ) bundlePath(bundleName string) string {
	return filepath.Join(i.BaseCachePath, bundleName)
}

func (i *TarGZ) unpackPath(bundleName string, digest digest.Digest) string {
	return filepath.Join(i.bundlePath(bundleName), digest.String())
}

func (i *TarGZ) unpackImage(ctx context.Context, unpackPath string, imageReference types.ImageReference, sourceContext *types.SystemContext) error {
	img, err := imageReference.NewImage(ctx, sourceContext)
	if err != nil {
		return fmt.Errorf("error reading image: %w", err)
	}
	defer func() {
		if err := img.Close(); err != nil {
			panic(err)
		}
	}()

	layoutSrc, err := imageReference.NewImageSource(ctx, sourceContext)
	if err != nil {
		return fmt.Errorf("error creating image source: %w", err)
	}

	if err := os.MkdirAll(unpackPath, 0700); err != nil {
		return fmt.Errorf("error creating unpack directory: %w", err)
	}
	l := log.FromContext(ctx)
	l.Info("unpacking image", "path", unpackPath)
	for i, layerInfo := range img.LayerInfos() {
		if err := func() error {
			layerReader, _, err := layoutSrc.GetBlob(ctx, layerInfo, none.NoCache)
			if err != nil {
				return fmt.Errorf("error getting blob for layer[%d]: %w", i, err)
			}
			defer layerReader.Close()

			if err := applyLayer(ctx, unpackPath, layerReader); err != nil {
				return fmt.Errorf("error applying layer[%d]: %w", i, err)
			}
			l.Info("applied layer", "layer", i)
			return nil
		}(); err != nil {
			return errors.Join(err, deleteRecursive(unpackPath))
		}
	}
	if err := setReadOnlyRecursive(unpackPath); err != nil {
		return fmt.Errorf("error making unpack directory read-only: %w", err)
	}
	return nil
}

func (i *TarGZ) deleteOtherImages(bundleName string, digestToKeep digest.Digest) error {
	bundlePath := i.bundlePath(bundleName)
	imgDirs, err := os.ReadDir(bundlePath)
	if err != nil {
		return fmt.Errorf("error reading image directories: %w", err)
	}
	for _, imgDir := range imgDirs {
		if imgDir.Name() == digestToKeep.String() {
			continue
		}
		imgDirPath := filepath.Join(bundlePath, imgDir.Name())
		if err := deleteRecursive(imgDirPath); err != nil {
			return fmt.Errorf("error removing image directory: %w", err)
		}
	}
	return nil
}
