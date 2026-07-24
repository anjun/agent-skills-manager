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

func TestSkillLinkStateOf(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "agents", "demo")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	tools := filepath.Join(tmp, "claude", "skills")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(tools, "missing")
	if SkillLinkStateOf(missing, target) != LinkMissing {
		t.Fatal("expected missing")
	}
	ok := filepath.Join(tools, "ok")
	if err := os.Symlink(target, ok); err != nil {
		t.Fatal(err)
	}
	if SkillLinkStateOf(ok, target) != LinkOK {
		t.Fatal("expected ok symlink")
	}
	wrong := filepath.Join(tools, "wrong")
	if err := os.MkdirAll(wrong, 0o755); err != nil {
		t.Fatal(err)
	}
	if SkillLinkStateOf(wrong, target) != LinkWrong {
		t.Fatal("expected wrong real dir")
	}
}

func TestIsASMBackupDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"dws.bak-20260721-161027-7b88", true},
		{"help.bak-20260101-000000-abcd", true},
		{"dws.bak-20260721-161027-7B88", true},
		{"dws", false},
		{"dws.bak", false},
		{"dws.bak-short", false},
		{"my.bak-skill", false},
		{".bak-20260721-161027-7b88", false}, // no name prefix
		{"dws.bak-20260721161027-7b88", false}, // missing time dash
		{"/tmp/x/dws.bak-20260721-161027-7b88", true},
	}
	for _, c := range cases {
		if got := IsASMBackupDir(c.name); got != c.want {
			t.Errorf("IsASMBackupDir(%q)=%v want %v", c.name, got, c.want)
		}
	}
}
