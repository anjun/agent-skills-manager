// Package selfupdate upgrades the asm binary from GitHub Releases.
package selfupdate

import (
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
	// AssetPrefix is the release asset name prefix.
	AssetPrefix = "agent-skills-manager"
	userAgent   = "agent-skills-manager"
	httpTimeout = 60 * time.Second
)

// HTTPClient is overridable in tests.
var HTTPClient = &http.Client{Timeout: httpTimeout}

// Options for Check / Apply.
type Options struct {
	// Current is the running version (e.g. "0.3.0" or "v0.3.0").
	Current string
	// Repo is "owner/repo". Empty → DefaultRepo or ASM_GITHUB_REPO.
	Repo string
	// Token is a GitHub token. Empty → env / gh auth token.
	Token string
	// CheckOnly skips download/replace.
	CheckOnly bool
	// Yes skips confirmation (caller decides).
	Yes bool
	// Executable is the path to replace. Empty → os.Executable().
	Executable string
}

// Result of a check or upgrade.
type Result struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	Repo        string `json:"repo"`
	Asset       string `json:"asset,omitempty"`
	URL         string `json:"url,omitempty"`
	UpToDate    bool   `json:"up_to_date"`
	Updated     bool   `json:"updated"`
	Message     string `json:"message"`
	ReleaseURL  string `json:"release_url,omitempty"`
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
	URL                string `json:"url"` // API URL for private download
	Size               int64  `json:"size"`
}

// Run checks (and optionally applies) a self-update from GitHub Releases.
func Run(ctx context.Context, opt Options) (*Result, error) {
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

	assetName := expectedAssetName()
	asset, ok := findAsset(rel.Assets, assetName)
	if !ok {
		return nil, fmt.Errorf("release %s has no asset %q (os=%s arch=%s)", rel.TagName, assetName, runtime.GOOS, runtime.GOARCH)
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

	tmp, err := downloadAsset(ctx, asset, token)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	if err := replaceExecutable(exe, tmp); err != nil {
		return nil, err
	}
	res.Updated = true
	res.UpToDate = false
	res.Message = fmt.Sprintf("updated %s → %s (%s)", current, latest, exe)
	return res, nil
}

func expectedAssetName() string {
	name := fmt.Sprintf("%s-%s-%s", AssetPrefix, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func findAsset(assets []ghAsset, want string) (ghAsset, bool) {
	wantLower := strings.ToLower(want)
	for _, a := range assets {
		if strings.EqualFold(a.Name, want) {
			return a, true
		}
	}
	// fuzzy: contains os-arch
	needle := strings.ToLower(runtime.GOOS + "-" + runtime.GOARCH)
	for _, a := range assets {
		n := strings.ToLower(a.Name)
		if strings.Contains(n, needle) && strings.Contains(n, "agent-skills-manager") {
			return a, true
		}
		_ = wantLower
	}
	return ghAsset{}, false
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
	// Prefer API asset URL (works for private repos with token).
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
	// For API asset downloads GitHub requires this Accept.
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
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func replaceExecutable(dest, src string) error {
	// write next to dest then rename
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
	// move current → bak, next → dest
	if err := os.Rename(dest, bak); err != nil {
		// maybe dest doesn't exist yet
		if !os.IsNotExist(err) {
			os.Remove(next)
			return fmt.Errorf("backup current binary: %w", err)
		}
	}
	if err := os.Rename(next, dest); err != nil {
		// try restore
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

// normalizeVersion strips leading v and whitespace.
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

// compareVersions returns -1 if a<b, 0 if equal, 1 if a>b (simple semver-ish).
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
