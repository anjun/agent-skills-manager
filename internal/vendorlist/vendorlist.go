package vendorlist

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/probe"

	"gopkg.in/yaml.v3"
)

type fileDoc struct {
	PathGlobs   []string            `yaml:"path_globs"`
	NamesByTool map[string][]string `yaml:"names_by_tool"`
	NamesGlobal []string            `yaml:"names_global"`
}

// List is the loaded vendor denylist + optional live probes.
type List struct {
	PathGlobs   []string
	NamesByTool map[string]map[string]bool
	NamesGlobal map[string]bool
	Extra       map[string]bool
	Ignore      map[string]bool

	// Grok probe (source of truth when OK).
	GrokProbeOK     bool
	GrokProbeErr    string
	GrokBundledPath map[string]bool
	GrokBundledName map[string]bool
}

func Load(yamlBytes []byte, cfg *config.Config) (*List, error) {
	var f fileDoc
	if len(yamlBytes) > 0 {
		if err := yaml.Unmarshal(yamlBytes, &f); err != nil {
			return nil, err
		}
	}
	l := &List{
		PathGlobs:       f.PathGlobs,
		NamesByTool:     map[string]map[string]bool{},
		NamesGlobal:     map[string]bool{},
		Extra:           map[string]bool{},
		Ignore:          map[string]bool{},
		GrokBundledPath: map[string]bool{},
		GrokBundledName: map[string]bool{},
	}
	for tool, names := range f.NamesByTool {
		m := map[string]bool{}
		for _, n := range names {
			m[strings.ToLower(n)] = true
		}
		l.NamesByTool[tool] = m
	}
	for _, n := range f.NamesGlobal {
		l.NamesGlobal[strings.ToLower(n)] = true
	}
	for _, n := range cfg.VendorExtra {
		l.Extra[strings.ToLower(n)] = true
	}
	for _, n := range cfg.VendorIgnore {
		l.Ignore[strings.ToLower(n)] = true
	}
	return l, nil
}

// AttachGrokProbe runs (or skips) the grok inspect probe and merges results.
func (l *List) AttachGrokProbe(cfg *config.Config) {
	if cfg.NoProbe {
		l.GrokProbeOK = false
		l.GrokProbeErr = "probe disabled (--no-probe)"
		return
	}
	if !cfg.ToolEnabled("grok") {
		l.GrokProbeOK = false
		l.GrokProbeErr = "grok tool disabled"
		return
	}
	g := probe.ProbeGrok()
	l.GrokProbeOK = g.OK
	l.GrokProbeErr = g.Err
	if !g.OK {
		return
	}
	for p := range g.Paths {
		l.GrokBundledPath[p] = true
	}
	for n := range g.Names {
		l.GrokBundledName[n] = true
	}
}

// ClassifyVendor reports whether a skill instance is vendor-owned and why.
// Priority: user ignore → user extra → path globs → grok probe (if OK) → name denylist.
// When grok probe is OK, static names_by_tool["grok"] is NOT used (probe is truth).
func (l *List) ClassifyVendor(tool, name, path string) (vendor bool, reason string) {
	name = strings.ToLower(strings.TrimSpace(name))
	path = fsutil.Abs(path)

	if name != "" && l.Ignore[name] {
		return false, ""
	}
	if name != "" && l.Extra[name] {
		return true, "user vendor-extra"
	}
	if l.IsVendorPath(path) {
		return true, "vendor path"
	}

	if tool == "grok" && l.GrokProbeOK {
		if l.GrokBundledPath[path] {
			return true, "grok inspect (bundled path)"
		}
		// also match if path is under a bundled skill dir
		for bp := range l.GrokBundledPath {
			if path == bp || strings.HasPrefix(path, bp+string(filepath.Separator)) {
				return true, "grok inspect (bundled path)"
			}
		}
		if name != "" && l.GrokBundledName[name] {
			return true, "grok inspect (bundled name)"
		}
		// probe succeeded: do not apply static grok denylist
		return false, ""
	}

	if l.IsVendorName(tool, name) {
		return true, "denylist name"
	}
	return false, ""
}

func (l *List) IsVendorPath(absPath string) bool {
	absPath = filepath.Clean(absPath)
	for _, g := range l.PathGlobs {
		g = fsutil.ExpandHome(g)
		if matchGlob(g, absPath) {
			return true
		}
	}
	sep := string(filepath.Separator)
	if strings.Contains(absPath, sep+".system"+sep) || strings.HasSuffix(absPath, sep+".system") {
		return true
	}
	if strings.Contains(absPath, "vendor_imports") {
		return true
	}
	if strings.Contains(absPath, "bundled"+sep+"skills") {
		return true
	}
	if strings.Contains(absPath, "antigravity") {
		return true
	}
	if strings.Contains(absPath, sep+"plugins"+sep) && strings.Contains(absPath, sep+"skills") {
		return true
	}
	return false
}

func matchGlob(pattern, path string) bool {
	pattern = filepath.Clean(pattern)
	path = filepath.Clean(path)
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+string(filepath.Separator))
	}
	if strings.Contains(pattern, "*") {
		ok, err := filepath.Match(pattern, path)
		return err == nil && ok
	}
	return path == pattern || strings.HasPrefix(path, pattern+string(filepath.Separator))
}

func (l *List) IsVendorName(tool, name string) bool {
	name = strings.ToLower(name)
	if l.Ignore[name] {
		return false
	}
	if l.Extra[name] || l.NamesGlobal[name] {
		return true
	}
	if m, ok := l.NamesByTool[tool]; ok && m[name] {
		return true
	}
	return false
}

// LoadFile reads yaml from path.
func LoadFile(path string, cfg *config.Config) (*List, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(b, cfg)
}
