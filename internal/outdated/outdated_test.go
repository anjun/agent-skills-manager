package outdated

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/skilllock"
)

func TestCheckOneUnknown(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := scan.Entry{Name: "demo", Path: skillDir, Class: scan.ClassUser, Tool: "claude"}
	got := checkOne(e, skillDir, &skilllock.Index{ByName: map[string]skilllock.Entry{}}, skilllock.NewTreeCache())
	if got.Status != StatusUnknown {
		t.Fatalf("status=%s msg=%s", got.Status, got.Message)
	}
}

func TestCheckOneGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@t")
	run("config", "user.name", "t")
	// skill at repo root
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: gitskill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	// bare remote
	bare := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v %s", err, out)
	}
	run("remote", "add", "origin", bare)
	// push current branch explicitly
	brOut, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	br := strings.TrimSpace(string(brOut))
	if br == "" {
		br = "master"
		run("branch", "-M", br)
	}
	run("push", "-u", "origin", br)

	e := scan.Entry{Name: "gitskill", Path: dir, Class: scan.ClassUser, Tool: "extra"}
	got := checkOne(e, dir, nil, nil)
	if got.Provenance != ProvGit {
		t.Fatalf("want git provenance, got %s (%s)", got.Provenance, got.Message)
	}
	if got.Status != StatusCurrent && got.Status != StatusOutdated {
		t.Fatalf("status=%s msg=%s local=%s remote=%s", got.Status, got.Message, got.LocalRef, got.RemoteRef)
	}
}

func TestCheckOneSkillLockCurrent(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "grill-me")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: grill-me\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	localHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /repos/owner/repo/git/trees/HEAD?recursive=1
		if !strings.Contains(r.URL.Path, "/git/trees/") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sha": "rootsha",
			"tree": []map[string]string{
				{"path": "skills/productivity/grill-me", "type": "tree", "sha": localHash},
				{"path": "skills/productivity/grill-me/SKILL.md", "type": "blob", "sha": "blob1"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	// Point GitHub API at test server via HTTPClient transport rewrite is hard;
	// instead unit-test checkSkillLockGitHub through a custom TreeCache that we
	// pre-seed... TreeCache hits real URL. Override by testing checkSkillLockGitHub
	// with a fake via package-level HTTPClient.Replace — skilllock uses full
	// https://api.github.com URL. Use roundTripper rewrite:
	old := skilllock.HTTPClient
	skilllock.HTTPClient = &http.Client{Transport: rewriteHost{base: srv.URL, next: http.DefaultTransport}}
	t.Cleanup(func() { skilllock.HTTPClient = old })

	le := skilllock.Entry{
		Name:            "grill-me",
		Source:          "mattpocock/skills",
		SourceType:      "github",
		SourceURL:       "https://github.com/mattpocock/skills.git",
		SkillPath:       "skills/productivity/grill-me/SKILL.md",
		SkillFolderHash: localHash,
		LockPath:        "/tmp/.skill-lock.json",
	}
	idx := &skilllock.Index{ByName: map[string]skilllock.Entry{"grill-me": le}}
	e := scan.Entry{Name: "grill-me", Path: skillDir, Class: scan.ClassCanonical, Tool: "agents"}
	got := checkOne(e, skillDir, idx, skilllock.NewTreeCache())
	if got.Provenance != ProvSkillLock {
		t.Fatalf("prov=%s msg=%s", got.Provenance, got.Message)
	}
	if got.Status != StatusCurrent {
		t.Fatalf("status=%s local=%s remote=%s msg=%s", got.Status, got.LocalRef, got.RemoteRef, got.Message)
	}
}

func TestCheckOneSkillLockOutdated(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: code-review\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	localHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	remoteHash := "cccccccccccccccccccccccccccccccccccccccc"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sha": "rootsha",
			"tree": []map[string]string{
				{"path": "skills/engineering/code-review", "type": "tree", "sha": remoteHash},
			},
		})
	}))
	t.Cleanup(srv.Close)

	old := skilllock.HTTPClient
	skilllock.HTTPClient = &http.Client{Transport: rewriteHost{base: srv.URL, next: http.DefaultTransport}}
	t.Cleanup(func() { skilllock.HTTPClient = old })

	le := skilllock.Entry{
		Name:            "code-review",
		Source:          "mattpocock/skills",
		SourceType:      "github",
		SkillPath:       "skills/engineering/code-review/SKILL.md",
		SkillFolderHash: localHash,
		LockPath:        "/tmp/.skill-lock.json",
	}
	idx := &skilllock.Index{ByName: map[string]skilllock.Entry{"code-review": le}}
	e := scan.Entry{Name: "code-review", Path: skillDir, Class: scan.ClassCanonical, Tool: "agents"}
	got := checkOne(e, skillDir, idx, skilllock.NewTreeCache())
	if got.Status != StatusOutdated {
		t.Fatalf("status=%s msg=%s", got.Status, got.Message)
	}
	if got.LocalRef != shortSHA(localHash) || got.RemoteRef != shortSHA(remoteHash) {
		t.Fatalf("local=%s remote=%s", got.LocalRef, got.RemoteRef)
	}
}

// rewriteHost sends api.github.com requests to the test server base URL.
type rewriteHost struct {
	base string
	next http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	next := r.next
	if next == nil {
		next = http.DefaultTransport
	}
	if strings.Contains(req.URL.Host, "api.github.com") {
		u := r.base + req.URL.Path
		if req.URL.RawQuery != "" {
			u += "?" + req.URL.RawQuery
		}
		nr, err := http.NewRequestWithContext(req.Context(), req.Method, u, req.Body)
		if err != nil {
			return nil, err
		}
		nr.Header = req.Header.Clone()
		return next.RoundTrip(nr)
	}
	return next.RoundTrip(req)
}
