package tui

import (
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/i18n"
	"github.com/anj/agent-skills-manager/internal/outdated"
	tea "github.com/charmbracelet/bubbletea"
)

func TestVisibleOutdatedPutsUpdatableFirst(t *testing.T) {
	m := newModel(&Deps{Cfg: config.Default(), Version: "test"})
	m.outdatedReport = &outdated.Result{
		Entries: []outdated.Entry{
			{Name: "current-a", Status: outdated.StatusCurrent},
			{Name: "outdated-b", Status: outdated.StatusOutdated},
			{Name: "current-c", Status: outdated.StatusCurrent},
			{Name: "outdated-d", Status: outdated.StatusOutdated},
			{Name: "unknown-e", Status: outdated.StatusUnknown},
		},
	}
	m.showAll = false
	vis := m.visibleOutdated()
	// unknown hidden; updatable first, others keep relative order
	want := []string{"outdated-b", "outdated-d", "current-a", "current-c"}
	if len(vis) != len(want) {
		t.Fatalf("len=%d want %d: %+v", len(vis), len(want), namesOf(vis))
	}
	for i, name := range want {
		if vis[i].Name != name {
			t.Errorf("pos %d: got %s want %s (full=%v)", i, vis[i].Name, name, namesOf(vis))
		}
	}
}

func namesOf(entries []outdated.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

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
