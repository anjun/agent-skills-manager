package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill is one Agent Skill directory (folder with SKILL.md).
type Skill struct {
	Name        string // from frontmatter name, else dir base
	Dir         string // absolute path to skill directory
	SkillMD     string // absolute path to SKILL.md
	Description string
	ContentHash string // hash of SKILL.md bytes (v1)
	Frontmatter map[string]any
}

// FindSkillMD returns the SKILL.md path if dir is a skill root.
func FindSkillMD(dir string) (string, bool) {
	for _, name := range []string{"SKILL.md", "skill.md"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// Load reads SKILL.md and returns a Skill.
func Load(dir string) (*Skill, error) {
	md, ok := FindSkillMD(dir)
	if !ok {
		return nil, os.ErrNotExist
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	mdAbs, err := filepath.Abs(md)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(mdAbs)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	s := &Skill{
		Dir:         abs,
		SkillMD:     mdAbs,
		ContentHash: hex.EncodeToString(h[:]),
		Name:        filepath.Base(abs),
		Frontmatter: map[string]any{},
	}
	fm, bodyOK := parseFrontmatter(string(b))
	if bodyOK {
		s.Frontmatter = fm
		if n, ok := fm["name"].(string); ok && strings.TrimSpace(n) != "" {
			s.Name = strings.TrimSpace(n)
		}
		if d, ok := fm["description"].(string); ok {
			s.Description = strings.TrimSpace(d)
		}
	}
	// Normalize name like tools do: spaces/underscores → hyphens, lower
	s.Name = normalizeName(s.Name)
	return s, nil
}

func normalizeName(n string) string {
	n = strings.TrimSpace(strings.ToLower(n))
	n = strings.ReplaceAll(n, "_", "-")
	n = strings.ReplaceAll(n, " ", "-")
	return n
}

func parseFrontmatter(content string) (map[string]any, bool) {
	if !strings.HasPrefix(content, "---") {
		return nil, false
	}
	rest := content[3:]
	// allow leading newline
	rest = strings.TrimPrefix(rest, "\n")
	rest = strings.TrimPrefix(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, false
	}
	block := rest[:end]
	var m map[string]any
	if err := yaml.Unmarshal([]byte(block), &m); err != nil {
		return nil, false
	}
	return m, true
}

// HashTree hashes SKILL.md plus optional scripts/references shallowly (v1: SKILL.md only).
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
