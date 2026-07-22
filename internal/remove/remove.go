// Package remove uninstalls user/canonical skills from all known global roots.
package remove

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/roots"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/skilllock"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Options for Run.
type Options struct {
	// Names to remove (required).
	Names []string
	// Yes skips confirmation.
	Yes bool
	// KeepLock does not edit ~/.agents/.skill-lock.json.
	KeepLock bool
	// IncludeProject also removes ClassProject entries (default false).
	IncludeProject bool
	Stdin  io.Reader
	Stdout io.Writer
}

// PlanItem is one filesystem path that will be removed.
type PlanItem struct {
	Name  string     `json:"name"`
	Path  string     `json:"path"`
	Class scan.Class `json:"class"`
	Tool  string     `json:"tool"`
	Kind  string     `json:"kind"` // symlink | directory
}

// Result of remove.
type Result struct {
	Planned      []PlanItem `json:"planned"`
	Removed      []string   `json:"removed"`
	Failed       []string   `json:"failed,omitempty"`
	Skipped      []string   `json:"skipped,omitempty"` // vendor / project / truly missing
	// LockOnly: requested names absent from disk but present in skill-lock (orphan entries).
	LockOnly     []string `json:"lock_only,omitempty"`
	LockModified []string `json:"lock_modified,omitempty"`
	Aborted      bool     `json:"aborted,omitempty"`
}

// Plan builds a removal plan without deleting.
func Plan(cfg *config.Config, vl *vendorlist.List, names []string, includeProject bool) (*Result, error) {
	want := normalizeNames(names)
	if len(want) == 0 {
		return nil, fmt.Errorf("specify at least one skill name")
	}
	res, err := scan.Run(cfg, vl)
	if err != nil {
		return nil, err
	}

	out := &Result{}
	seenPath := map[string]bool{}
	foundName := map[string]bool{}

	for _, e := range res.Entries {
		key := strings.ToLower(e.Name)
		if !want[key] {
			continue
		}
		foundName[key] = true
		switch e.Class {
		case scan.ClassVendor:
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s (%s vendor, never remove)", e.Name, e.Tool))
			continue
		case scan.ClassProject:
			if !includeProject {
				out.Skipped = append(out.Skipped, fmt.Sprintf("%s (project; pass --project to include)", e.Name))
				continue
			}
		case scan.ClassCanonical, scan.ClassUser:
			// ok
		default:
			continue
		}
		p := fsutil.Abs(e.Path)
		if seenPath[p] {
			continue
		}
		seenPath[p] = true
		kind := "directory"
		if e.Symlink || fsutil.IsSymlink(p) {
			kind = "symlink"
		}
		out.Planned = append(out.Planned, PlanItem{
			Name: e.Name, Path: p, Class: e.Class, Tool: e.Tool, Kind: kind,
		})
	}

	// Always probe <root>/<name> on global roots (scan skips symlink-as-dir entries).
	for name := range want {
		for _, r := range roots.DefaultRoots(cfg) {
			if r.Kind == "vendor" {
				continue
			}
			if r.Kind == "project" && !includeProject {
				continue
			}
			cand := filepath.Join(r.Path, name)
			st, err := os.Lstat(cand)
			if err != nil {
				continue
			}
			p := fsutil.Abs(cand)
			if seenPath[p] {
				continue
			}
			// skip if this path is under a vendor-classified tree (extra safety)
			if r.Kind == "vendor" {
				continue
			}
			seenPath[p] = true
			foundName[name] = true
			kind := "directory"
			if st.Mode()&os.ModeSymlink != 0 {
				kind = "symlink"
			}
			class := scan.ClassUser
			switch r.Kind {
			case "canonical":
				class = scan.ClassCanonical
			case "project":
				class = scan.ClassProject
			}
			out.Planned = append(out.Planned, PlanItem{
				Name: name, Path: p, Class: class, Tool: r.Tool, Kind: kind,
			})
		}
	}

	// Orphan skill-lock entries: doctor recommends `asm remove <name>` even when nothing is on disk.
	lockIdx := skilllock.LoadIndex()
	for name := range want {
		if foundName[name] {
			continue
		}
		if lockIdx.Contains(name) {
			out.LockOnly = append(out.LockOnly, name)
			continue
		}
		out.Skipped = append(out.Skipped, name+" (not found in global roots)")
	}
	sort.Strings(out.LockOnly)

	sort.Slice(out.Planned, func(i, j int) bool {
		if out.Planned[i].Name != out.Planned[j].Name {
			return out.Planned[i].Name < out.Planned[j].Name
		}
		return out.Planned[i].Path < out.Planned[j].Path
	})
	return out, nil
}

// Run plans and executes removal.
func Run(cfg *config.Config, vl *vendorlist.List, opt Options) (*Result, error) {
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Stdin == nil {
		opt.Stdin = os.Stdin
	}

	plan, err := Plan(cfg, vl, opt.Names, opt.IncludeProject)
	if err != nil {
		return nil, err
	}

	// Disk paths and/or orphan skill-lock entries are both uninstall targets.
	// KeepLock means lock-only names are not actionable.
	lockCleanup := !opt.KeepLock && len(plan.LockOnly) > 0
	if len(plan.Planned) == 0 && !lockCleanup {
		fmt.Fprintln(opt.Stdout, "没有可卸载的全局 skill。")
		for _, s := range plan.Skipped {
			fmt.Fprintf(opt.Stdout, "  skip: %s\n", s)
		}
		// With --keep-lock, orphan lock names would have been LockOnly; surface them.
		if opt.KeepLock {
			for _, n := range plan.LockOnly {
				fmt.Fprintf(opt.Stdout, "  skip: %s (skill-lock only; omit --keep-lock to clean)\n", n)
			}
		}
		return plan, nil
	}

	if len(plan.Planned) > 0 {
		fmt.Fprintln(opt.Stdout, "将删除以下路径：")
		for _, it := range plan.Planned {
			fmt.Fprintf(opt.Stdout, "  [%s] %s  %s  (%s)\n", it.Kind, it.Name, fsutil.RelHome(it.Path), it.Tool)
		}
	}
	if lockCleanup {
		if len(plan.Planned) == 0 {
			fmt.Fprintln(opt.Stdout, "磁盘无此 skill，将清理 skill-lock 孤立条目：")
		} else {
			fmt.Fprintln(opt.Stdout, "并清理 skill-lock 孤立条目：")
		}
		for _, n := range plan.LockOnly {
			fmt.Fprintf(opt.Stdout, "  [skill-lock] %s\n", n)
		}
	}
	for _, s := range plan.Skipped {
		fmt.Fprintf(opt.Stdout, "  skip: %s\n", s)
	}
	// Disk paths also try to drop matching lock entries (npx skills provenance).
	if !opt.KeepLock && len(plan.Planned) > 0 {
		fmt.Fprintln(opt.Stdout, "  并尝试从 ~/.agents/.skill-lock.json 移除同名条目")
	}

	if !opt.Yes {
		// Non-interactive: require -y (same as apply)
		if !config.IsTTY() {
			return nil, fmt.Errorf("non-TTY: pass -y to confirm remove")
		}
		fmt.Fprint(opt.Stdout, "确认卸载？[y/N] ")
		line, _ := bufio.NewReader(opt.Stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			plan.Aborted = true
			fmt.Fprintln(opt.Stdout, "已取消。")
			return plan, nil
		}
	}

	// Delete symlinks first, then directories (so we don't break mid-walk assumptions)
	var links, dirs []PlanItem
	for _, it := range plan.Planned {
		if it.Kind == "symlink" {
			links = append(links, it)
		} else {
			dirs = append(dirs, it)
		}
	}
	for _, it := range append(links, dirs...) {
		if err := os.RemoveAll(it.Path); err != nil {
			plan.Failed = append(plan.Failed, it.Path+": "+err.Error())
			continue
		}
		plan.Removed = append(plan.Removed, it.Path)
	}

	if !opt.KeepLock {
		// Use all requested names so disk hits and lock-only orphans both leave the lock file.
		mod, err := skilllock.RemoveSkills(opt.Names)
		if err != nil {
			plan.Failed = append(plan.Failed, "skill-lock: "+err.Error())
		} else {
			plan.LockModified = mod
		}
	}
	return plan, nil
}

func normalizeNames(names []string) map[string]bool {
	want := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(strings.ToLower(n))
		n = strings.ReplaceAll(n, "_", "-")
		n = strings.ReplaceAll(n, " ", "-")
		if n != "" {
			want[n] = true
		}
	}
	return want
}
