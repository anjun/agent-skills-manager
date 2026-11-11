package remove

import (
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"io"
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
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "xdg-state"))
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

// Orphan skill-lock entries (doctor U2) must be removable even when the skill dir is gone.
func TestRunCleansOrphanSkillLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "xdg-state"))

	agents := filepath.Join(home, ".agents", "skills")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(home, ".agents", ".skill-lock.json")
	raw := `{"version":3,"skills":{"ui-ux-pro-max":{"source":"x/y","sourceType":"github","skillPath":"a/SKILL.md","skillFolderHash":"abc"},"keep-me":{"source":"a/b","sourceType":"github","skillPath":"b/SKILL.md","skillFolderHash":"def"}}}`
	if err := os.WriteFile(lock, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = agents
	vl, _ := vendorlist.Load(nil, cfg)

	// Plan should mark lock-only, not "not found"
	plan, err := Plan(cfg, vl, []string{"ui-ux-pro-max"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Planned) != 0 {
		t.Fatalf("expected no disk paths, planned=%+v", plan.Planned)
	}
	if len(plan.LockOnly) != 1 || plan.LockOnly[0] != "ui-ux-pro-max" {
		t.Fatalf("LockOnly=%v skipped=%v", plan.LockOnly, plan.Skipped)
	}

	var buf strings.Builder
	res, err := Run(cfg, vl, Options{
		Names:  []string{"ui-ux-pro-max"},
		Yes:    true,
		Stdout: &buf,
		Stdin:  strings.NewReader(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.LockModified) == 0 {
		t.Fatalf("expected lock modified, out=%s res=%+v", buf.String(), res)
	}
	b, _ := os.ReadFile(lock)
	if strings.Contains(string(b), `"ui-ux-pro-max"`) {
		t.Fatalf("orphan still in lock: %s", b)
	}
	if !strings.Contains(string(b), `"keep-me"`) {
		t.Fatalf("unrelated lock entry removed: %s", b)
	}
	if !strings.Contains(buf.String(), "skill-lock") {
		t.Fatalf("expected lock-cleanup message, out=%s", buf.String())
	}
}

func TestRunOrphanKeepLockDoesNotTouchLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "xdg-state"))

	agents := filepath.Join(home, ".agents", "skills")
	_ = os.MkdirAll(agents, 0o755)
	lock := filepath.Join(home, ".agents", ".skill-lock.json")
	raw := `{"version":3,"skills":{"ghost":{"source":"x/y","sourceType":"github","skillPath":"a/SKILL.md","skillFolderHash":"abc"}}}`
	if err := os.WriteFile(lock, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = agents
	vl, _ := vendorlist.Load(nil, cfg)

	var buf strings.Builder
	res, err := Run(cfg, vl, Options{
		Names:    []string{"ghost"},
		Yes:      true,
		KeepLock: true,
		Stdout:   &buf,
		Stdin:    strings.NewReader(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.LockModified) != 0 {
		t.Fatalf("keep-lock should not modify: %+v", res)
	}
	b, _ := os.ReadFile(lock)
	if !strings.Contains(string(b), `"ghost"`) {
		t.Fatalf("lock was modified with --keep-lock: %s", b)
	}
}

func TestRemovePreservesOtherGatesAndContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	cfg.NoProject = true
	cfg.NoProbe = true
	vl, err := vendorlist.Load(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	gates, err := skillgate.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"demo", "keep-me"} {
		p := filepath.Join(cfg.AgentsSkills, name)
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := gates.Disable(name, nil); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Run(cfg, vl, Options{Names: []string{"demo"}, Yes: true, KeepLock: true, Stdout: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Failed) > 0 {
		t.Fatal(res.Failed)
	}
	gates, err = skillgate.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if gates.IsGlobalOff("demo") || !gates.IsGlobalOff("keep-me") {
		t.Fatalf("wrong remaining gates: %v", gates.List())
	}
	if _, err := os.Stat(filepath.Join(gates.DisabledDir, "keep-me", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}
