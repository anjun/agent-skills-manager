package tui

import (
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMainMenuStableIDs(t *testing.T) {
	m := newModel(&Deps{Cfg: config.Default(), Version: "test"})
	items := m.mainMenu()
	want := []string{"sync", "outdated", "doctor", "manage", "about"}
	if len(items) != len(want) {
		t.Fatalf("len=%d want %d", len(items), len(want))
	}
	for i, id := range want {
		if items[i].id != id {
			t.Errorf("item %d: got %s want %s", i, items[i].id, id)
		}
	}
}

func TestHomeNavigation(t *testing.T) {
	m := newModel(&Deps{Cfg: config.Default(), Version: "test"})
	m.screen = screenHome
	m.cursor = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.cursor != 1 {
		t.Fatalf("down: cursor=%d", m.cursor)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = next.(model)
	if m.cursor != 0 {
		t.Fatalf("k up: cursor=%d", m.cursor)
	}
	// manage is index 3 → submenu (no async)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = next.(model)
	if m.screen != screenManage {
		t.Fatalf("after 4: screen=%v want manage", m.screen)
	}
}

func TestConfirmNoReturns(t *testing.T) {
	m := newModel(&Deps{Cfg: config.Default(), Version: "test"})
	m.screen = screenConfirm
	m.confirmYes = afterSyncApply
	m.cursor = 1 // no
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.screen != screenSyncPreview {
		t.Fatalf("want sync preview, got %v", m.screen)
	}
}

func TestShouldOpen(t *testing.T) {
	cfg := config.Default()
	cfg.JSON = true
	if ShouldOpen(cfg) {
		t.Fatal("--json must not open TUI")
	}
}

func TestViewHomeRendersBothLangs(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ZH, i18n.EN} {
		cfg := config.Default()
		cfg.Lang = lang
		m := newModel(&Deps{Cfg: cfg, Version: "0.0.0-test"})
		m.screen = screenHome
		out := m.View()
		if out == "" {
			t.Fatalf("empty view for %s", lang)
		}
		if !containsAny(out, "asm") {
			t.Fatalf("expected asm title in %s view", lang)
		}
	}
}

func containsAny(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (stringIndex(s, sub) >= 0)))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
