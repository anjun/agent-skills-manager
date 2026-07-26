package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupBakLeftover(t *testing.T) {
	root := t.TempDir()
	skills := filepath.Join(root, "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	bak := filepath.Join(skills, "dws.bak-20260726-175612-6980")
	if err := os.MkdirAll(filepath.Join(bak, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	// real skill dir must stay
	keep := filepath.Join(skills, "dws")
	if err := os.MkdirAll(keep, 0o755); err != nil {
		t.Fatal(err)
	}

	res := Cleanup([]Issue{
		{Code: CodeBakLeftover, Path: bak},
		{Code: CodeBakLeftover, Path: keep}, // not bak name → fail
	})
	if len(res.Removed) != 1 || res.Removed[0] != bak {
		t.Fatalf("removed=%v", res.Removed)
	}
	if len(res.Failed) != 1 {
		t.Fatalf("failed=%v", res.Failed)
	}
	if _, err := os.Stat(bak); !os.IsNotExist(err) {
		t.Fatal("bak should be gone")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("keep should remain")
	}
}

func TestCleanupBrokenLink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "orphan")
	if err := os.Symlink(filepath.Join(root, "missing-target"), link); err != nil {
		t.Fatal(err)
	}
	// live link should not be removed
	liveTarget := filepath.Join(root, "exists")
	if err := os.WriteFile(liveTarget, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(root, "live")
	if err := os.Symlink(liveTarget, live); err != nil {
		t.Fatal(err)
	}

	res := Cleanup([]Issue{
		{Code: CodeBrokenLink, Path: link},
		{Code: CodeBrokenLink, Path: live},
	})
	if len(res.Removed) != 1 || res.Removed[0] != link {
		t.Fatalf("removed=%v failed=%v", res.Removed, res.Failed)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatal("broken link should be gone")
	}
	if _, err := os.Lstat(live); err != nil {
		t.Fatal("live link should remain")
	}
}
