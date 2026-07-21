// Package skilllock reads vercel-labs/skills lock files for provenance.
package skilllock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anj/agent-skills-manager/internal/fsutil"
)

// Entry is one skill in a lock file (global or project).
type Entry struct {
	Name            string
	Source          string // e.g. owner/repo
	SourceType      string // github, local, ...
	SourceURL       string
	Ref             string
	SkillPath       string // path to SKILL.md or skill folder in repo
	SkillFolderHash string // GitHub tree SHA (global lock v3)
	ComputedHash    string // content hash (project skills-lock.json)
	LockPath        string // which lock file provided this entry
}

// File is a parsed lock file (subset of v3 global / v1 project).
type File struct {
	Version int              `json:"version"`
	Skills  map[string]rawEntry `json:"skills"`
	Path    string           `json:"-"`
}

type rawEntry struct {
	Source          string `json:"source"`
	SourceType      string `json:"sourceType"`
	SourceURL       string `json:"sourceUrl"`
	Ref             string `json:"ref"`
	SkillPath       string `json:"skillPath"`
	SkillFolderHash string `json:"skillFolderHash"`
	ComputedHash    string `json:"computedHash"`
}

// Index maps skill name → first matching lock entry (global preferred).
type Index struct {
	ByName map[string]Entry
}

// LoadIndex loads global ~/.agents/.skill-lock.json (and XDG path if set).
// Project skills-lock.json is not auto-loaded here; use LoadFile for that.
func LoadIndex() *Index {
	idx := &Index{ByName: map[string]Entry{}}
	for _, p := range globalLockCandidates() {
		f, err := LoadFile(p)
		if err != nil || f == nil {
			continue
		}
		for name, e := range f.Skills {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, exists := idx.ByName[name]; exists {
				continue
			}
			idx.ByName[name] = toEntry(name, e, f.Path)
		}
	}
	return idx
}

// LoadFile reads one lock JSON path.
func LoadFile(path string) (*File, error) {
	path = fsutil.Abs(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Skills == nil {
		f.Skills = map[string]rawEntry{}
	}
	f.Path = path
	return &f, nil
}

// Lookup returns lock entry for skill name, if any.
func (idx *Index) Lookup(name string) (Entry, bool) {
	if idx == nil || idx.ByName == nil {
		return Entry{}, false
	}
	e, ok := idx.ByName[strings.TrimSpace(name)]
	return e, ok
}

// CanCheckGitHub reports whether we can compare skillFolderHash via Trees API.
func (e Entry) CanCheckGitHub() bool {
	if !strings.EqualFold(e.SourceType, "github") {
		return false
	}
	if strings.TrimSpace(e.SkillFolderHash) == "" {
		return false
	}
	if strings.TrimSpace(e.SkillPath) == "" && strings.TrimSpace(e.Source) == "" {
		return false
	}
	return ownerRepo(e) != ""
}

// OwnerRepo returns "owner/repo" for GitHub sources.
func ownerRepo(e Entry) string {
	s := strings.TrimSpace(e.Source)
	if s != "" && !strings.Contains(s, "://") && strings.Count(s, "/") == 1 {
		return s
	}
	u := strings.TrimSpace(e.SourceURL)
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")
	// https://github.com/owner/repo
	for _, prefix := range []string{
		"https://github.com/",
		"http://github.com/",
		"git@github.com:",
		"ssh://git@github.com/",
	} {
		if strings.HasPrefix(u, prefix) {
			rest := strings.TrimPrefix(u, prefix)
			parts := strings.Split(rest, "/")
			if len(parts) >= 2 {
				return parts[0] + "/" + parts[1]
			}
		}
	}
	return s
}

func toEntry(name string, r rawEntry, lockPath string) Entry {
	return Entry{
		Name:            name,
		Source:          strings.TrimSpace(r.Source),
		SourceType:      strings.TrimSpace(r.SourceType),
		SourceURL:       strings.TrimSpace(r.SourceURL),
		Ref:             strings.TrimSpace(r.Ref),
		SkillPath:       strings.TrimSpace(r.SkillPath),
		SkillFolderHash: strings.TrimSpace(r.SkillFolderHash),
		ComputedHash:    strings.TrimSpace(r.ComputedHash),
		LockPath:        lockPath,
	}
}

func globalLockCandidates() []string {
	var out []string
	if xdg := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdg != "" {
		out = append(out, filepath.Join(xdg, "skills", ".skill-lock.json"))
	}
	home := fsutil.Home()
	if home != "" {
		out = append(out, filepath.Join(home, ".agents", ".skill-lock.json"))
	}
	return out
}

// RemoveSkills deletes skill names from global .skill-lock.json file(s).
// Returns paths that were modified.
func RemoveSkills(names []string) (modified []string, err error) {
	want := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(strings.ToLower(n))
		if n != "" {
			want[n] = true
		}
	}
	if len(want) == 0 {
		return nil, nil
	}
	for _, p := range globalLockCandidates() {
		f, err := LoadFile(p)
		if err != nil {
			continue
		}
		changed := false
		for name := range f.Skills {
			if want[strings.ToLower(name)] {
				delete(f.Skills, name)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if err := SaveFile(f); err != nil {
			return modified, err
		}
		modified = append(modified, p)
	}
	return modified, nil
}

// SaveFile writes a lock File back to disk (pretty JSON).
func SaveFile(f *File) error {
	if f == nil || f.Path == "" {
		return fmt.Errorf("skilllock: empty file path")
	}
	// Keep version at least 3 for global format
	if f.Version < 3 {
		f.Version = 3
	}
	type outFile struct {
		Version int                  `json:"version"`
		Skills  map[string]rawEntry  `json:"skills"`
	}
	// preserve unknown top-level fields by re-read merge? keep simple: only version+skills
	// re-read original to preserve dismissed / lastSelectedAgents
	raw := map[string]any{}
	if b, err := os.ReadFile(f.Path); err == nil {
		_ = json.Unmarshal(b, &raw)
	}
	raw["version"] = f.Version
	raw["skills"] = f.Skills
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(f.Path, b, 0o644)
}
