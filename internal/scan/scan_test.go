package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

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
