package tui

import (
	"testing"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/plan"
)

func TestFormatPlanSummary(t *testing.T) {
	cfg := config.Default()
	p := &plan.Plan{
		Summary: plan.Summary{Migrate: 1, Link: 2, Skip: 3, Conflict: 0},
		Actions: []plan.Action{
			{Type: plan.ActionMigrate, Name: "a", Tool: "claude", SourcePath: "/tmp/a"},
			{Type: plan.ActionLink, Name: "b", Tool: "cursor", SourcePath: "/tmp/b"},
			{Type: plan.ActionSkip, Name: "c"},
		},
	}
	lines := formatPlanSummary(cfg, p)
	if len(lines) < 2 {
		t.Fatalf("expected summary + rows, got %v", lines)
	}
}

func TestListWindowKeepsCursorVisible(t *testing.T) {
	// 25 items, page 18: at cursor 0 window is [0,18)
	s, e := listWindow(0, 25, 18)
	if s != 0 || e != 18 {
		t.Fatalf("cursor0: got [%d,%d)", s, e)
	}
	// cursor past first page → window slides
	s, e = listWindow(18, 25, 18)
	if s != 1 || e != 19 {
		t.Fatalf("cursor18: got [%d,%d) want [1,19)", s, e)
	}
	// last item
	s, e = listWindow(24, 25, 18)
	if s != 7 || e != 25 {
		t.Fatalf("cursor24: got [%d,%d) want [7,25)", s, e)
	}
	// short list
	s, e = listWindow(2, 5, 18)
	if s != 0 || e != 5 {
		t.Fatalf("short: got [%d,%d)", s, e)
	}
}
