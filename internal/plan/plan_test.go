package plan

import (
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/scan"
)

func TestBuildMigrateUserSkill(t *testing.T) {
	cfg := config.Default()
	cfg.AgentsSkills = filepath.Join(t.TempDir(), "agents", "skills")
	cfg.Home = t.TempDir()
	res := &scan.Result{
		Entries: []scan.Entry{
			{
				Class: scan.ClassUser,
				Name:  "demo-skill",
				Path:  filepath.Join(cfg.Home, ".codex", "skills", "demo-skill"),
				Tool:  "codex",
				Hash:  "abc",
			},
			{
				Class: scan.ClassVendor,
				Name:  "help",
				Path:  filepath.Join(cfg.Home, ".grok", "skills", "help"),
				Tool:  "grok",
				Hash:  "def",
			},
		},
	}
	p := Build(cfg, res)
	var mig, skip int
	for _, a := range p.Actions {
		switch a.Type {
		case ActionMigrate:
			if a.Name == "demo-skill" {
				mig++
			}
		case ActionSkip:
			if a.Name == "help" {
				skip++
			}
		}
	}
	if mig == 0 {
		t.Fatal("expected migrate for demo-skill")
	}
	if skip == 0 {
		t.Fatal("expected skip for vendor help")
	}
}
