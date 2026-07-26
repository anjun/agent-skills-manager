package tui

import (
	"fmt"
	"os"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// ShouldOpen reports whether bare `asm` should launch the interactive menu.
// Uses term.IsTerminal (not ModeCharDevice) so /dev/null and pipes stay non-interactive.
func ShouldOpen(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if cfg.JSON {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Run starts the interactive menu. Returns process exit code.
func Run(cfg *config.Config, vl *vendorlist.List, version string) int {
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "tui: nil config")
		return 1
	}
	deps := &Deps{Cfg: cfg, VL: vl, Version: version}
	m := newModel(deps)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, cfg.T().S("main.error_prefix"), err)
		return 1
	}
	return 0
}
