package selfupdate

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	if compareVersions("0.2.0", "0.3.0") != -1 {
		t.Fatal()
	}
	if compareVersions("v0.3.1", "0.3.0") != 1 {
		t.Fatal()
	}
}

func TestExpectedAssetName(t *testing.T) {
	if expectedAssetName() != MacOSZip {
		t.Fatalf("got %s", expectedAssetName())
	}
}

func TestRunCheckOnly(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("upgrade is macOS-only")
	}
	body := map[string]any{
		"tag_name": "v0.9.0",
		"html_url": "https://github.com/anjun/agent-skills-manager/releases/tag/v0.9.0",
		"assets": []map[string]any{
			{
				"name":                 MacOSZip,
				"browser_download_url": "https://example.com/" + MacOSZip,
				"url":                  "https://api.example.com/asset/1",
				"size":                 12,
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/latest") {
			_ = json.NewEncoder(w).Encode(body)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	old := HTTPClient
	HTTPClient = &http.Client{Transport: rewriteAPI{base: srv.URL}}
	t.Cleanup(func() { HTTPClient = old })

	res, err := Run(context.Background(), Options{
		Current:   "0.2.0",
		Repo:      "anjun/agent-skills-manager",
		CheckOnly: true,
		Token:     "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UpToDate || res.Latest != "0.9.0" || res.Asset != MacOSZip {
		t.Fatalf("%+v", res)
	}
}

func TestExtractFromZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "asm-macos.zip")
	if err := writeTestZip(zipPath, map[string]string{
		"asm-arm64": "arm-bin",
		"asm-amd64": "amd-bin",
	}); err != nil {
		t.Fatal(err)
	}
	// force prefer name
	got, err := extractFromZip(zipPath, "asm-arm64", "asm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(got)
	b, _ := os.ReadFile(got)
	if string(b) != "arm-bin" {
		t.Fatalf("got %q", b)
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "asm")
	_ = os.WriteFile(dest, []byte("old"), 0o755)
	src := filepath.Join(dir, "newbin")
	_ = os.WriteFile(src, []byte("new"), 0o755)
	if err := replaceExecutable(dest, src); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "new" {
		t.Fatalf("%q", b)
	}
}

func writeTestZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			return err
		}
	}
	return w.Close()
}

type rewriteAPI struct{ base string }

func (r rewriteAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	u := r.base + req.URL.Path
	if req.URL.RawQuery != "" {
		u += "?" + req.URL.RawQuery
	}
	nr, err := http.NewRequestWithContext(req.Context(), req.Method, u, req.Body)
	if err != nil {
		return nil, err
	}
	nr.Header = req.Header.Clone()
	return http.DefaultTransport.RoundTrip(nr)
}

func TestDownloadOutlastsMetadataTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "first")
		w.(http.Flusher).Flush()
		select {
		case <-time.After(300 * time.Millisecond):
			fmt.Fprint(w, "last")
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	old := HTTPClient
	HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { HTTPClient = old }()
	path, err := downloadAsset(context.Background(), ghAsset{Name: "slow", URL: srv.URL, Size: 9}, "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "firstlast" {
		t.Fatalf("%q %v", data, err)
	}
}

func TestDownloadCancellationAndIncompleteBody(t *testing.T) {
	for _, cancelRequest := range []bool{true, false} {
		t.Run(fmt.Sprint(cancelRequest), func(t *testing.T) {
			temp := t.TempDir()
			t.Setenv("TMPDIR", temp)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "part")
				w.(http.Flusher).Flush()
				if cancelRequest {
					<-r.Context().Done()
				}
			}))
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			path, err := downloadAsset(ctx, ghAsset{Name: "test", URL: srv.URL, Size: 100}, "")
			if err == nil || path != "" {
				t.Fatalf("expected incomplete download failure: %q %v", path, err)
			}
			if cancelRequest && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("lost cancellation: %v", err)
			}
			files, err := os.ReadDir(temp)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) > 0 {
				t.Fatalf("partial files leaked: %v", files)
			}
		})
	}
}
