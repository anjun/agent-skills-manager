package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusDisabledByDefaultForBuffer(t *testing.T) {
	var buf bytes.Buffer
	r := NewWriter(&buf, false)
	// bytes.Buffer is not a terminal → disabled
	if r.Enabled() {
		t.Fatal("buffer should not enable progress")
	}
	r.Statusf("hello %d", 1)
	if buf.Len() != 0 {
		t.Fatalf("wrote to non-tty: %q", buf.String())
	}
}

func TestJSONModeSilent(t *testing.T) {
	var buf bytes.Buffer
	r := NewWriter(&buf, true)
	r.Statusf("x")
	if buf.Len() != 0 {
		t.Fatalf("json mode must be silent: %q", buf.String())
	}
}

func TestForceEnabledPaintAndDone(t *testing.T) {
	var buf bytes.Buffer
	r := &Reporter{w: &buf, enabled: true}
	r.Statusf("checking 1/2 foo")
	if !strings.Contains(buf.String(), "checking 1/2 foo") {
		t.Fatalf("got %q", buf.String())
	}
	r.Statusf("checking 2/2 bar")
	if !strings.Contains(buf.String(), "checking 2/2 bar") {
		t.Fatalf("got %q", buf.String())
	}
	r.Done()
	// After Done, line should be cleared (spaces + \r); no leftover "checking"
	// content without being wiped — last write is clear sequence.
	if !strings.Contains(buf.String(), "\r") {
		t.Fatalf("expected CR clears, got %q", buf.String())
	}
}

func TestSilentNoPanic(t *testing.T) {
	var r *Reporter
	r.Statusf("x")
	r.Linef("y")
	r.Done()
	Silent().Statusf("z")
}

func TestSanitizeStripsNewlines(t *testing.T) {
	var buf bytes.Buffer
	r := &Reporter{w: &buf, enabled: true}
	r.Statusf("a\nb\rc")
	if strings.Contains(buf.String(), "\n") {
		t.Fatalf("newline leaked: %q", buf.String())
	}
}

func TestCJKWidthPadClears(t *testing.T) {
	var buf bytes.Buffer
	r := &Reporter{w: &buf, enabled: true}
	r.Statusf("检查更新 1/10 long-skill-name…")
	r.Statusf("检查 2/10 x")
	// After shorter line, output should include padding spaces so prior CJK does not ghost.
	if !strings.Contains(buf.String(), "  ") {
		t.Fatalf("expected pad spaces, got %q", buf.String())
	}
	if displayWidth("检查") != 4 {
		t.Fatalf("CJK width: %d", displayWidth("检查"))
	}
}
