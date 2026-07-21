package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

func TestLocalProvenanceHintManual(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: solo\n---\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hint := localProvenanceHint("solo", dir, nil)
	if hint == "" || !strings.Contains(hint, "provenance") && !strings.Contains(hint, "手工") {
		// Chinese default in production; tests may use default lang from env
		if !strings.Contains(hint, "asm remove") {
			t.Fatalf("hint=%q", hint)
		}
	}
}

func TestLocalProvenanceHintKnownEmpty(t *testing.T) {
	// dws with NOTICE should not get unknown hint when matched
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "NOTICE"), []byte("DingTalk Workspace CLI\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: dws\n---\n"), 0o644)
	if got := localProvenanceHint("dws", dir, nil); got != "" {
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
