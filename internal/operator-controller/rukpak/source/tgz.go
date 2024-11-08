package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TarGZ struct {
	BaseCachePath string
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
	if strings.Contains(fileName, ":") {
		s := strings.Split(fileName, ":")
		fileName = fmt.Sprintf("%s-%s.tgz", s[0], s[1])
	}

	// Append OLMv1 CA certificate file in PEM format to the system certificate pool
	certPool, err := buildCertPool(ctx, "olmv1-cert", "olmv1-system")

	// Create a custom HTTP client that will use the new certificate pool
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: err != nil,
				RootCAs:            certPool,
			},
		},
	}

	// Download the .tgz file
	downloader, err := newChartDownloader(bundle.Image.Ref)
	if err != nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error setting up chart downloader: %v", err))
	}
	content, err := downloader.Download(ctx, httpClient)
	if err != nil {
		return nil, reconcile.TerminalError(fmt.Errorf("error downloading bundle '%s': %v", bundle.Name, err))
	}

	// Open a gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(content))
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
