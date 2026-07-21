// Package probe discovers tool-owned skills via each product's own APIs.
package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anj/agent-skills-manager/internal/fsutil"
)

const grokProbeTimeout = 12 * time.Second

// GrokBundled is the set of skills Grok itself labels as bundled.
type GrokBundled struct {
	// Paths are absolute skill directories (parent of SKILL.md).
	Paths map[string]bool
	// Names are skill names (lowercased).
	Names map[string]bool
	// OK is true when inspect succeeded and returned data.
	OK bool
	// Err is a soft failure reason when OK is false.
	Err string
	// Binary is the grok executable used.
	Binary string
}

// ProbeGrok runs `grok inspect --json` and collects skills with source.type=bundled.
func ProbeGrok() *GrokBundled {
	out := &GrokBundled{
		Paths: map[string]bool{},
		Names: map[string]bool{},
	}
	bin := findGrokBinary()
	if bin == "" {
		out.Err = "grok binary not found"
		return out
	}
	out.Binary = bin

	ctx, cancel := context.WithTimeout(context.Background(), grokProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "inspect", "--json")
	// Avoid inheriting a huge env that might force interactive VPN wrappers oddly;
	// still pass PATH/HOME so config resolves.
	cmd.Env = filteredEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			out.Err = "grok inspect timed out"
		} else {
			out.Err = fmt.Sprintf("grok inspect: %v", err)
			if s := strings.TrimSpace(stderr.String()); s != "" {
				out.Err += ": " + truncate(s, 200)
			}
		}
		return out
	}

	paths, names, err := ParseGrokInspectJSON(stdout.Bytes())
	if err != nil {
		out.Err = "parse grok inspect: " + err.Error()
		return out
	}
	for _, p := range paths {
		out.Paths[fsutil.Abs(p)] = true
	}
	for _, n := range names {
		out.Names[strings.ToLower(n)] = true
	}
	out.OK = true
	return out
}

// ParseGrokInspectJSON extracts bundled skill dirs and names from inspect JSON.
func ParseGrokInspectJSON(data []byte) (paths []string, names []string, err error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, nil, err
	}
	raw, ok := root["skills"]
	if !ok {
		return nil, nil, fmt.Errorf("missing skills array")
	}
	var skills []grokSkill
	if err := json.Unmarshal(raw, &skills); err != nil {
		return nil, nil, err
	}
	seenPath := map[string]bool{}
	seenName := map[string]bool{}
	for _, s := range skills {
		if s.Source.Type != "bundled" {
			continue
		}
		name := strings.TrimSpace(s.Name)
		if name != "" && !seenName[strings.ToLower(name)] {
			seenName[strings.ToLower(name)] = true
			names = append(names, name)
		}
		p := strings.TrimSpace(s.Source.Path)
		if p == "" {
			continue
		}
		// source.path is usually .../SKILL.md
		dir := p
		base := filepath.Base(p)
		if strings.EqualFold(base, "SKILL.md") || strings.EqualFold(base, "skill.md") {
			dir = filepath.Dir(p)
		}
		dir = fsutil.Abs(dir)
		if !seenPath[dir] {
			seenPath[dir] = true
			paths = append(paths, dir)
		}
	}
	return paths, names, nil
}

type grokSkill struct {
	Name   string `json:"name"`
	Source struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"source"`
}

func findGrokBinary() string {
	if p, err := exec.LookPath("grok"); err == nil {
		return p
	}
	// common install location
	for _, rel := range []string{
		filepath.Join(".grok", "bin", "grok"),
		filepath.Join("bin", "grok"),
	} {
		p := filepath.Join(fsutil.Home(), rel)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	// nested under home .grok/bin only
	p := filepath.Join(fsutil.Home(), ".grok", "bin", "grok")
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return p
	}
	return ""
}

func filteredEnv() []string {
	keep := map[string]bool{
		"PATH": true, "HOME": true, "USER": true, "LOGNAME": true,
		"TMPDIR": true, "XDG_CONFIG_HOME": true, "XDG_STATE_HOME": true,
		"LANG": true, "LC_ALL": true, "TERM": true,
	}
	var out []string
	for _, e := range os.Environ() {
		i := strings.IndexByte(e, '=')
		if i <= 0 {
			continue
		}
		k := e[:i]
		if keep[k] || strings.HasPrefix(k, "GROK_") || strings.HasPrefix(k, "XAI_") {
			out = append(out, e)
		}
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
