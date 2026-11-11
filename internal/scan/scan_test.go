package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

func TestComputeWireGaps(t *testing.T) {
	tmp := t.TempDir()
	agents := filepath.Join(tmp, ".agents", "skills")
	claude := filepath.Join(tmp, ".claude", "skills")
	cursor := filepath.Join(tmp, ".cursor", "skills")
	for _, d := range []string{agents, claude, cursor} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// skill only in agents — no tool links
	solo := filepath.Join(agents, "solo")
	if err := os.MkdirAll(solo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(solo, "SKILL.md"), []byte("---\nname: solo\n---\n# s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// skill in agents + correct claude link, missing cursor
	wired := filepath.Join(agents, "wired")
	if err := os.MkdirAll(wired, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wired, "SKILL.md"), []byte("---\nname: wired\n---\n# w\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wired, filepath.Join(claude, "wired")); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = tmp
	cfg.AgentsSkills = agents
	cfg.OnlyTools = []string{"claude", "cursor"}
	cfg.NoProbe = true
	cfg.NoProject = true

	vl, err := vendorlist.Load(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(cfg, vl)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]WireGap{}
	for _, g := range res.WireGaps {
		byName[g.Name] = g
	}
	soloG, ok := byName["solo"]
	if !ok {
		t.Fatalf("solo should be unwired, gaps=%v", res.WireGaps)
	}
	if len(soloG.Missing) != 2 {
		t.Fatalf("solo missing=%v want claude+cursor", soloG.Missing)
	}
	wiredG, ok := byName["wired"]
	if !ok {
		t.Fatal("wired should still miss cursor")
	}
	if len(wiredG.Missing) != 1 || wiredG.Missing[0] != "cursor" {
		t.Fatalf("wired missing=%v", wiredG.Missing)
	}
	if len(wiredG.Wrong) != 0 {
		t.Fatalf("wired wrong=%v", wiredG.Wrong)
	}
}

func TestRunSkipsASMBackupDirs(t *testing.T) {
	tmp := t.TempDir()
	agents := filepath.Join(tmp, ".agents", "skills")
	claude := filepath.Join(tmp, ".claude", "skills")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claude, 0o755); err != nil {
		t.Fatal(err)
	}

	// real skill
	realDir := filepath.Join(claude, "demo")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "SKILL.md"), []byte("---\nname: demo\n---\n# demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// apply backup leftover — must not appear in scan
	bakDir := filepath.Join(claude, "demo.bak-20260721-161027-7b88")
	if err := os.MkdirAll(bakDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bakDir, "SKILL.md"), []byte("---\nname: demo\n---\n# old demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.Home = tmp
	cfg.AgentsSkills = agents
	cfg.OnlyTools = []string{"claude"}
	cfg.NoProbe = true
	cfg.NoProject = true

	vl, err := vendorlist.Load(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(cfg, vl)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Entries {
		if fsutil.IsASMBackupDir(filepath.Base(e.Path)) {
			t.Fatalf("backup dir should not be scanned: %s", e.Path)
		}
	}
	if len(res.Entries) != 1 {
		t.Fatalf("entries=%d want 1 demo", len(res.Entries))
	}
	if res.Entries[0].Name != "demo" {
		t.Fatalf("got %s", res.Entries[0].Name)
	}
}

func TestScanDisabledLinksAndExternalContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	cfg.NoProject = true
	cfg.NoProbe = true
	cfg.OnlyTools = []string{"claude"}
	canonical := filepath.Join(cfg.AgentsSkills, "demo")
	external := filepath.Join(home, "external")
	toolRoot := filepath.Join(home, ".claude", "skills")
	for _, p := range []string{canonical, external, toolRoot} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for p, name := range map[string]string{canonical: "demo", external: "external"} {
		if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(external, filepath.Join(toolRoot, "external")); err != nil {
		t.Fatal(err)
	}
	gates, err := skillgate.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = gates.Disable("demo", []string{"claude"}); err != nil {
		t.Fatal(err)
	}
	vl, err := vendorlist.Load(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(cfg, vl)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.WireGaps) > 0 {
		t.Errorf("disabled tool falsely reported missing: %+v", res.WireGaps)
	}
	found := false
	for _, e := range res.Entries {
		if e.Name == "external" && e.Symlink {
			found = true
		}
	}
	if !found {
		t.Error("live symlink skill missing from scan")
	}
}
