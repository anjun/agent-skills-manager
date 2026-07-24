package plan

import (
	"os"
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

func TestBuildWiresOnlyMissingLinks(t *testing.T) {
	tmp := t.TempDir()
	agents := filepath.Join(tmp, ".agents", "skills")
	claude := filepath.Join(tmp, ".claude", "skills")
	cursor := filepath.Join(tmp, ".cursor", "skills")
	for _, d := range []string{agents, claude, cursor} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	skillDir := filepath.Join(agents, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// claude already linked correctly
	if err := os.Symlink(skillDir, filepath.Join(claude, "demo")); err != nil {
		t.Fatal(err)
	}
	// cursor missing

	cfg := config.Default()
	cfg.Home = tmp
	cfg.AgentsSkills = agents
	cfg.OnlyTools = []string{"claude", "cursor"}
	cfg.NoProject = true
	cfg.NoProbe = true

	res := &scan.Result{
		Entries: []scan.Entry{
			{
				Class: scan.ClassCanonical,
				Name:  "demo",
				Path:  skillDir,
				Tool:  "agents",
				Hash:  "h1",
			},
			{
				Class:   scan.ClassCanonical,
				Name:    "demo",
				Path:    filepath.Join(claude, "demo"),
				Tool:    "claude",
				Hash:    "h1",
				Symlink: true,
				Target:  skillDir,
			},
		},
	}
	p := Build(cfg, res)
	var linkCursor, skipClaude, linkClaude int
	for _, a := range p.Actions {
		if a.Name != "demo" {
			continue
		}
		switch a.Type {
		case ActionLink:
			if a.Tool == "cursor" {
				linkCursor++
			}
			if a.Tool == "claude" {
				linkClaude++
			}
		case ActionSkip:
			if a.Tool == "claude" {
				skipClaude++
			}
		}
	}
	if linkCursor != 1 {
		t.Fatalf("want 1 cursor link, got %d; actions=%+v", linkCursor, p.Actions)
	}
	if linkClaude != 0 {
		t.Fatalf("claude already linked; should not ActionLink, got %d", linkClaude)
	}
	if skipClaude == 0 {
		t.Fatal("claude should be skip already-linked")
	}
}
