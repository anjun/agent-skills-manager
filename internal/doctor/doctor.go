package doctor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/i18n"
	"github.com/anj/agent-skills-manager/internal/knownsource"
	"github.com/anj/agent-skills-manager/internal/roots"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/skill"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/skilllock"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Actionable issue codes (TUI / automation can offer cleanup).
const (
	CodeBakLeftover = "bak_leftover" // apply-side <name>.bak-<txID> dirs
	CodeBrokenLink  = "broken_link"  // dangling symlink
)

// Issue found by doctor.
type Issue struct {
	Severity string `json:"severity"` // error|warn|info
	Path     string `json:"path"`
	Message  string `json:"message"`
	// Code marks issues that tools may auto-clean (empty = report only).
	Code string `json:"code,omitempty"`
}

// Cleanable reports whether Cleanup may act on this issue.
func (i Issue) Cleanable() bool {
	return i.Code == CodeBakLeftover || i.Code == CodeBrokenLink
}

// Run checks broken symlinks, duplicate real copies, wire roots, and
// U2 provenance discoverability (no network calls).
func Run(cfg *config.Config, vl *vendorlist.List) ([]Issue, error) {
	tr := cfg.T()
	var issues []Issue
	res, err := scan.Run(cfg, vl)
	if err != nil {
		return nil, err
	}
	lockIdx := skilllock.LoadIndex()
	seenSkill := map[string]bool{} // lower name → present on disk (non-vendor)

	byName := map[string][]scan.Entry{}
	for _, e := range res.Entries {
		if e.Class == scan.ClassVendor {
			continue
		}
		key := strings.ToLower(e.Name)
		seenSkill[key] = true
		if e.Class == scan.ClassProject {
			// project skills: only broken-link checks below if symlink
			if e.Symlink {
				if e.Target == "" {
					issues = append(issues, Issue{Severity: "error", Path: e.Path, Message: tr.S("doctor.broken_no_target"), Code: CodeBrokenLink})
				} else if _, err := os.Stat(e.Target); err != nil {
					issues = append(issues, Issue{Severity: "error", Path: e.Path, Message: tr.S("doctor.broken_target", e.Target), Code: CodeBrokenLink})
				}
			}
			continue
		}
		byName[e.Name] = append(byName[e.Name], e)
		if e.Symlink {
			if e.Target == "" {
				issues = append(issues, Issue{Severity: "error", Path: e.Path, Message: tr.S("doctor.broken_no_target"), Code: CodeBrokenLink})
				continue
			}
			if _, err := os.Stat(e.Target); err != nil {
				issues = append(issues, Issue{Severity: "error", Path: e.Path, Message: tr.S("doctor.broken_target", e.Target), Code: CodeBrokenLink})
			}
		}
	}
	for name, list := range byName {
		var reals int
		for _, e := range list {
			if !e.Symlink && e.Class == scan.ClassUser {
				reals++
			}
		}
		if reals > 1 {
			// Prefer a real path so RelHome does not resolve the bare name against cwd.
			p := name
			for _, e := range list {
				if !e.Symlink {
					p = e.Path
					break
				}
			}
			issues = append(issues, Issue{Severity: "warn", Path: p, Message: tr.S("doctor.dup_reals", reals)})
		}
	}
	// wire roots missing; also flag apply-side *.bak-<txID> leftovers
	for _, wr := range roots.WireRoots(cfg) {
		if _, err := os.Stat(wr.Path); err != nil {
			issues = append(issues, Issue{Severity: "info", Path: wr.Path, Message: tr.S("doctor.wire_missing")})
			continue
		}
		ents, err := os.ReadDir(wr.Path)
		if err != nil {
			continue
		}
		for _, ent := range ents {
			if !ent.IsDir() || !fsutil.IsASMBackupDir(ent.Name()) {
				continue
			}
			p := filepath.Join(wr.Path, ent.Name())
			issues = append(issues, Issue{Severity: "info", Path: p, Message: tr.S("doctor.bak_leftover", fsutil.RelHome(p)), Code: CodeBakLeftover})
		}
	}
	// agents dir
	if _, err := os.Stat(cfg.AgentsSkills); err != nil {
		issues = append(issues, Issue{Severity: "warn", Path: cfg.AgentsSkills, Message: tr.S("doctor.agents_missing")})
	}

	// U2: skill-lock entries with no on-disk skill
	if lockIdx != nil {
		var orphans []string
		for name := range lockIdx.ByName {
			if !seenSkill[strings.ToLower(name)] {
				orphans = append(orphans, name)
			}
		}
		sort.Strings(orphans)
		for _, name := range orphans {
			le, _ := lockIdx.Lookup(name)
			msg := tr.S("doctor.lock_orphan", name, name)
			path := "skill-lock:" + name
			if le.LockPath != "" {
				msg = tr.S("doctor.lock_orphan_at", name, fsutil.RelHome(le.LockPath), name)
				path = le.LockPath + "#" + name
			}
			issues = append(issues, Issue{Severity: "warn", Path: path, Message: msg})
		}
	}

	// skill gates (disabled globally or per-tool)
	if gates, err := skillgate.Open(cfg); err == nil {
		for _, name := range gates.List() {
			ent := gates.Get(name)
			if ent.GlobalOff {
				issues = append(issues, Issue{Severity: "info", Path: "gate:" + name, Message: tr.S("doctor.gate_global", name, name)})
				continue
			}
			if len(ent.ToolsOff) > 0 {
				issues = append(issues, Issue{Severity: "info", Path: "gate:" + name,
					Message: tr.S("doctor.gate_tools", name, strings.Join(ent.ToolsOff, ", "), name)})
			}
		}
	}

	// U2: per-skill local provenance (no network) — one row per content path
	seenPath := map[string]bool{}
	for _, e := range res.Entries {
		if e.Class != scan.ClassCanonical && e.Class != scan.ClassUser {
			continue
		}
		contentPath := e.Path
		if e.Symlink && e.Target != "" {
			contentPath = e.Target
		}
		contentPath = fsutil.Abs(contentPath)
		if seenPath[contentPath] {
			continue
		}
		seenPath[contentPath] = true

		hint := localProvenanceHint(tr, e.Name, contentPath, lockIdx)
		if hint == "" {
			continue
		}
		issues = append(issues, Issue{Severity: "info", Path: contentPath, Message: hint})
	}

	_ = filepath.Separator
	return issues, nil
}

// localProvenanceHint returns an actionable message when updates cannot be checked.
// Empty string means provenance is OK for outdated (git / skill-lock / known).
func localProvenanceHint(tr *i18n.T, name, contentPath string, lockIdx *skilllock.Index) string {
	if tr == nil {
		tr = i18n.New(i18n.ZH)
	}
	contentPath = fsutil.Abs(contentPath)
	// git skill root
	if root, ok := fsutil.GitWorkTreeRoot(contentPath); ok && root == contentPath {
		return ""
	}
	// skill-lock
	if lockIdx != nil {
		if _, ok := lockIdx.Lookup(name); ok {
			return ""
		}
	}
	// knownsource
	if _, ok := knownsource.Match(name, contentPath); ok {
		return ""
	}

	// frontmatter source without lock
	if sk, err := skill.Load(contentPath); err == nil && sk != nil {
		src := frontmatterSource(sk)
		if src != "" {
			return tr.S("doctor.prov_source", src, name)
		}
		if ver, ok := sk.Frontmatter["version"].(string); ok && strings.TrimSpace(ver) != "" {
			return tr.S("doctor.prov_version", strings.TrimSpace(ver), name)
		}
	}

	return tr.S("doctor.prov_none", name)
}

func frontmatterSource(sk *skill.Skill) string {
	if sk == nil || sk.Frontmatter == nil {
		return ""
	}
	for _, k := range []string{"source", "repository", "repo", "url", "homepage"} {
		if v, ok := sk.Frontmatter[k].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				return v
			}
		}
	}
	return ""
}
