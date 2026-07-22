// Package progress writes ephemeral status lines so slow CLI commands don't look stuck.
// Output goes to stderr and only when stderr is a TTY and JSON mode is off.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Reporter paints single-line status updates (overwritten with \r).
// Safe for concurrent use; methods are no-ops when disabled.
type Reporter struct {
	w       io.Writer
	enabled bool

	mu      sync.Mutex
	active  bool
	lastW   int // display width of last status (runes approx)
}

// New returns a reporter. Disabled when jsonMode is true or stderr is not a TTY.
func New(jsonMode bool) *Reporter {
	return NewWriter(os.Stderr, jsonMode)
}

// NewWriter is like New but with an explicit writer (tests).
func NewWriter(w io.Writer, jsonMode bool) *Reporter {
	if w == nil {
		w = io.Discard
	}
	return &Reporter{
		w:       w,
		enabled: !jsonMode && isTerminal(w),
	}
}

// Silent returns a permanently disabled reporter.
func Silent() *Reporter {
	return &Reporter{w: io.Discard, enabled: false}
}

// Enabled reports whether status lines will be shown.
func (r *Reporter) Enabled() bool {
	return r != nil && r.enabled
}

// Statusf draws/updates the current status line (no trailing newline).
func (r *Reporter) Statusf(format string, args ...any) {
	if r == nil || !r.enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = sanitize(msg)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paint(msg)
}

// Linef prints a permanent line (ends with newline), clearing any status first.
func (r *Reporter) Linef(format string, args ...any) {
	if r == nil || !r.enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = sanitize(msg)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLocked()
	fmt.Fprintln(r.w, msg)
}

// Done clears the status line so subsequent stdout is clean.
func (r *Reporter) Done() {
	if r == nil || !r.enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearLocked()
}

func (r *Reporter) paint(msg string) {
	// Clear residual chars from a longer previous line.
	pad := ""
	w := displayWidth(msg)
	if r.lastW > w {
		pad = strings.Repeat(" ", r.lastW-w)
	}
	fmt.Fprintf(r.w, "\r%s%s", msg, pad)
	// Keep cursor after visible text for nicer partial redraws.
	if pad != "" {
		fmt.Fprintf(r.w, "\r%s", msg)
	}
	r.lastW = w
	r.active = true
}

func (r *Reporter) clearLocked() {
	if !r.active && r.lastW == 0 {
		return
	}
	if r.lastW > 0 {
		fmt.Fprintf(r.w, "\r%s\r", strings.Repeat(" ", r.lastW))
	} else {
		fmt.Fprint(r.w, "\r")
	}
	r.lastW = 0
	r.active = false
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	// Keep one line short enough for small terminals.
	const max = 96
	if displayWidth(s) > max {
		s = truncateWidth(s, max-1) + "…"
	}
	return s
}

// displayWidth approximates terminal columns (CJK / fullwidth ≈ 2).
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	// Common wide ranges: CJK, fullwidth forms, Hangul, etc.
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case r < 0x1100:
		return 1
	case r <= 0x115f || // Hangul Jamo
		r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf) || // CJK ... Yi
		(r >= 0xac00 && r <= 0xd7a3) || // Hangul Syllables
		(r >= 0xf900 && r <= 0xfaff) || // CJK Compatibility Ideographs
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) || // Fullwidth Forms
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1f9ff) || // Emoji block (approx)
		(r >= 0x20000 && r <= 0x2fffd) ||
		(r >= 0x30000 && r <= 0x3fffd):
		return 2
	default:
		return 1
	}
}

func truncateWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if displayWidth(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		rw := runeWidth(r)
		if n+rw > max {
			break
		}
		b.WriteRune(r)
		n += rw
	}
	return b.String()
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
