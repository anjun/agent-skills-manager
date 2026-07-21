package remove

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

func TestRunRemovesCanonicalAndSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "xdg-state"))

	agents := filepath.Join(home, ".agents", "skills")
	claude := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(agents, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "demo", "SKILL.md"), []byte("---\nname: demo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}
	// symlink claude/demo -> agents/demo
	if err := os.Symlink(filepath.Join(agents, "demo"), filepath.Join(claude, "demo")); err != nil {
		t.Fatal(err)
	}
	// lock file
	lockDir := filepath.Join(home, ".agents")
	lock := filepath.Join(lockDir, ".skill-lock.json")
	if err := os.WriteFile(lock, []byte(`{"version":3,"skills":{"demo":{"source":"x/y","sourceType":"github","skillPath":"a/SKILL.md","skillFolderHash":"abc"},"other":{"source":"a/b","sourceType":"github","skillPath":"b/SKILL.md","skillFolderHash":"def"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = agents
	cfg.Yes = true
	// only scan agents + claude
	cfg.OnlyTools = []string{"claude"}

	vl, _ := vendorlist.Load(nil, cfg)
	var buf strings.Builder
	res, err := Run(cfg, vl, Options{
		Names:  []string{"demo"},
		Yes:    true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) < 1 {
		t.Fatalf("removed=%v planned=%v", res.Removed, res.Planned)
	}
	if _, err := os.Lstat(filepath.Join(agents, "demo")); !os.IsNotExist(err) {
		t.Fatalf("canonical still exists: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(claude, "demo")); !os.IsNotExist(err) {
		t.Fatalf("symlink still exists: %v", err)
	}
	b, _ := os.ReadFile(lock)
	if strings.Contains(string(b), `"demo"`) {
		t.Fatalf("lock still has demo: %s", b)
	}
	if !strings.Contains(string(b), `"other"`) {
		t.Fatalf("lock lost other: %s", b)
	}
	if len(res.LockModified) == 0 {
		t.Fatalf("expected lock modified, buf=%s", buf.String())
	}
}

func TestPlanSkipsVendor(t *testing.T) {
	// minimal: empty scan with no paths — name not found
	home := t.TempDir()
	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = filepath.Join(home, ".agents", "skills")
	_ = os.MkdirAll(cfg.AgentsSkills, 0o755)
	vl, _ := vendorlist.Load(nil, cfg)
	res, err := Plan(cfg, vl, []string{"nope"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Planned) != 0 {
		t.Fatalf("%+v", res)
	}
}
