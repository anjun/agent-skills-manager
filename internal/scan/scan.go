package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/roots"
	"github.com/anj/agent-skills-manager/internal/skill"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Class of a discovered skill instance.
type Class string

const (
	ClassCanonical Class = "canonical"
	ClassUser      Class = "user"
	ClassVendor    Class = "vendor"
	ClassProject   Class = "project"
)

// Entry is one skill directory found on disk.
type Entry struct {
	Class   Class  `json:"class"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Tool    string `json:"tool"`
	Hash    string `json:"hash"`
	Symlink bool   `json:"symlink"`
	Target  string `json:"target,omitempty"` // if symlink
	Reason  string `json:"reason,omitempty"`
}

// WireGap is a skill already under ~/.agents/skills but missing or wrong
// symlinks in one or more tool wire roots (claude/cursor/grok/…).
type WireGap struct {
	Name       string   `json:"name"`
	AgentsPath string   `json:"agents_path"`
	Missing    []string `json:"missing,omitempty"` // tools with no path
	Wrong      []string `json:"wrong,omitempty"`   // real dir / bad symlink
}

// Result is a full scan.
type Result struct {
	Entries  []Entry   `json:"entries"`
	WireGaps []WireGap `json:"wire_gaps,omitempty"`
}

// Run discovers skills under known roots.
func Run(cfg *config.Config, vl *vendorlist.List) (*Result, error) {
	rs := roots.Existing(roots.DefaultRoots(cfg))
	var entries []Entry
	seenPath := map[string]bool{}

	for _, r := range rs {
		found, err := walkRoot(r, cfg, vl)
		if err != nil {
			continue
		}
		for _, e := range found {
			if seenPath[e.Path] {
				continue
			}
			seenPath[e.Path] = true
			entries = append(entries, e)
		}
	}
	res := &Result{Entries: entries}
	res.WireGaps = ComputeWireGaps(cfg, res)
	return res, nil
}

// ComputeWireGaps finds canonical skills that lack correct tool symlinks.
// Manual drops into ~/.agents/skills are the main case: content is home, but
// claude/cursor/grok/codex/gemini roots may have nothing yet.
func ComputeWireGaps(cfg *config.Config, res *Result) []WireGap {
	if res == nil {
		return nil
	}
	agents := fsutil.Abs(cfg.AgentsSkills)
	// name → agents content path
	agentsPath := map[string]string{}
	for _, e := range res.Entries {
		if e.Class != ClassCanonical {
			continue
		}
		content := e.Path
		if e.Symlink && e.Target != "" {
			content = e.Target
		}
		content = fsutil.Abs(content)
		if !pathUnder(content, agents) {
			continue
		}
		// prefer the real agents dir entry over a tool symlink row
		if cur, ok := agentsPath[e.Name]; ok {
			if pathUnder(e.Path, agents) && !e.Symlink {
				agentsPath[e.Name] = fsutil.Abs(e.Path)
			} else if cur == "" {
				agentsPath[e.Name] = content
			}
			continue
		}
		if pathUnder(e.Path, agents) && !e.Symlink {
			agentsPath[e.Name] = fsutil.Abs(e.Path)
		} else {
			agentsPath[e.Name] = content
		}
	}

	wireRoots := roots.WireRoots(cfg)
	if len(wireRoots) == 0 || len(agentsPath) == 0 {
		return nil
	}

	names := make([]string, 0, len(agentsPath))
	for n := range agentsPath {
		names = append(names, n)
	}
	sort.Strings(names)

	gates, _ := skillgate.Open(cfg)
	var gaps []WireGap
	for _, name := range names {
		dest := agentsPath[name]
		g := WireGap{Name: name, AgentsPath: dest}
		for _, wr := range wireRoots {
			if gates != nil && gates.IsToolOff(name, wr.Tool) {
				continue
			}
			linkPath := filepath.Join(wr.Path, name)
			switch fsutil.SkillLinkStateOf(linkPath, dest) {
			case fsutil.LinkOK:
				// good
			case fsutil.LinkMissing:
				g.Missing = append(g.Missing, wr.Tool)
			case fsutil.LinkWrong:
				g.Wrong = append(g.Wrong, wr.Tool)
			}
		}
		if len(g.Missing) > 0 || len(g.Wrong) > 0 {
			gaps = append(gaps, g)
		}
	}
	return gaps
}

func pathUnder(path, root string) bool {
	path = fsutil.Abs(path)
	root = fsutil.Abs(root)
	if path == root {
		return true
	}
	prefix := root + string(filepath.Separator)
	return strings.HasPrefix(path, prefix)
}

func walkRoot(r roots.Root, cfg *config.Config, vl *vendorlist.List) ([]Entry, error) {
	var out []Entry
	// depth-limited: skill dirs are either immediate children or nested one level (multi packs)
	// Walk but skip huge trees: max depth 4 from root
	base := r.Path
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		rel, _ := filepath.Rel(base, path)
		baseName := filepath.Base(path)
		// apply leaves displaced real dirs as name.bak-<txID>; never treat as skills
		if path != base && fsutil.IsASMBackupDir(baseName) {
			return filepath.SkipDir
		}
		if rel != "." {
			depth := strings.Count(rel, string(filepath.Separator)) + 1
			if depth > 4 {
				return filepath.SkipDir
			}
			// skip hidden system-ish dirs content already handled as vendor root
			if baseName == "node_modules" || baseName == ".git" {
				return filepath.SkipDir
			}
		}
		md, ok := skill.FindSkillMD(path)
		if !ok {
			return nil
		}
		_ = md
		// skip if this is the root itself without being a skill (skills root isn't a skill)
		if path == base {
			// only if SKILL.md at root of skills dir (unusual)
		}
		e := classifyPath(path, r, cfg, vl)
		if e != nil {
			out = append(out, *e)
			// Do not descend into packages; SkipDir on a symlink would skip siblings.
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})
	return out, err
}

func classifyPath(path string, r roots.Root, cfg *config.Config, vl *vendorlist.List) *Entry {
	// resolve symlink for hashing content of target
	sym := fsutil.IsSymlink(path)
	target := ""
	loadPath := path
	if sym {
		t, err := fsutil.ReadLinkAbs(path)
		if err == nil {
			target = t
			loadPath = t
		}
	}

	sk, err := skill.Load(loadPath)
	if err != nil {
		return nil
	}

	e := &Entry{
		Name:    sk.Name,
		Path:    path,
		Tool:    r.Tool,
		Hash:    sk.ContentHash,
		Symlink: sym,
		Target:  target,
	}

	// explicit project roots (repo-local) — never migrate
	if r.Kind == "project" {
		e.Class = ClassProject
		e.Tool = "project"
		e.Reason = "project skill root"
		return e
	}

	// vendor first (before project — curated caches may be git clones)
	if r.Kind == "vendor" {
		e.Class = ClassVendor
		e.Reason = "vendor root"
		return e
	}
	if ok, reason := vl.ClassifyVendor(r.Tool, sk.Name, path); ok {
		e.Class = ClassVendor
		e.Reason = reason
		return e
	}
	if loadPath != path {
		if ok, reason := vl.ClassifyVendor(r.Tool, sk.Name, loadPath); ok {
			e.Class = ClassVendor
			e.Reason = reason
			return e
		}
	}

	// project: under git work tree, but NOT under home skill global roots
	if isProjectPath(path, cfg) {
		e.Class = ClassProject
		e.Reason = "inside git work tree"
		return e
	}

	// canonical
	agents := fsutil.Abs(cfg.AgentsSkills)
	if path == agents || strings.HasPrefix(path, agents+string(filepath.Separator)) {
		// only direct skill under agents, not nested multi? multi under agents is ok
		e.Class = ClassCanonical
		e.Reason = "under ~/.agents/skills"
		return e
	}
	if target != "" && (target == agents || strings.HasPrefix(target, agents+string(filepath.Separator))) {
		e.Class = ClassCanonical
		e.Reason = "symlink into ~/.agents/skills"
		return e
	}

	e.Class = ClassUser
	e.Reason = "user tool root"
	return e
}

func isProjectPath(path string, cfg *config.Config) bool {
	// global skill roots under home should never be project
	globals := []string{
		cfg.AgentsSkills,
		filepath.Join(cfg.Home, ".claude", "skills"),
		filepath.Join(cfg.Home, ".cursor", "skills"),
		filepath.Join(cfg.Home, ".grok", "skills"),
		filepath.Join(cfg.Home, ".codex", "skills"),
		filepath.Join(cfg.Home, ".gemini", "skills"),
		filepath.Join(cfg.Home, ".gemini", "config", "skills"),
	}
	ap := fsutil.Abs(path)
	for _, g := range globals {
		g = fsutil.Abs(g)
		if ap == g || strings.HasPrefix(ap, g+string(filepath.Separator)) {
			return false
		}
	}
	return fsutil.IsInsideGitWorkTree(path)
}
