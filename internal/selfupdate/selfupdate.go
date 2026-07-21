// Package selfupdate upgrades the asm binary from GitHub Releases.
package selfupdate

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultRepo is the GitHub owner/repo for releases.
	DefaultRepo = "anjun/agent-skills-manager"
	// MacOSZip is the unified macOS installer package (arm64 + amd64).
	MacOSZip = "asm-macos.zip"
	userAgent   = "agent-skills-manager"
	httpTimeout = 60 * time.Second
)

// HTTPClient is overridable in tests.
var HTTPClient = &http.Client{Timeout: httpTimeout}

// Options for Check / Apply.
type Options struct {
	Current    string
	Repo       string
	Token      string
	CheckOnly  bool
	Yes        bool
	Executable string
}

// Result of a check or upgrade.
type Result struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Repo       string `json:"repo"`
	Asset      string `json:"asset,omitempty"`
	URL        string `json:"url,omitempty"`
	UpToDate   bool   `json:"up_to_date"`
	Updated    bool   `json:"updated"`
	Message    string `json:"message"`
	ReleaseURL string `json:"release_url,omitempty"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	HTMLURL string    `json:"html_url"`
	Assets  []ghAsset `json:"assets"`
	Draft   bool      `json:"draft"`
	Pre     bool      `json:"prerelease"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	URL                string `json:"url"`
	Size               int64  `json:"size"`
}

// Run checks (and optionally applies) a self-update from GitHub Releases.
func Run(ctx context.Context, opt Options) (*Result, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("asm upgrade currently supports macOS only (got %s)", runtime.GOOS)
	}

	repo := strings.TrimSpace(opt.Repo)
	if repo == "" {
		repo = strings.TrimSpace(os.Getenv("ASM_GITHUB_REPO"))
	}
	if repo == "" {
		repo = DefaultRepo
	}
	token := strings.TrimSpace(opt.Token)
	if token == "" {
		token = githubToken()
	}

	rel, err := fetchLatestRelease(ctx, repo, token)
	if err != nil {
		return nil, err
	}
	latest := normalizeVersion(rel.TagName)
	current := normalizeVersion(opt.Current)
	res := &Result{
		Current:    current,
		Latest:     latest,
		Repo:       repo,
		ReleaseURL: rel.HTMLURL,
	}

	if compareVersions(current, latest) >= 0 {
		res.UpToDate = true
		res.Message = fmt.Sprintf("already on latest %s", latest)
		return res, nil
	}

	asset, ok := findAsset(rel.Assets, candidateAssetNames()...)
	if !ok {
		return nil, fmt.Errorf("release %s has no macOS package (want %s)", rel.TagName, MacOSZip)
	}
	res.Asset = asset.Name
	res.URL = asset.BrowserDownloadURL
	res.Message = fmt.Sprintf("%s → %s available", current, latest)

	if opt.CheckOnly {
		return res, nil
	}

	exe := opt.Executable
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable: %w", err)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return nil, fmt.Errorf("resolve executable symlink: %w", err)
		}
	}

	dl, err := downloadAsset(ctx, asset, token)
	if err != nil {
		return nil, err
	}
	defer os.Remove(dl)

	bin, err := materializeBinary(dl, asset.Name)
	if err != nil {
		return nil, err
	}
	if bin != dl {
		defer os.Remove(bin)
	}

	if err := replaceExecutable(exe, bin); err != nil {
		return nil, err
	}
	res.Updated = true
	res.UpToDate = false
	res.Message = fmt.Sprintf("updated %s → %s (%s)", current, latest, exe)
	return res, nil
}

// candidateAssetNames preferred first.
func candidateAssetNames() []string {
	return []string{
		MacOSZip,
		// legacy names from older releases
		"agent-skills-manager-darwin-arm64.tar.gz",
		"agent-skills-manager-darwin-amd64.tar.gz",
		"agent-skills-manager-darwin-arm64",
		"agent-skills-manager-darwin-amd64",
	}
}

// expectedAssetName is the primary asset (tests / display).
func expectedAssetName() string {
	return MacOSZip
}

func findAsset(assets []ghAsset, names ...string) (ghAsset, bool) {
	for _, want := range names {
		for _, a := range assets {
			if strings.EqualFold(a.Name, want) {
				// for legacy per-arch assets, skip wrong arch
				if strings.Contains(a.Name, "arm64") && runtime.GOARCH != "arm64" {
					continue
				}
				if strings.Contains(a.Name, "amd64") && runtime.GOARCH != "amd64" {
					continue
				}
				return a, true
			}
		}
	}
	// fuzzy: any asm-macos / agent-skills-manager zip for mac
	for _, a := range assets {
		n := strings.ToLower(a.Name)
		if strings.Contains(n, "asm-macos") && strings.HasSuffix(n, ".zip") {
			return a, true
		}
	}
	return ghAsset{}, false
}

func archBinaryName() string {
	switch runtime.GOARCH {
	case "arm64":
		return "asm-arm64"
	case "amd64":
		return "asm-amd64"
	default:
		return "asm"
	}
}

// materializeBinary turns a downloaded asset into a raw executable path.
func materializeBinary(path, assetName string) (string, error) {
	lower := strings.ToLower(assetName)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractFromZip(path, archBinaryName(), "asm")
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		// legacy: prefer tar via system tar for simplicity
		return extractTarGzAsm(path)
	default:
		// bare binary
		_ = os.Chmod(path, 0o755)
		return path, nil
	}
}

func extractFromZip(zipPath string, preferNames ...string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var match *zip.File
	for _, name := range preferNames {
		for _, f := range r.File {
			base := filepath.Base(f.Name)
			if base == name || f.Name == name {
				match = f
				break
			}
		}
		if match != nil {
			break
		}
	}
	if match == nil {
		// any file named asm*
		for _, f := range r.File {
			base := filepath.Base(f.Name)
			if base == "asm" || strings.HasPrefix(base, "asm-") {
				match = f
				break
			}
		}
	}
	if match == nil {
		return "", fmt.Errorf("zip %s has no asm binary for this Mac", filepath.Base(zipPath))
	}

	rc, err := match.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	out, err := os.CreateTemp("", "asm-bin-*")
	if err != nil {
		return "", err
	}
	outPath := out.Name()
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		os.Remove(outPath)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(outPath)
		return "", err
	}
	if err := os.Chmod(outPath, 0o755); err != nil {
		os.Remove(outPath)
		return "", err
	}
	return outPath, nil
}

func extractTarGzAsm(path string) (string, error) {
	dir, err := os.MkdirTemp("", "asm-tar-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	cmd := exec.Command("tar", "-xzf", path, "-C", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar extract: %w (%s)", err, truncate(string(out), 120))
	}
	// find asm
	var found string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		if base == "asm" || base == archBinaryName() {
			found = p
			return io.EOF
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("tar has no asm binary")
	}
	out, err := os.CreateTemp("", "asm-bin-*")
	if err != nil {
		return "", err
	}
	outPath := out.Name()
	out.Close()
	b, err := os.ReadFile(found)
	if err != nil {
		os.Remove(outPath)
		return "", err
	}
	if err := os.WriteFile(outPath, b, 0o755); err != nil {
		os.Remove(outPath)
		return "", err
	}
	return outPath, nil
}

func fetchLatestRelease(ctx context.Context, repo, token string) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req, token, false)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s (create a GitHub Release first)", repo)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		hint := "set GITHUB_TOKEN/GH_TOKEN or run: gh auth login"
		return nil, fmt.Errorf("GitHub API HTTP %d for %s: %s (%s)", resp.StatusCode, repo, truncate(string(body), 160), hint)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("empty tag_name in latest release")
	}
	return &rel, nil
}

func downloadAsset(ctx context.Context, asset ghAsset, token string) (tmpPath string, err error) {
	url := asset.URL
	if url == "" {
		url = asset.BrowserDownloadURL
	}
	if url == "" {
		return "", fmt.Errorf("asset %s has no download URL", asset.Name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setGitHubHeaders(req, token, true)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("download asset HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	f, err := os.CreateTemp("", "asm-upgrade-*")
	if err != nil {
		return "", err
	}
	tmpPath = f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func replaceExecutable(dest, src string) error {
	next := dest + ".new"
	bak := dest + ".old"
	_ = os.Remove(next)
	_ = os.Remove(bak)

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(next, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("write %s: %w (try sudo or install to a writable path)", next, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(next)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(next)
		return err
	}
	if err := os.Rename(dest, bak); err != nil {
		if !os.IsNotExist(err) {
			os.Remove(next)
			return fmt.Errorf("backup current binary: %w", err)
		}
	}
	if err := os.Rename(next, dest); err != nil {
		_ = os.Rename(bak, dest)
		os.Remove(next)
		return fmt.Errorf("install new binary: %w", err)
	}
	_ = os.Remove(bak)
	return nil
}

func setGitHubHeaders(req *http.Request, token string, assetDownload bool) {
	req.Header.Set("User-Agent", userAgent)
	if assetDownload {
		req.Header.Set("Accept", "application/octet-stream")
	} else {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func githubToken() string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("GH_TOKEN")); t != "" {
		return t
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func compareVersions(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			ai = atoiPrefix(as[i])
		}
		if i < len(bs) {
			bi = atoiPrefix(bs[i])
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

func atoiPrefix(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
