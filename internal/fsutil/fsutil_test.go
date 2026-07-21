package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitWorkTreeRoot(t *testing.T) {
	tmp := t.TempDir()
	// isolate from real home: use a nested dir under tmp as "path"
	repo := filepath.Join(tmp, "repo")
	sub := filepath.Join(repo, "pkg", "x")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, ok := GitWorkTreeRoot(sub)
	if !ok {
		t.Fatal("expected git root")
	}
	if root != repo {
		t.Fatalf("got %s want %s", root, repo)
	}
}

func TestRelHome(t *testing.T) {
	home := Home()
	if home == "" {
		t.Skip("no home")
	}
	got := RelHome(filepath.Join(home, ".agents", "skills"))
	if got != "~/.agents/skills" {
		t.Fatalf("got %q", got)
	}
}
