package selfupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.2.0", "0.3.0", -1},
		{"0.3.0", "0.3.0", 0},
		{"v0.3.1", "0.3.0", 1},
		{"1.0.0", "0.9.9", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Fatalf("compare(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestRunCheckOnly(t *testing.T) {
	asset := expectedAssetName()
	body := map[string]any{
		"tag_name": "v0.9.0",
		"html_url": "https://github.com/anjun/agent-skills-manager/releases/tag/v0.9.0",
		"assets": []map[string]any{
			{
				"name":                 asset,
				"browser_download_url": "https://example.com/" + asset,
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
	if res.UpToDate || res.Latest != "0.9.0" {
		t.Fatalf("%+v", res)
	}
	if res.Asset != asset {
		t.Fatalf("asset=%s", res.Asset)
	}
}

func TestRunUpToDate(t *testing.T) {
	body := map[string]any{
		"tag_name": "v0.2.0",
		"html_url": "https://example.com",
		"assets":   []any{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	old := HTTPClient
	HTTPClient = &http.Client{Transport: rewriteAPI{base: srv.URL}}
	t.Cleanup(func() { HTTPClient = old })

	res, err := Run(context.Background(), Options{Current: "0.2.0", Repo: "o/r", CheckOnly: true, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpToDate {
		t.Fatalf("%+v", res)
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "asm")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "newbin")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceExecutable(dest, src); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "new" {
		t.Fatalf("got %q", b)
	}
}

func TestExpectedAssetName(t *testing.T) {
	got := expectedAssetName()
	want := "agent-skills-manager-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
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
