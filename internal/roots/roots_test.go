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
	if !looksLikeSkillRoot("/repo/.agents/skills") {
		t.Fatal("expected .agents/skills to match")
	}
	if !looksLikeSkillRoot("/repo/skills") {
		t.Fatal("expected skills/ to match")
	}
	if looksLikeSkillRoot("/repo") {
		t.Fatal("repo root should not look like skill root")
	}
}
