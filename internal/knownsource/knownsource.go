// Package knownsource fingerprints well-known skills installed outside skill-lock
// (e.g. dws via dingtalk-workspace-cli installer, not npx skills).
package knownsource

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/skilllock"
)

// Source is a known upstream for a skill that may lack skill-lock provenance.
type Source struct {
	// Name is the skill directory / frontmatter name.
	Name string
	// OwnerRepo is GitHub owner/repo.
	OwnerRepo string
	// SkillPath is the path to SKILL.md (or skill folder) inside the repo.
	SkillPath string
	// SourceURL optional clone URL for display.
	SourceURL string
	// Detect returns true if contentPath looks like this known skill.
	Detect func(contentPath string) bool
}

// Catalog is the built-in known-source list (v1: dws only).
var Catalog = []Source{
	{
		Name:      "dws",
		OwnerRepo: "DingTalk-Real-AI/dingtalk-workspace-cli",
		SkillPath: "skills/mono/SKILL.md",
		SourceURL: "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli",
		Detect:    detectDWS,
	},
}

// Match returns the first known source for a skill name + content path.
func Match(name, contentPath string) (Source, bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, s := range Catalog {
		if !strings.EqualFold(s.Name, name) {
			continue
		}
		if s.Detect != nil && !s.Detect(contentPath) {
			continue
		}
		return s, true
	}
	return Source{}, false
}

// AsLockEntry builds a skilllock.Entry for GitHub folder-hash checks.
func (s Source) AsLockEntry() skilllock.Entry {
	return skilllock.Entry{
		Name:       s.Name,
		Source:     s.OwnerRepo,
		SourceType: "github",
		SourceURL:  s.SourceURL,
		SkillPath:  s.SkillPath,
	}
}

// FolderPath returns the skill folder path inside the repo (no SKILL.md).
func (s Source) FolderPath() string {
	p := strings.ReplaceAll(s.SkillPath, "\\", "/")
	lower := strings.ToLower(p)
	if strings.HasSuffix(lower, "/skill.md") {
		return strings.TrimSuffix(p[:len(p)-len("/skill.md")], "/")
	}
	if strings.HasSuffix(lower, "skill.md") {
		return strings.TrimSuffix(p[:len(p)-len("skill.md")], "/")
	}
	return strings.TrimSuffix(p, "/")
}

func detectDWS(contentPath string) bool {
	// Official skill ships NOTICE + LICENSE from the CLI product.
	notice := filepath.Join(contentPath, "NOTICE")
	b, err := os.ReadFile(notice)
	if err != nil {
		// fallback: SKILL.md mentions 钉钉全产品 Skill / cli_version
		md, err2 := os.ReadFile(filepath.Join(contentPath, "SKILL.md"))
		if err2 != nil {
			return false
		}
		s := string(md)
		return strings.Contains(s, "cli_version:") && (strings.Contains(s, "钉钉") || strings.Contains(s, "dws"))
	}
	return strings.Contains(string(b), "DingTalk Workspace CLI") ||
		strings.Contains(string(b), "DingTalk")
}

// GitBlobSHA1 returns the git blob object id for a file (same as GitHub blob sha).
func GitBlobSHA1(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return GitBlobSHA1Bytes(b), nil
}

// GitBlobSHA1Bytes hashes content as a git blob.
func GitBlobSHA1Bytes(b []byte) string {
	h := sha1.New()
	_, _ = fmt.Fprintf(h, "blob %d\x00", len(b))
	_, _ = h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
