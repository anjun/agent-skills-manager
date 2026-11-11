package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/i18n"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

func TestLocalProvenanceHintManual(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: solo\n---\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zh := localProvenanceHint(i18n.New(i18n.ZH), "solo", dir, nil)
	en := localProvenanceHint(i18n.New(i18n.EN), "solo", dir, nil)
	if !strings.Contains(zh, "asm remove") || !strings.Contains(zh, "手工") {
		t.Fatalf("zh hint=%q", zh)
	}
	if !strings.Contains(en, "no provenance") || !strings.Contains(en, "asm remove") {
		t.Fatalf("en hint=%q", en)
	}
	if zh == en {
		t.Fatal("zh and en should differ")
	}
}

func TestLocalProvenanceHintKnownEmpty(t *testing.T) {
	// dws with NOTICE should not get unknown hint when matched
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "NOTICE"), []byte("DingTalk Workspace CLI\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: dws\n---\n"), 0o644)
	if got := localProvenanceHint(i18n.New(i18n.EN), "dws", dir, nil); got != "" {
		t.Fatalf("want empty for known dws, got %q", got)
	}
}

func TestRunWithEmptyHomeish(t *testing.T) {
	// smoke: Run should not panic with default config (may find real skills)
	cfg := config.Default()
	vl, err := vendorlist.Load([]byte("skills: []\n"), cfg)
	if err != nil {
		// minimal list may still load empty
		vl, err = vendorlist.Load(nil, cfg)
		if err != nil {
			t.Fatal(err)
		}
	}
	issues, err := Run(cfg, vl)
	if err != nil {
		t.Fatal(err)
	}
	_ = issues
}

func TestGateHealthMatchesDisk(t *testing.T) {
	for _, mode := range []string{"orphan", "parked", "broken", "backup"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			cfg := config.Default()
			cfg.NoProject = true
			cfg.NoProbe = true
			p := filepath.Join(cfg.AgentsSkills, "demo")
			if err := os.MkdirAll(p, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte("---\nname: demo\n---\n"), 0644); err != nil {
				t.Fatal(err)
			}
			g, err := skillgate.Open(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "parked" {
				if _, err := g.Disable("demo", nil); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(home, ".agents", ".skill-lock.json"), []byte(`{"version":3,"skills":{"demo":{"source":"x/y"}}}`), 0644); err != nil {
					t.Fatal(err)
				}
			} else if mode == "orphan" || mode == "backup" {
				if _, err := g.Disable("demo", []string{"claude"}); err != nil {
					t.Fatal(err)
				}
				if mode == "backup" {
					backup := filepath.Join(g.DisabledDir, "_toolbak", "claude__demo")
					if err := os.MkdirAll(filepath.Dir(backup), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(p, backup); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.RemoveAll(p); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Symlink(filepath.Join(home, "missing"), filepath.Join(cfg.AgentsSkills, "broken")); err != nil {
					t.Fatal(err)
				}
			}
			vl, err := vendorlist.Load(nil, cfg)
			if err != nil {
				t.Fatal(err)
			}
			issues, err := Run(cfg, vl)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, is := range issues {
				if mode == "orphan" && is.Path == "gate:demo" {
					found = true
					if strings.Contains(is.Message, "asm enable") || !strings.Contains(is.Message, "asm remove") {
						t.Errorf("misleading orphan advice: %s", is.Message)
					}
				}
				if mode == "backup" && is.Path == "gate:demo" {
					found = true
					if strings.Contains(is.Message, "asm enable") || strings.Contains(is.Message, "asm remove") {
						t.Errorf("misleading backup advice: %s", is.Message)
					}
				}
				if mode == "parked" && strings.Contains(is.Message, "skill-lock") {
					t.Errorf("parked skill reported missing: %s", is.Message)
				}
				if mode == "broken" && is.Code == CodeBrokenLink {
					found = true
				}
			}
			if mode != "parked" && !found {
				t.Errorf("missing %s finding: %+v", mode, issues)
			}
		})
	}
}
