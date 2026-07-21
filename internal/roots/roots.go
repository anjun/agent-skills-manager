package roots

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
)

// Root is a known skill root directory.
type Root struct {
	Tool   string // agents|claude|cursor|grok|codex|gemini|extra|project
	Path   string // absolute
	Kind   string // user|vendor|canonical|project
	Wire   bool   // after migrate, create symlinks here for user skills
}

// Project skill directory names under a repo root (agentskills.io + common tool layouts).
var projectSkillRels = []string{
	".agents/skills",
	".claude/skills",
	".cursor/skills",
	".codex/skills",
	".grok/skills",
	".gemini/skills",
	"skills",
}

// DefaultRoots returns the allowlist of scan roots (decision A).
func DefaultRoots(cfg *config.Config) []Root {
	home := cfg.Home
	var out []Root

	add := func(tool, rel, kind string, wire bool) {
		if tool != "agents" && tool != "extra" && !cfg.ToolEnabled(tool) {
			return
		}
		p := filepath.Join(home, rel)
		out = append(out, Root{Tool: tool, Path: fsutil.Abs(p), Kind: kind, Wire: wire})
	}

	// canonical
	out = append(out, Root{
		Tool: "agents",
		Path: fsutil.Abs(cfg.AgentsSkills),
		Kind: "canonical",
		Wire: false,
	})

	// mainstream user roots (wire after migrate)
	add("claude", ".claude/skills", "user", true)
	add("cursor", ".cursor/skills", "user", true)
	add("grok", ".grok/skills", "user", true)
	add("codex", ".codex/skills", "user", true)
	add("gemini", ".gemini/skills", "user", true)

	// secondary user roots (scan; wire only if listed in WireTools and Wire true)
	if cfg.ToolEnabled("gemini") {
		out = append(out, Root{
			Tool: "gemini",
			Path: fsutil.Abs(filepath.Join(home, ".gemini/config/skills")),
			Kind: "user",
			Wire: false, // wire primary ~/.gemini/skills only
		})
	}
	if cfg.ToolEnabled("codex") {
		out = append(out, Root{
			Tool: "codex",
			Path: fsutil.Abs(filepath.Join(home, ".codex/.agents/skills")),
			Kind: "user",
			Wire: false,
		})
	}

	// vendor path roots (classify only)
	vendorRels := []struct{ tool, rel string }{
		{"grok", ".grok/bundled/skills"},
		{"codex", ".codex/skills/.system"},
		{"codex", ".codex/vendor_imports"},
	}
	for _, v := range vendorRels {
		if !cfg.ToolEnabled(v.tool) {
			continue
		}
		out = append(out, Root{
			Tool: v.tool,
			Path: fsutil.Abs(filepath.Join(home, v.rel)),
			Kind: "vendor",
			Wire: false,
		})
	}

	for _, er := range cfg.ExtraRoots {
		out = append(out, Root{
			Tool: "extra",
			Path: fsutil.Abs(er),
			Kind: "user",
			Wire: false,
		})
	}

	// project-level skills (never migrate): cwd git root + explicit --project-root
	out = append(out, ProjectRoots(cfg)...)

	// de-dupe by path (first wins — global roots take precedence over project)
	seen := map[string]bool{}
	uniq := make([]Root, 0, len(out))
	for _, r := range out {
		if seen[r.Path] {
			continue
		}
		seen[r.Path] = true
		uniq = append(uniq, r)
	}
	return uniq
}

// ProjectRoots returns skill roots for repositories the user cares about.
// Sources: --project-root PATH (repeatable) and, unless --no-project, the
// git work tree containing the current working directory.
//
// PATH may be a repo root (expands known skill subdirs) or a skill container
// directory itself (e.g. /path/to/repo/.agents/skills).
func ProjectRoots(cfg *config.Config) []Root {
	var out []Root
	seenRepo := map[string]bool{}

	addRepo := func(repo string) {
		repo = fsutil.Abs(repo)
		if repo == "" || repo == fsutil.Abs(cfg.Home) {
			return
		}
		if seenRepo[repo] {
			return
		}
		seenRepo[repo] = true
		for _, rel := range projectSkillRels {
			out = append(out, Root{
				Tool: "project",
				Path: fsutil.Abs(filepath.Join(repo, rel)),
				Kind: "project",
				Wire: false,
			})
		}
	}

	for _, p := range cfg.ProjectRoots {
		p = fsutil.Abs(p)
		if looksLikeSkillRoot(p) {
			out = append(out, Root{
				Tool: "project",
				Path: p,
				Kind: "project",
				Wire: false,
			})
			continue
		}
		if root, ok := fsutil.GitWorkTreeRoot(p); ok {
			addRepo(root)
		} else {
			addRepo(p)
		}
	}

	if !cfg.NoProject {
		if cwd, err := os.Getwd(); err == nil {
			if root, ok := fsutil.GitWorkTreeRoot(cwd); ok {
				addRepo(root)
			}
		}
	}
	return out
}

func looksLikeSkillRoot(p string) bool {
	p = filepath.Clean(p)
	for _, rel := range projectSkillRels {
		rel = filepath.FromSlash(rel)
		if p == rel || strings.HasSuffix(p, string(filepath.Separator)+rel) {
			return true
		}
	}
	return false
}

// Existing returns roots whose path exists as a directory.
func Existing(roots []Root) []Root {
	var out []Root
	for _, r := range roots {
		st, err := os.Stat(r.Path)
		if err == nil && st.IsDir() {
			out = append(out, r)
		}
	}
	return out
}

// WireRoots returns user-level roots that should receive symlinks after migrate.
func WireRoots(cfg *config.Config) []Root {
	var out []Root
	for _, r := range DefaultRoots(cfg) {
		if r.Wire {
			out = append(out, r)
		}
	}
	return out
}
