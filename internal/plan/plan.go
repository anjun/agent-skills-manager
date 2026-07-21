package plan

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/roots"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/skillgate"
)

type ActionType string

const (
	ActionMigrate  ActionType = "migrate"
	ActionLink     ActionType = "link"
	ActionSkip     ActionType = "skip"
	ActionConflict ActionType = "conflict"
)

type Action struct {
	Type       ActionType  `json:"type"`
	Name       string      `json:"name"`
	SourcePath string      `json:"source_path,omitempty"`
	DestPath   string      `json:"dest_path,omitempty"`
	Hash       string      `json:"hash,omitempty"`
	Tool       string      `json:"tool,omitempty"`
	Note       string      `json:"note,omitempty"`
	Candidates []Candidate `json:"candidates,omitempty"`
}

type Candidate struct {
	Path string `json:"path"`
	Tool string `json:"tool"`
	Hash string `json:"hash"`
}

type Plan struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	AgentsDir string    `json:"agents_dir"`
	Actions   []Action  `json:"actions"`
	Summary   Summary   `json:"summary"`
}

type Summary struct {
	Migrate  int `json:"migrate"`
	Link     int `json:"link"`
	Skip     int `json:"skip"`
	Conflict int `json:"conflict"`
}

type contentSrc struct {
	path string // real directory with SKILL.md content
	tool string
	hash string
}

// Build creates a plan from scan results.
func Build(cfg *config.Config, res *scan.Result) *Plan {
	p := &Plan{
		Version:   1,
		CreatedAt: time.Now().UTC(),
		AgentsDir: fsutil.Abs(cfg.AgentsSkills),
	}
	byName := map[string][]scan.Entry{}
	for _, e := range res.Entries {
		byName[e.Name] = append(byName[e.Name], e)
	}
	names := sortedKeys(byName)
	wireRoots := roots.WireRoots(cfg)
	gates, _ := skillgate.Open(cfg) // best-effort; missing state = all enabled

	for _, name := range names {
		entries := byName[name]
		dest := filepath.Join(p.AgentsDir, name)

		// Globally disabled skills must not be re-migrated or re-wired by sync/apply.
		if gates != nil && gates.IsGlobalOff(name) {
			for _, e := range entries {
				p.Actions = append(p.Actions, Action{
					Type: ActionSkip, Name: name, SourcePath: e.Path, Tool: e.Tool, Hash: e.Hash,
					Note: "globally disabled (asm enable " + name + ")",
				})
			}
			continue
		}

		var contents []contentSrc
		seenContentPath := map[string]bool{}

		for _, e := range entries {
			switch e.Class {
			case scan.ClassVendor:
				p.Actions = append(p.Actions, Action{Type: ActionSkip, Name: name, SourcePath: e.Path, Tool: e.Tool, Hash: e.Hash, Note: "vendor"})
				continue
			case scan.ClassProject:
				p.Actions = append(p.Actions, Action{Type: ActionSkip, Name: name, SourcePath: e.Path, Tool: e.Tool, Hash: e.Hash, Note: "project"})
				continue
			case scan.ClassCanonical, scan.ClassUser:
				// ok
			default:
				continue
			}

			// already correctly linked
			if e.Symlink && under(e.Target, p.AgentsDir) {
				p.Actions = append(p.Actions, Action{Type: ActionSkip, Name: name, SourcePath: e.Path, Tool: e.Tool, Hash: e.Hash, Note: "already linked to agents"})
				continue
			}

			contentPath := e.Path
			if e.Symlink && e.Target != "" {
				contentPath = e.Target
			}
			// if it's a real dir (or resolved target) under agents, it's canonical content
			if !seenContentPath[contentPath] {
				seenContentPath[contentPath] = true
				contents = append(contents, contentSrc{path: contentPath, tool: e.Tool, hash: e.Hash})
			}
		}

		if len(contents) == 0 {
			continue
		}

		// group by hash
		byHash := map[string][]contentSrc{}
		for _, c := range contents {
			byHash[c.hash] = append(byHash[c.hash], c)
		}
		if len(byHash) > 1 {
			var cands []Candidate
			for _, c := range contents {
				cands = append(cands, Candidate{Path: c.path, Tool: c.tool, Hash: c.hash})
			}
			p.Actions = append(p.Actions, Action{
				Type: ActionConflict, Name: name, DestPath: dest, Candidates: cands,
				Note: "same name, different content — resolve at apply",
			})
			continue
		}

		// single content hash
		var primary contentSrc
		for _, c := range contents {
			primary = c
			if under(c.path, p.AgentsDir) {
				break
			}
		}
		for _, c := range contents {
			if under(c.path, p.AgentsDir) {
				primary = c
				break
			}
		}

		inAgents := under(primary.path, p.AgentsDir)
		if !inAgents {
			p.Actions = append(p.Actions, Action{
				Type: ActionMigrate, Name: name,
				SourcePath: primary.path, DestPath: dest, Hash: primary.hash, Tool: primary.tool,
				Note: "move content into ~/.agents/skills",
			})
		}

		// replace other real copies (same hash) with links later via apply wire
		for _, c := range contents {
			if c.path == primary.path || under(c.path, p.AgentsDir) {
				continue
			}
			p.Actions = append(p.Actions, Action{
				Type: ActionLink, Name: name,
				SourcePath: c.path, DestPath: dest, Hash: primary.hash, Tool: c.tool,
				Note: "replace duplicate real copy with symlink",
			})
		}

		// wire mainstream tool roots (respect per-tool disable gates)
		for _, wr := range wireRoots {
			if gates != nil && gates.IsToolOff(name, wr.Tool) {
				p.Actions = append(p.Actions, Action{
					Type: ActionSkip, Name: name,
					SourcePath: filepath.Join(wr.Path, name), DestPath: dest, Tool: wr.Tool,
					Note: "disabled for " + wr.Tool + " (asm enable " + name + " --tool " + wr.Tool + ")",
				})
				continue
			}
			linkPath := filepath.Join(wr.Path, name)
			// skip if linkPath is the primary source path and we're migrating (handled after move)
			p.Actions = append(p.Actions, Action{
				Type: ActionLink, Name: name,
				SourcePath: linkPath, DestPath: dest, Tool: wr.Tool,
				Note: "ensure symlink in " + wr.Tool,
			})
		}
	}

	for _, a := range p.Actions {
		switch a.Type {
		case ActionMigrate:
			p.Summary.Migrate++
		case ActionLink:
			p.Summary.Link++
		case ActionSkip:
			p.Summary.Skip++
		case ActionConflict:
			p.Summary.Conflict++
		}
	}
	return p
}

func sortedKeys(m map[string][]scan.Entry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func under(path, root string) bool {
	path = fsutil.Abs(path)
	root = fsutil.Abs(root)
	if path == root {
		return true
	}
	prefix := root + string(filepath.Separator)
	return len(path) > len(prefix) && path[:len(prefix)] == prefix
}
