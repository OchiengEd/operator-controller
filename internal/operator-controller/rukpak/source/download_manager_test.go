package source

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func Test_chartDownloader(t *testing.T) {
	testcases := []struct {
		name    string
		url     string
		wantMd5 string
		wantErr bool
	}{
		{
			name:    "Download chart from OCI registry",
			url:     "oci://localhost:5000/charts/metrics-server:3.12.0",
			wantErr: false,
			wantMd5: "9e68a30ac986aab2aa7ad20187125e44",
		},
		{
			name:    "Download chart from HTTP registry",
			url:     "https://github.com/kubernetes-sigs/metrics-server/releases/download/metrics-server-helm-chart-3.12.0/metrics-server-3.12.0.tgz",
			wantErr: false,
			wantMd5: "9e68a30ac986aab2aa7ad20187125e44",
		},
	}

	ctx := context.Background()
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := httpClientWithCA(context.Background(), "olmv1-cert", "olmv1-system")
			downloader := newDownloadManager(tc.url)
			got, err := downloader.Download(ctx, client)
			if (err != nil) != tc.wantErr {
				t.Errorf("error downloading chart; %v", err)
				return
			}
			hash := md5.Sum(got)
			checksum := hex.EncodeToString(hash[:])
			fileName := filepath.Base(tc.url)
			if strings.Contains(fileName, ":") {
				s := strings.Split(fileName, ":")
				fileName = fmt.Sprintf("%s-%s.tgz", s[0], s[1])
			}

			if checksum != tc.wantMd5 {
				t.Errorf("The md5.Sum() = %s is %s but, want = %s",
					fileName,
					checksum,
					tc.wantMd5,
				)
			}
		})
	}
}

func Test_chartNameConverter(t *testing.T) {
	testtable := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "Helm accepted OCI chart name",
			image: "oci://localhost:5000/metrics-server-3.12.0.tgz",
			want:  "oci://localhost:5000/metrics-server:3.12.0",
		},
		{
			name:  "Helm with tagged chart name",
			image: "oci://localhost:5000/metrics-server:3.12.0",
			want:  "oci://localhost:5000/metrics-server:3.12.0",
		},
	}

	for _, tc := range testtable {
		t.Run(tc.name, func(t *testing.T) {
			got := chartNameConverter(tc.image)
			if got != tc.want {
				t.Errorf("chartNameConverter() should return %s but, we got %s\n", tc.want, got)
			}
		})
	}
}
