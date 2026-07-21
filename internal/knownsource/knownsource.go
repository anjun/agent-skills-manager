// Package knownsource fingerprints well-known skills installed outside skill-lock
// (e.g. dws via dingtalk-workspace-cli installer, not npx skills).
//
// Each catalog entry describes:
//   - how to detect the skill on disk (Detect)
//   - where its upstream lives on GitHub (OwnerRepo + SkillPath)
//   - how to refresh it (Update: CLI steps, optional binary dependency)
package knownsource

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/skilllock"
)

// UpdateStep is one external command used to refresh a known skill.
type UpdateStep struct {
	// Argv is program + args, e.g. {"dws", "upgrade", "-y"}.
	Argv []string
	// ContinueOnError: log failure and continue (e.g. upgrade optional, setup required).
	ContinueOnError bool
	// Label overrides the log line; empty uses joined Argv.
	Label string
}

// UpdateSpec is the auto-update path for a known-source skill.
// Leave nil if check-only (outdated can still report; --update will skip).
type UpdateSpec struct {
	// Action is the short id reported in UpdateItem.Action (e.g. "dws", "cli").
	Action string
	// RequiresBin must exist on PATH before running Steps (optional).
	RequiresBin string
	// InstallHint is shown when RequiresBin is missing.
	InstallHint string
	// OncePerMachine: only one update run even if multiple agent homes have copies.
	// Default treated as true when unset via Once() helper — set explicitly.
	OncePerMachine bool
	// Steps run in order via the caller's RunCmd.
	Steps []UpdateStep
	// SuccessMessage is stored on successful UpdateItem.
	SuccessMessage string
	// OutdatedHint is appended to outdated check messages when files differ.
	OutdatedHint string
}

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
	// PreferRelease compares against GitHub latest release tag instead of default branch HEAD.
	// Use for products where installers ship release-bundled skills (e.g. dws).
	PreferRelease bool
	// Detect returns true if contentPath looks like this known skill.
	// If nil, name match alone is enough.
	Detect func(contentPath string) bool
	// Update describes how --update refreshes this skill (optional).
	Update *UpdateSpec
}

// Catalog is the built-in known-source list.
// Prefer skill-lock / git when possible; only add installers that bypass both.
var Catalog = []Source{
	{
		Name:          "dws",
		OwnerRepo:     "DingTalk-Real-AI/dingtalk-workspace-cli",
		SkillPath:     "skills/mono/SKILL.md",
		SourceURL:     "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli",
		PreferRelease: true, // dws skill setup / upgrade install release embeds, not main HEAD
		Detect:        detectDWS,
		Update: &UpdateSpec{
			Action:         "dws",
			RequiresBin:    "dws",
			InstallHint:    "curl -fsSL https://raw.githubusercontent.com/DingTalk-Real-AI/dingtalk-workspace-cli/main/scripts/install-skills.sh | DWS_SKILLS_ROOT=$HOME sh",
			OncePerMachine: true,
			Steps: []UpdateStep{
				{Argv: []string{"dws", "upgrade", "-y"}, ContinueOnError: true},
				{Argv: []string{"dws", "skill", "setup", "--mode", "mono", "--target", "all", "--yes"}},
			},
			SuccessMessage: "dws upgrade -y + skill setup mono (once; compare uses latest release not main HEAD)",
			OutdatedHint:   "update via dws upgrade -y then dws skill setup --mode mono --yes",
		},
	},
}

// Match returns the first known source for a skill name + content path fingerprint.
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

// Lookup returns a catalog entry by skill name (no filesystem Detect).
// Used by --update when provenance is already ProvKnown.
func Lookup(name string) (Source, bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, s := range Catalog {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
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

// OutdatedHint returns the human hint for check messages.
func (s Source) OutdatedHint() string {
	if s.Update != nil && strings.TrimSpace(s.Update.OutdatedHint) != "" {
		return s.Update.OutdatedHint
	}
	if s.PreferRelease {
		return "update via upstream installer / release"
	}
	return "update via upstream installer"
}

// ActionID is the UpdateItem.Action string for this source.
func (s Source) ActionID() string {
	if s.Update != nil && s.Update.Action != "" {
		return s.Update.Action
	}
	return "known"
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
