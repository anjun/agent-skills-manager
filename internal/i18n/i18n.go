// Package i18n provides human-facing CLI strings (default Chinese, optional English).
// Machine JSON fields stay English regardless of Lang.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a UI language code.
type Lang string

const (
	ZH Lang = "zh"
	EN Lang = "en"
)

// Parse normalizes user/env input to Lang. Empty and unknown → ZH (project default).
func Parse(s string) Lang {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "", "zh", "zh-cn", "zh-hans", "cn", "chinese", "中文":
		return ZH
	case "en", "en-us", "en-gb", "english":
		return EN
	default:
		if strings.HasPrefix(s, "en") {
			return EN
		}
		if strings.HasPrefix(s, "zh") {
			return ZH
		}
		return ZH
	}
}

// FromEnv reads language from environment; default ZH.
// Supported: ASM_LANG, AGENT_SKILLS_MANAGER_LANG, AGENTS_LANG,
// and legacy AGENTSKILLS_MIGRATE_LANG.
func FromEnv() Lang {
	for _, k := range []string{
		"ASM_LANG",
		"AGENT_SKILLS_MANAGER_LANG",
		"AGENTS_LANG",
		"AGENTSKILLS_MIGRATE_LANG", // legacy
	} {
		if v := os.Getenv(k); v != "" {
			return Parse(v)
		}
	}
	return ZH
}

// T is a message catalog for one language.
type T struct {
	Lang Lang
	m    map[string]string
}

// New returns the catalog for lang (fallback ZH).
func New(lang Lang) *T {
	if lang == EN {
		return &T{Lang: EN, m: en}
	}
	return &T{Lang: ZH, m: zh}
}

// S looks up key; optional fmt args. Missing keys fall back to English then key.
func (t *T) S(key string, args ...any) string {
	s, ok := t.m[key]
	if !ok || s == "" {
		s, ok = en[key]
	}
	if !ok {
		s = key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

// IsEN reports English UI.
func (t *T) IsEN() bool { return t.Lang == EN }

// ToolLabel maps tool id for display.
func (t *T) ToolLabel(tool string) string {
	switch tool {
	case "agents":
		return t.S("tool.agents")
	case "project":
		return t.S("tool.project")
	case "extra":
		return t.S("tool.extra")
	default:
		return tool
	}
}

// ActionLabel for plan action types.
func (t *T) ActionLabel(typ string) string {
	switch typ {
	case "migrate":
		return t.S("action.migrate")
	case "link":
		return t.S("action.link")
	case "skip":
		return t.S("action.skip")
	case "conflict":
		return t.S("action.conflict")
	default:
		return typ
	}
}

// SeverityLabel for doctor.
func (t *T) SeverityLabel(s string) string {
	switch s {
	case "error":
		return t.S("sev.error")
	case "warn":
		return t.S("sev.warn")
	case "info":
		return t.S("sev.info")
	default:
		return s
	}
}

// NoteLabel translates known plan notes (English keys from plan package).
func (t *T) NoteLabel(note string) string {
	if note == "" {
		return ""
	}
	if zh := t.S("note." + note); zh != "note."+note {
		return zh
	}
	// prefix patterns
	const ensure = "ensure symlink in "
	if strings.HasPrefix(note, ensure) {
		return t.S("note.ensure_symlink", strings.TrimPrefix(note, ensure))
	}
	const fixWire = "fix wire link in "
	if strings.HasPrefix(note, fixWire) {
		return t.S("note.fix_wire_link", strings.TrimPrefix(note, fixWire))
	}
	return note
}
