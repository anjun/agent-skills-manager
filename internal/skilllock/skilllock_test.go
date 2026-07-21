package skilllock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileAndLookup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".skill-lock.json")
	raw := map[string]any{
		"version": 3,
		"skills": map[string]any{
			"grill-me": map[string]any{
				"source":          "mattpocock/skills",
				"sourceType":      "github",
				"sourceUrl":       "https://github.com/mattpocock/skills.git",
				"skillPath":       "skills/productivity/grill-me/SKILL.md",
				"skillFolderHash": "abc123def456",
			},
		},
	}
	b, _ := json.Marshal(raw)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Skills) != 1 {
		t.Fatalf("skills=%d", len(f.Skills))
	}
	e := toEntry("grill-me", f.Skills["grill-me"], p)
	if !e.CanCheckGitHub() {
		t.Fatalf("expected checkable: %+v", e)
	}
	if ownerRepo(e) != "mattpocock/skills" {
		t.Fatalf("ownerRepo=%s", ownerRepo(e))
	}
}

func TestOwnerRepoFromURL(t *testing.T) {
	e := Entry{SourceURL: "https://github.com/anthropics/skills.git"}
	if got := ownerRepo(e); got != "anthropics/skills" {
		t.Fatalf("got %s", got)
	}
}

func TestSkillFolderHashFromTree(t *testing.T) {
	tree := &RepoTree{
		SHA: "rootsha",
		Folders: map[string]string{
			"skills/productivity/grill-me": "foldersha1",
			"skills/frontend-design":       "foldersha2",
		},
	}
	if got := skillFolderHashFromTree(tree, "skills/productivity/grill-me/SKILL.md"); got != "foldersha1" {
		t.Fatalf("got %s", got)
	}
	if got := skillFolderHashFromTree(tree, "skills/frontend-design/skill.md"); got != "foldersha2" {
		t.Fatalf("got %s", got)
	}
	if got := skillFolderHashFromTree(tree, "SKILL.md"); got != "rootsha" {
		t.Fatalf("root got %s", got)
	}
}
