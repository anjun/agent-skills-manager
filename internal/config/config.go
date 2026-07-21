package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/i18n"
)

const (
	// AppName is the product / binary name.
	AppName = "agent-skills-manager"
	// AppAlias is the short CLI alias.
	AppAlias = "asm"
	// LegacyAppName is the previous product name (state migration).
	LegacyAppName = "agentskills-migrate"
)

// Config holds runtime options.
type Config struct {
	Home         string
	AgentsSkills string // ~/.agents/skills
	StateDir     string // ~/.local/state/agent-skills-manager
	ConfigDir    string // ~/.config/agent-skills-manager
	ExtraRoots   []string
	ProjectRoots []string // explicit project roots (repo dirs or skill dirs)
	NoProject    bool     // skip auto project discovery from cwd
	ShowAll      bool     // list vendor rows in human scan output
	NoProbe      bool     // skip live tool probes (e.g. grok inspect)
	Lang         i18n.Lang
	DisabledTools map[string]bool // tool id -> disabled
	VendorExtra   []string        // extra vendor names
	VendorIgnore  []string        // names force NOT vendor
	WireTools     []string        // tools to symlink after migrate
	Strategy      string          // skip|keep-agents|keep-newest|interactive
	Yes           bool
	JSON          bool
	OnlyTools     []string
	ExcludeTools  []string
}

// T returns the UI message catalog for the selected language.
func (c *Config) T() *i18n.T {
	return i18n.New(c.Lang)
}

func Default() *Config {
	home := fsutil.Home()
	return &Config{
		Home:          home,
		AgentsSkills:  filepath.Join(home, ".agents", "skills"),
		StateDir:      filepath.Join(home, ".local", "state", AppName),
		ConfigDir:     filepath.Join(home, ".config", AppName),
		WireTools:     []string{"claude", "cursor", "grok", "codex", "gemini"},
		Strategy:      "interactive",
		Lang:          i18n.FromEnv(), // default zh unless env overrides
		DisabledTools: map[string]bool{},
	}
}

// ToolEnabled reports whether a tool adapter is active.
func (c *Config) ToolEnabled(id string) bool {
	if c.DisabledTools[id] {
		return false
	}
	if len(c.OnlyTools) > 0 {
		ok := false
		for _, t := range c.OnlyTools {
			if t == id {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, t := range c.ExcludeTools {
		if t == id {
			return false
		}
	}
	return true
}

// ParseCSV splits comma-separated tool ids.
func ParseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// EnsureStateDirs creates state dirs and migrates legacy state if needed.
func EnsureStateDirs(c *Config) error {
	if err := migrateLegacyState(c); err != nil {
		// non-fatal: log via return only when hard fail; migration soft-fails
		_ = err
	}
	if err := fsutil.EnsureDir(c.StateDir); err != nil {
		return err
	}
	return fsutil.EnsureDir(filepath.Join(c.StateDir, "transactions"))
}

// MigrateLegacyState moves ~/.local/state/agentskills-migrate → new StateDir when needed.
// Exported for tests.
func MigrateLegacyState(c *Config) error {
	return migrateLegacyState(c)
}

func migrateLegacyState(c *Config) error {
	home := c.Home
	if home == "" {
		home = fsutil.Home()
	}
	legacy := filepath.Join(home, ".local", "state", LegacyAppName)
	dest := c.StateDir
	if dest == "" {
		dest = filepath.Join(home, ".local", "state", AppName)
	}
	// already migrated marker
	if samePath(legacy, dest) {
		return nil
	}
	lst, err := os.Stat(legacy)
	if err != nil || !lst.IsDir() {
		return nil // nothing to migrate
	}
	// if dest already has content, leave legacy in place
	if ents, err := os.ReadDir(dest); err == nil && len(ents) > 0 {
		return nil
	}
	if err := fsutil.EnsureDir(filepath.Dir(dest)); err != nil {
		return err
	}
	// rename whole tree
	if err := os.Rename(legacy, dest); err != nil {
		// cross-device fallback: copy+leave note
		return fmt.Errorf("migrate state %s -> %s: %w", legacy, dest, err)
	}
	return nil
}

func samePath(a, b string) bool {
	return fsutil.Abs(a) == fsutil.Abs(b)
}

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
