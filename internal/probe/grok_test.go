package probe

import (
	"path/filepath"
	"testing"
)

func TestParseGrokInspectJSON(t *testing.T) {
	raw := []byte(`{
  "skills": [
    {
      "name": "help",
      "source": { "type": "bundled", "path": "/Users/me/.grok/skills/help/SKILL.md" }
    },
    {
      "name": "strict-code-review",
      "source": { "type": "user", "path": "/Users/me/.grok/skills/strict-code-review/SKILL.md" }
    },
    {
      "name": "design",
      "source": { "type": "bundled", "path": "/Users/me/.grok/bundled/skills/design/SKILL.md" }
    }
  ]
}`)
	paths, names, err := ParseGrokInspectJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("names=%v", names)
	}
	wantHelp := filepath.Clean("/Users/me/.grok/skills/help")
	wantDesign := filepath.Clean("/Users/me/.grok/bundled/skills/design")
	got := map[string]bool{}
	for _, p := range paths {
		got[filepath.Clean(p)] = true
	}
	if !got[wantHelp] || !got[wantDesign] {
		t.Fatalf("paths=%v want %s and %s", paths, wantHelp, wantDesign)
	}
	// user skill must not appear
	for _, n := range names {
		if n == "strict-code-review" {
			t.Fatal("user skill should not be in bundled names")
		}
	}
}
