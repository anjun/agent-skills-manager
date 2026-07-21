package skillgate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
)

func testCfg(t *testing.T) *config.Config {
	t.Helper()
	home := t.TempDir()
	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = filepath.Join(home, ".agents", "skills")
	cfg.StateDir = filepath.Join(home, ".local", "state", config.AppName)
	cfg.ConfigDir = filepath.Join(home, ".config", config.AppName)
	if err := fsutil.EnsureDir(cfg.AgentsSkills); err != nil {
		t.Fatal(err)
	}
	// wire roots
	for _, rel := range []string{".claude/skills", ".cursor/skills", ".grok/skills", ".codex/skills", ".gemini/skills"} {
		if err := fsutil.EnsureDir(filepath.Join(home, rel)); err != nil {
			t.Fatal(err)
		}
	}
	return cfg
}

func writeSkill(t *testing.T, dir string) {
	t.Helper()
	if err := fsutil.EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\n---\n# demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDisableEnableTool(t *testing.T) {
	cfg := testCfg(t)
	name := "demo"
	agents := filepath.Join(cfg.AgentsSkills, name)
	writeSkill(t, agents)
	// pre-wire codex + claude
	for _, tool := range []string{"codex", "claude"} {
		link := filepath.Join(cfg.Home, "."+tool, "skills", name)
		if err := fsutil.SymlinkRel(agents, link); err != nil {
			t.Fatal(err)
		}
	}

	s, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Disable(name, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Already || len(res.Removed) == 0 {
		t.Fatalf("%+v", res)
	}
	if !s.IsToolOff(name, "codex") {
		t.Fatal("codex should be off")
	}
	if s.IsToolOff(name, "claude") {
		t.Fatal("claude should still be on")
	}
	if _, err := os.Lstat(filepath.Join(cfg.Home, ".codex", "skills", name)); !os.IsNotExist(err) {
		t.Fatalf("codex link should be gone: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(cfg.Home, ".claude", "skills", name)); err != nil {
		t.Fatal("claude link should remain")
	}
	// agents content stays
	if !existsDir(agents) {
		t.Fatal("agents content must stay for per-tool disable")
	}

	res2, err := s.Enable(name, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Linked) == 0 {
		t.Fatalf("%+v", res2)
	}
	if s.IsToolOff(name, "codex") {
		t.Fatal("codex should be on")
	}
	if _, err := os.Lstat(filepath.Join(cfg.Home, ".codex", "skills", name)); err != nil {
		t.Fatal("codex link restored")
	}
}

func TestDisableEnableGlobal(t *testing.T) {
	cfg := testCfg(t)
	name := "gskill"
	agents := filepath.Join(cfg.AgentsSkills, name)
	writeSkill(t, agents)
	link := filepath.Join(cfg.Home, ".codex", "skills", name)
	if err := fsutil.SymlinkRel(agents, link); err != nil {
		t.Fatal(err)
	}

	s, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Disable(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.GlobalOff || res.Parked == "" {
		t.Fatalf("%+v", res)
	}
	if existsDir(agents) {
		t.Fatal("agents path should be parked")
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatal("tool link should be removed")
	}
	if !s.IsGlobalOff(name) {
		t.Fatal("global off")
	}

	res2, err := s.Enable(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Restored == "" || len(res2.Linked) == 0 {
		t.Fatalf("%+v", res2)
	}
	if !existsDir(agents) {
		t.Fatal("agents restored")
	}
	if s.IsGlobalOff(name) {
		t.Fatal("should be on")
	}
}

func TestUnknownTool(t *testing.T) {
	cfg := testCfg(t)
	s, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Disable("x", []string{"nope"}); err == nil {
		t.Fatal("want error")
	}
}

func TestPlanRespectsToolOff(t *testing.T) {
	// integration-ish: store gate then IsToolOff
	cfg := testCfg(t)
	writeSkill(t, filepath.Join(cfg.AgentsSkills, "a"))
	s, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Disable("a", []string{"grok"}); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.IsToolOff("a", "grok") || s2.IsToolOff("a", "claude") {
		t.Fatal(s2.Get("a"))
	}
}
