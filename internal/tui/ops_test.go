package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

func TestUninstallRemovesGateChoices(t *testing.T) {
	for _, scope := range []string{"claude", "global", "orphan"} {
		t.Run(scope, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			cfg := config.Default()
			cfg.NoProject = true
			cfg.NoProbe = true
			vl, err := vendorlist.Load(nil, cfg)
			if err != nil {
				t.Fatal(err)
			}
			d := &Deps{Cfg: cfg, VL: vl}
			path := filepath.Join(cfg.AgentsSkills, "demo")
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("---\nname: demo\ndescription: demo\n---\n"), 0644); err != nil {
				t.Fatal(err)
			}
			// A real tool copy is parked separately when disabled.
			toolPath := filepath.Join(home, ".claude", "skills", "demo")
			if err := os.MkdirAll(toolPath, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(toolPath, "SKILL.md"), []byte("demo"), 0644); err != nil {
				t.Fatal(err)
			}
			tools := []string{"claude"}
			if scope == "global" {
				tools = nil
			}
			if _, err := d.disableSkill("demo", tools); err != nil {
				t.Fatal(err)
			}
			if scope == "orphan" {
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
			}
			choices, err := d.listRemovableSkills()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(choices, "demo") {
				t.Fatalf("disabled/orphan skill missing from uninstall choices: %v", choices)
			}
			before, err := d.listGateEntries()
			if err != nil {
				t.Fatal(err)
			}
			if scope == "orphan" && len(before) != 0 {
				t.Fatalf("old orphan shown as enableable: %v", before)
			}
			if scope != "orphan" && len(before) != 1 {
				t.Fatalf("installed disabled skill missing: %v", before)
			}
			res, _, err := d.removeSkills([]string{"demo"})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Failed) != 0 {
				t.Fatal(res.Failed)
			}
			rows, err := d.listGateEntries()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 0 {
				t.Errorf("uninstalled skill still in enable list: %+v", rows)
			}
			names, err := d.listCanonicalSkills()
			if err != nil {
				t.Fatal(err)
			}
			if len(names) != 0 {
				t.Errorf("uninstalled skill still in disable list: %v", names)
			}
			store, err := skillgate.Open(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if len(store.List()) != 0 {
				t.Errorf("uninstalled gate remains: %v", store.List())
			}
			for _, path := range store.StoredPaths("demo") {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Errorf("disabled content remains at %s: %v", path, err)
				}
			}
			if _, err := d.enableSkill("demo", nil); err == nil {
				t.Fatal("uninstalled skill can be restored")
			}
		})
	}
}
