package vendorlist

import (
	"path/filepath"
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
)

func TestVendorPathAndName(t *testing.T) {
	yaml := []byte(`
path_globs:
  - "/tmp/fake-home/.codex/skills/.system/**"
names_by_tool:
  grok:
    - help
names_global:
  - create-skill
`)
	cfg := config.Default()
	l, err := Load(yaml, cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join("/tmp/fake-home", ".codex", "skills", ".system", "skill-creator")
	if !l.IsVendorPath(p) {
		t.Fatalf("expected vendor path %s", p)
	}
	if !l.IsVendorName("grok", "help") {
		t.Fatal("help should be vendor")
	}
	if !l.IsVendorName("claude", "create-skill") {
		t.Fatal("global create-skill")
	}
	if l.IsVendorName("claude", "grill-me") {
		t.Fatal("grill-me should not be vendor")
	}
}

func TestClassifyVendorGrokProbeOverridesDenylist(t *testing.T) {
	yaml := []byte(`
names_by_tool:
  grok:
    - help
    - strict-code-review
names_global:
  - help
  - create-skill
  - check-work
`)
	cfg := config.Default()
	l, err := Load(yaml, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate successful probe: only help is bundled, not strict-code-review
	l.GrokProbeOK = true
	helpDir := filepath.Clean("/Users/me/.grok/skills/help")
	l.GrokBundledPath[helpDir] = true
	l.GrokBundledName["help"] = true

	ok, reason := l.ClassifyVendor("grok", "help", helpDir)
	if !ok {
		t.Fatalf("help should be vendor via probe, reason=%s", reason)
	}
	if reason == "denylist name" {
		t.Fatal("should use probe reason, not denylist")
	}

	strict := filepath.Clean("/Users/me/.grok/skills/strict-code-review")
	ok, _ = l.ClassifyVendor("grok", "strict-code-review", strict)
	if ok {
		t.Fatal("strict-code-review is user per probe; denylist must not apply")
	}

	// User-dir override of a reserved platform skill: inspect says type=user,
	// but names_global must still block migration into multi-tool agents.
	createSkillUser := filepath.Clean("/Users/me/.grok/skills/create-skill")
	ok, reason = l.ClassifyVendor("grok", "create-skill", createSkillUser)
	if !ok || reason != "global denylist name" {
		t.Fatalf("create-skill user override should be global denylist, got ok=%v reason=%s", ok, reason)
	}
	checkWork := filepath.Clean("/Users/me/.grok/skills/check-work")
	ok, reason = l.ClassifyVendor("grok", "check-work", checkWork)
	if !ok || reason != "global denylist name" {
		t.Fatalf("check-work should be global denylist, got ok=%v reason=%s", ok, reason)
	}
}

func TestClassifyVendorGrokFallbackWhenProbeFails(t *testing.T) {
	yaml := []byte(`
names_by_tool:
  grok:
    - help
`)
	cfg := config.Default()
	l, err := Load(yaml, cfg)
	if err != nil {
		t.Fatal(err)
	}
	l.GrokProbeOK = false
	l.GrokProbeErr = "no binary"
	ok, reason := l.ClassifyVendor("grok", "help", "/Users/me/.grok/skills/help")
	if !ok || reason != "denylist name" {
		t.Fatalf("fallback denylist expected, got ok=%v reason=%s", ok, reason)
	}
}
