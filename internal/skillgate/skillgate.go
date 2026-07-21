// Package skillgate enables/disables skills globally or per agent tool.
//
// State lives in ~/.local/state/agent-skills-manager/skill-gates.json.
// Global-off moves content to disabled-skills/ so tools that also read
// ~/.agents/skills cannot see the skill. Per-tool off only removes that
// tool's symlink under its wire root.
package skillgate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/roots"
)

const (
	stateFileName   = "skill-gates.json"
	disabledDirName = "disabled-skills"
	stateVersion    = 1
)

// KnownTools are valid --tool values (wire tools).
var KnownTools = []string{"claude", "cursor", "grok", "codex", "gemini"}

// Entry is gate state for one skill name.
type Entry struct {
	// GlobalOff hides the skill from all tools and removes it from ~/.agents/skills.
	GlobalOff bool `json:"global_off,omitempty"`
	// ToolsOff lists tool ids where the skill must not appear (even if GlobalOff is false).
	ToolsOff []string `json:"tools_off,omitempty"`
}

// File is the on-disk gate document.
type File struct {
	Version int              `json:"version"`
	Skills  map[string]Entry `json:"skills"`
}

// Store reads/writes gates and applies filesystem changes.
type Store struct {
	Path        string // skill-gates.json
	DisabledDir string // disabled-skills/
	AgentsDir   string // ~/.agents/skills
	cfg         *config.Config
	file        *File
}

// Open loads the gate store for cfg (missing file → empty).
func Open(cfg *config.Config) (*Store, error) {
	if err := config.EnsureStateDirs(cfg); err != nil {
		return nil, err
	}
	s := &Store{
		Path:        filepath.Join(cfg.StateDir, stateFileName),
		DisabledDir: filepath.Join(cfg.StateDir, disabledDirName),
		AgentsDir:   fsutil.Abs(cfg.AgentsSkills),
		cfg:         cfg,
		file:        &File{Version: stateVersion, Skills: map[string]Entry{}},
	}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.Path, err)
	}
	if f.Skills == nil {
		f.Skills = map[string]Entry{}
	}
	if f.Version == 0 {
		f.Version = stateVersion
	}
	s.file = &f
	return s, nil
}

// Save writes the gate file.
func (s *Store) Save() error {
	if err := fsutil.EnsureDir(filepath.Dir(s.Path)); err != nil {
		return err
	}
	s.file.Version = stateVersion
	// drop empty entries
	for name, e := range s.file.Skills {
		e.ToolsOff = normalizeTools(e.ToolsOff)
		if !e.GlobalOff && len(e.ToolsOff) == 0 {
			delete(s.file.Skills, name)
			continue
		}
		s.file.Skills[name] = e
	}
	b, err := json.MarshalIndent(s.file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, b, 0o644)
}

// Get returns gate entry for name (zero value if none).
func (s *Store) Get(name string) Entry {
	if s == nil || s.file == nil {
		return Entry{}
	}
	return s.file.Skills[strings.ToLower(strings.TrimSpace(name))]
}

// IsGlobalOff reports whether the skill is fully disabled.
func (s *Store) IsGlobalOff(name string) bool {
	return s.Get(name).GlobalOff
}

// IsToolOff reports whether the skill is hidden for this tool.
// Global off implies all tools off.
func (s *Store) IsToolOff(name, tool string) bool {
	e := s.Get(name)
	if e.GlobalOff {
		return true
	}
	tool = strings.ToLower(strings.TrimSpace(tool))
	for _, t := range e.ToolsOff {
		if t == tool {
			return true
		}
	}
	return false
}

// List returns sorted skill names that have any gate set.
func (s *Store) List() []string {
	if s == nil || s.file == nil {
		return nil
	}
	names := make([]string, 0, len(s.file.Skills))
	for n, e := range s.file.Skills {
		if e.GlobalOff || len(e.ToolsOff) > 0 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// Result describes one disable/enable operation.
type Result struct {
	Name        string   `json:"name"`
	GlobalOff   bool     `json:"global_off"`
	ToolsOff    []string `json:"tools_off,omitempty"`
	Removed     []string `json:"removed,omitempty"`      // paths removed (links)
	Restored    string   `json:"restored,omitempty"`     // agents path after global enable
	Parked      string   `json:"parked,omitempty"`       // disabled-skills path after global disable
	Linked      []string `json:"linked,omitempty"`       // paths created
	Message     string   `json:"message,omitempty"`
	Already     bool     `json:"already,omitempty"`
}

// Disable turns a skill off globally (tools empty) or for listed tools.
func (s *Store) Disable(name string, tools []string) (*Result, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("skill name required")
	}
	key := strings.ToLower(name)
	tools, err := parseTools(tools)
	if err != nil {
		return nil, err
	}
	res := &Result{Name: name}
	e := s.Get(key)

	if len(tools) == 0 {
		// global off
		tr := s.cfg.T()
		if e.GlobalOff {
			res.Already = true
			res.GlobalOff = true
			res.ToolsOff = e.ToolsOff
			res.Message = tr.S("gate.msg.already_global")
			return res, nil
		}
		removed, err := s.removeToolLinks(name, nil) // all wire tools
		if err != nil {
			return nil, err
		}
		res.Removed = removed
		parked, err := s.parkAgents(name)
		if err != nil {
			return nil, err
		}
		res.Parked = parked
		e.GlobalOff = true
		// keep ToolsOff so enable can re-apply per-tool prefs after global on
		s.file.Skills[key] = e
		if err := s.Save(); err != nil {
			return nil, err
		}
		res.GlobalOff = true
		res.ToolsOff = e.ToolsOff
		res.Message = tr.S("gate.msg.global_off")
		return res, nil
	}

	// per-tool off
	tr := s.cfg.T()
	if e.GlobalOff {
		res.GlobalOff = true
		res.ToolsOff = e.ToolsOff
		res.Message = tr.S("gate.msg.global_redundant", name)
		res.Already = true
		return res, nil
	}
	set := map[string]bool{}
	for _, t := range e.ToolsOff {
		set[t] = true
	}
	var newly []string
	for _, t := range tools {
		if !set[t] {
			set[t] = true
			newly = append(newly, t)
		}
	}
	if len(newly) == 0 {
		res.Already = true
		res.ToolsOff = e.ToolsOff
		res.Message = tr.S("gate.msg.already_tools")
		return res, nil
	}
	removed, err := s.removeToolLinks(name, newly)
	if err != nil {
		return nil, err
	}
	res.Removed = removed
	e.ToolsOff = mapKeys(set)
	s.file.Skills[key] = e
	if err := s.Save(); err != nil {
		return nil, err
	}
	res.ToolsOff = e.ToolsOff
	res.Message = tr.S("gate.msg.tools_off", strings.Join(newly, ", "))
	return res, nil
}

// Enable turns a skill on globally (tools empty) or for listed tools.
// Global enable clears all per-tool offs and re-wires every known tool.
func (s *Store) Enable(name string, tools []string) (*Result, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("skill name required")
	}
	key := strings.ToLower(name)
	tools, err := parseTools(tools)
	if err != nil {
		return nil, err
	}
	res := &Result{Name: name}
	e := s.Get(key)

	if len(tools) == 0 {
		// global on: unpark + clear all gates + wire every tool
		if e.GlobalOff || existsDir(filepath.Join(s.DisabledDir, name)) {
			restored, err := s.unparkAgents(name)
			if err != nil {
				return nil, err
			}
			res.Restored = restored
		} else if !existsDir(filepath.Join(s.AgentsDir, name)) {
			return nil, fmt.Errorf("skill %q not found under agents or disabled store", name)
		}
		delete(s.file.Skills, key)
		if err := s.Save(); err != nil {
			return nil, err
		}
		// temporary clear so linkTools does not skip
		s.file.Skills = s.file.Skills // already deleted
		linked, err := s.linkTools(name, KnownTools)
		if err != nil {
			return nil, err
		}
		res.Linked = linked
		res.GlobalOff = false
		res.Message = s.cfg.T().S("gate.msg.global_on")
		return res, nil
	}

	// per-tool on
	tr := s.cfg.T()
	if e.GlobalOff {
		return nil, fmt.Errorf("%s", tr.S("gate.err.global_need_full", name, name))
	}
	if !existsDir(filepath.Join(s.AgentsDir, name)) {
		return nil, fmt.Errorf("canonical skill missing at %s", filepath.Join(s.AgentsDir, name))
	}
	set := map[string]bool{}
	for _, t := range e.ToolsOff {
		set[t] = true
	}
	var cleared []string
	for _, t := range tools {
		if set[t] {
			delete(set, t)
			cleared = append(cleared, t)
		}
	}
	if len(cleared) == 0 {
		res.Message = tr.S("gate.msg.ensure_links")
	} else {
		res.Message = tr.S("gate.msg.tools_on", strings.Join(cleared, ", "))
	}
	e.ToolsOff = mapKeys(set)
	if len(e.ToolsOff) == 0 {
		delete(s.file.Skills, key)
	} else {
		s.file.Skills[key] = e
	}
	if err := s.Save(); err != nil {
		return nil, err
	}
	linked, err := s.linkTools(name, tools)
	if err != nil {
		return nil, err
	}
	res.Linked = linked
	res.ToolsOff = e.ToolsOff
	return res, nil
}

func (s *Store) parkAgents(name string) (string, error) {
	src := filepath.Join(s.AgentsDir, name)
	dst := filepath.Join(s.DisabledDir, name)
	if !existsAny(src) {
		// already parked or never in agents
		if existsDir(dst) {
			return dst, nil
		}
		return "", nil
	}
	if existsAny(dst) {
		// replace old park
		_ = os.RemoveAll(dst)
	}
	if err := fsutil.EnsureDir(s.DisabledDir); err != nil {
		return "", err
	}
	if err := fsutil.MoveDir(src, dst); err != nil {
		return "", fmt.Errorf("park skill to disabled store: %w", err)
	}
	return dst, nil
}

func (s *Store) unparkAgents(name string) (string, error) {
	src := filepath.Join(s.DisabledDir, name)
	dst := filepath.Join(s.AgentsDir, name)
	if existsAny(dst) {
		// already active
		if existsDir(src) {
			// prefer active; drop park duplicate
			_ = os.RemoveAll(src)
		}
		return dst, nil
	}
	if !existsAny(src) {
		return "", fmt.Errorf("no content for %q under agents or disabled store", name)
	}
	if err := fsutil.EnsureDir(s.AgentsDir); err != nil {
		return "", err
	}
	if err := fsutil.MoveDir(src, dst); err != nil {
		return "", fmt.Errorf("restore skill from disabled store: %w", err)
	}
	return dst, nil
}

// removeToolLinks removes name under wire roots. tools nil/empty = all KnownTools.
func (s *Store) removeToolLinks(name string, tools []string) ([]string, error) {
	want := toolSet(tools)
	var removed []string
	for _, wr := range roots.WireRoots(s.cfg) {
		if len(want) > 0 && !want[wr.Tool] {
			continue
		}
		p := filepath.Join(wr.Path, name)
		if !existsAny(p) {
			continue
		}
		// only remove symlink, or real dir that is not the agents truth
		if fsutil.IsSymlink(p) {
			if err := os.Remove(p); err != nil {
				return removed, err
			}
			removed = append(removed, p)
			continue
		}
		// real directory: only remove if it is outside agents (duplicate); never delete agents
		if fsutil.Abs(p) == filepath.Join(s.AgentsDir, name) {
			continue
		}
		// conservative: if real copy, rename aside is too aggressive — remove only if empty of unique content?
		// User wants tool not to see it: rename to .asm-disabled-<name> sibling or remove if symlink was expected.
		// For real dirs under tool root, move to bak under state.
		bak := filepath.Join(s.DisabledDir, "_toolbak", wr.Tool+"__"+name)
		_ = os.RemoveAll(bak)
		if err := fsutil.EnsureDir(filepath.Dir(bak)); err != nil {
			return removed, err
		}
		if err := fsutil.MoveDir(p, bak); err != nil {
			return removed, fmt.Errorf("hide real skill dir %s: %w", p, err)
		}
		removed = append(removed, p+" (moved to "+bak+")")
	}
	return removed, nil
}

func (s *Store) linkTools(name string, tools []string) ([]string, error) {
	target := filepath.Join(s.AgentsDir, name)
	if !existsDir(target) {
		return nil, fmt.Errorf("canonical skill missing at %s — enable global first or install skill", target)
	}
	want := toolSet(tools)
	var linked []string
	for _, wr := range roots.WireRoots(s.cfg) {
		if len(want) > 0 && !want[wr.Tool] {
			continue
		}
		if s.IsToolOff(name, wr.Tool) {
			continue
		}
		linkPath := filepath.Join(wr.Path, name)
		if err := fsutil.EnsureDir(wr.Path); err != nil {
			return linked, err
		}
		// replace existing wrong link/dir
		if fsutil.IsSymlink(linkPath) {
			cur, _ := fsutil.ReadLinkAbs(linkPath)
			if fsutil.Abs(cur) == fsutil.Abs(target) {
				continue // already ok
			}
			_ = os.Remove(linkPath)
		} else if st, err := os.Lstat(linkPath); err == nil && st.IsDir() {
			// do not clobber real dirs silently
			return linked, fmt.Errorf("%s exists as a real directory; remove or migrate it first", linkPath)
		}
		if err := fsutil.SymlinkRel(target, linkPath); err != nil {
			return linked, err
		}
		linked = append(linked, linkPath)
	}
	return linked, nil
}

func parseTools(tools []string) ([]string, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	var out []string
	seen := map[string]bool{}
	for _, t := range tools {
		for _, part := range strings.Split(t, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if part == "" {
				continue
			}
			if !isKnownTool(part) {
				return nil, fmt.Errorf("unknown tool %q (want one of: %s)", part, strings.Join(KnownTools, ", "))
			}
			if !seen[part] {
				seen[part] = true
				out = append(out, part)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func normalizeTools(in []string) []string {
	out, _ := parseTools(in)
	return out
}

func isKnownTool(t string) bool {
	for _, k := range KnownTools {
		if k == t {
			return true
		}
	}
	return false
}

func toolSet(tools []string) map[string]bool {
	m := map[string]bool{}
	for _, t := range tools {
		m[strings.ToLower(t)] = true
	}
	return m
}

func mapKeys(m map[string]bool) []string {
	var out []string
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func existsAny(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func existsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
