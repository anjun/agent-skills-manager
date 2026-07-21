package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFrontmatterName(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := "---\nname: Grill Me\ndescription: test skill\n---\n\n# Body\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(skillDir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "grill-me" {
		t.Fatalf("name=%q want grill-me", s.Name)
	}
	if s.ContentHash == "" {
		t.Fatal("empty hash")
	}
	if s.Description != "test skill" {
		t.Fatalf("desc=%q", s.Description)
	}
}

func TestFindSkillMDMissing(t *testing.T) {
	dir := t.TempDir()
	if _, ok := FindSkillMD(dir); ok {
		t.Fatal("expected missing")
	}
}
