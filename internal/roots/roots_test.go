package roots

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
)

func TestProjectRootsFromExplicitRepo(t *testing.T) {
	tmp := t.TempDir()
	// fake repo with .git and project skills
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(tmp, ".agents", "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: demo\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = t.TempDir()
	cfg.NoProject = true
	cfg.ProjectRoots = []string{tmp}

	rs := ProjectRoots(cfg)
	found := false
	want := filepath.Join(tmp, ".agents", "skills")
	for _, r := range rs {
		if r.Kind != "project" {
			t.Fatalf("expected project kind, got %s", r.Kind)
		}
		if r.Path == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected skill root %s in %v", want, rs)
	}
}

func TestLooksLikeSkillRoot(t *testing.T) {
	for _, p := range []string{
		"/repo/.agents/skills",
		"/repo/.agent/skills",
		"/repo/_agents/skills",
		"/repo/_agent/skills",
		"/repo/skills",
	} {
		if !looksLikeSkillRoot(p) {
			t.Fatalf("expected %s to match", p)
		}
	}
	if looksLikeSkillRoot("/repo") {
		t.Fatal("repo root should not look like skill root")
	}
}

func TestAntigravityWireRootIndependentOfGemini(t *testing.T) {
	home := t.TempDir()
	cfg := config.Default()
	cfg.Home = home
	cfg.AgentsSkills = filepath.Join(home, ".agents", "skills")
	cfg.NoProject = true

	var gotAntigravity, gotGemini *Root
	for _, r := range WireRoots(cfg) {
		rc := r
		switch r.Tool {
		case "antigravity":
			gotAntigravity = &rc
		case "gemini":
			gotGemini = &rc
		}
	}
	wantAntigravity := filepath.Join(home, ".gemini", "config", "skills")
	wantGemini := filepath.Join(home, ".gemini", "skills")
	if gotAntigravity == nil || gotAntigravity.Path != wantAntigravity || !gotAntigravity.Wire {
		t.Fatalf("antigravity wire root mismatch: %+v", gotAntigravity)
	}
	if gotGemini == nil || gotGemini.Path != wantGemini || !gotGemini.Wire {
		t.Fatalf("gemini wire root mismatch: %+v", gotGemini)
	}

	// Excluding gemini must keep antigravity enabled, and vice versa.
	cfg.ExcludeTools = []string{"gemini"}
	var hasAG, hasGem bool
	for _, r := range WireRoots(cfg) {
		if r.Tool == "antigravity" {
			hasAG = true
		}
		if r.Tool == "gemini" {
			hasGem = true
		}
	}
	if !hasAG || hasGem {
		t.Fatalf("exclude gemini should keep antigravity only: hasAG=%v hasGem=%v", hasAG, hasGem)
	}
}
