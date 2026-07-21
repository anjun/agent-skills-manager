package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/i18n"
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
