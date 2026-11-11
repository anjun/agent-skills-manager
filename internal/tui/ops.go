package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anj/agent-skills-manager/internal/apply"
	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/doctor"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/outdated"
	"github.com/anj/agent-skills-manager/internal/plan"
	"github.com/anj/agent-skills-manager/internal/remove"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/selfupdate"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// Deps holds runtime dependencies for the TUI (real packages).
type Deps struct {
	Cfg     *config.Config
	VL      *vendorlist.List
	Version string
}

// quietStdout runs fn while capturing os.Stdout writes (packages print progress).
func quietStdout(fn func() error) (captured string, err error) {
	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", fn()
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	err = fn()
	_ = w.Close()
	os.Stdout = old
	captured = <-done
	_ = r.Close()
	return captured, err
}

func (d *Deps) buildPlan() (*plan.Plan, error) {
	res, err := scan.Run(d.Cfg, d.VL)
	if err != nil {
		return nil, err
	}
	return plan.Build(d.Cfg, res), nil
}

func (d *Deps) applyPlan(p *plan.Plan, strategy string) (*apply.Result, string, error) {
	cfg := *d.Cfg
	cfg.Yes = true
	if strategy != "" {
		cfg.Strategy = strategy
	} else if p != nil && p.Summary.Conflict > 0 && (cfg.Strategy == "" || cfg.Strategy == "interactive") {
		// Conflicts must not re-prompt stdin inside TUI.
		cfg.Strategy = "skip"
	}
	var result *apply.Result
	out, err := quietStdout(func() error {
		var e error
		result, e = apply.Run(&cfg, p, os.Stdin)
		return e
	})
	return result, out, err
}

func (d *Deps) undoLast() (string, error) {
	tx, err := apply.LoadTx(d.Cfg, "last")
	if err != nil {
		return "", err
	}
	out, err := quietStdout(func() error {
		return apply.Undo(d.Cfg, tx)
	})
	return out, err
}

func (d *Deps) runDoctor() ([]doctor.Issue, error) {
	return doctor.Run(d.Cfg, d.VL)
}

func (d *Deps) runOutdated(showAll bool) (*outdated.Result, error) {
	cfg := *d.Cfg
	cfg.ShowAll = showAll
	return outdated.RunOpts(&cfg, d.VL, outdated.RunOptions{})
}

func (d *Deps) updateSkills(names []string) (*outdated.UpdateResult, string, error) {
	cfg := *d.Cfg
	cfg.ShowAll = true
	res, err := outdated.RunOpts(&cfg, d.VL, outdated.RunOptions{})
	if err != nil {
		return nil, "", err
	}
	var ur *outdated.UpdateResult
	var buf bytes.Buffer
	ur = outdated.ApplyUpdates(res, outdated.UpdateOptions{
		Yes:    true,
		Names:  names,
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	})
	return ur, buf.String(), nil
}

func (d *Deps) listRemovableSkills() ([]string, error) {
	names, err := d.listActiveSkills()
	if err != nil {
		return nil, err
	}
	store, err := skillgate.Open(d.Cfg)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, name := range names {
		seen[strings.ToLower(name)] = true
	}
	for _, name := range store.List() {
		if !seen[strings.ToLower(name)] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func (d *Deps) listActiveSkills() ([]string, error) {
	res, err := scan.Run(d.Cfg, d.VL)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, e := range res.Entries {
		if e.Class != scan.ClassCanonical && e.Class != scan.ClassUser {
			continue
		}
		key := strings.ToLower(e.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (d *Deps) removeSkills(names []string) (*remove.Result, string, error) {
	var result *remove.Result
	var buf bytes.Buffer
	result, err := remove.Run(d.Cfg, d.VL, remove.Options{
		Names:  names,
		Yes:    true,
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	})
	return result, buf.String(), err
}

func (d *Deps) listGateEntries() ([]gateRow, error) {
	store, err := skillgate.Open(d.Cfg)
	if err != nil {
		return nil, err
	}
	var rows []gateRow
	for _, n := range store.List() {
		if !store.CanEnable(n) {
			continue
		}
		e := store.Get(n)
		rows = append(rows, gateRow{
			Name:      n,
			GlobalOff: e.GlobalOff,
			ToolsOff:  append([]string(nil), e.ToolsOff...),
		})
	}
	return rows, nil
}

type gateRow struct {
	Name      string
	GlobalOff bool
	ToolsOff  []string
}

func (d *Deps) disableSkill(name string, tools []string) (*skillgate.Result, error) {
	store, err := skillgate.Open(d.Cfg)
	if err != nil {
		return nil, err
	}
	return store.Disable(name, tools)
}

func (d *Deps) enableSkill(name string, tools []string) (*skillgate.Result, error) {
	store, err := skillgate.Open(d.Cfg)
	if err != nil {
		return nil, err
	}
	return store.Enable(name, tools)
}

func (d *Deps) listCanonicalSkills() ([]string, error) {
	// Prefer agents dir listing for disable targets.
	ents, err := os.ReadDir(d.Cfg.AgentsSkills)
	if err != nil {
		if os.IsNotExist(err) {
			return d.listActiveSkills()
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		p := filepath.Join(d.Cfg.AgentsSkills, name)
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err != nil {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return d.listActiveSkills()
	}
	sort.Strings(names)
	return names, nil
}

func (d *Deps) upgradeCheck() (*selfupdate.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return selfupdate.Run(ctx, selfupdate.Options{
		Current:   d.Version,
		CheckOnly: true,
	})
}

func (d *Deps) upgradeApply() (*selfupdate.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), selfupdate.UpgradeTimeout)
	defer cancel()
	return selfupdate.Run(ctx, selfupdate.Options{
		Current:   d.Version,
		CheckOnly: false,
		Yes:       true,
	})
}

func formatPlanSummary(cfg *config.Config, p *plan.Plan) []string {
	tr := cfg.T()
	var lines []string
	lines = append(lines, tr.S("plan.summary", p.Summary.Migrate, p.Summary.Link, p.Summary.Skip, p.Summary.Conflict))
	max := 12
	n := 0
	for _, a := range p.Actions {
		if a.Type == plan.ActionSkip {
			continue
		}
		if n >= max {
			lines = append(lines, tr.S("tui.more_items", len(p.Actions)-n))
			break
		}
		src := fsutil.RelHome(a.SourcePath)
		if a.Type == plan.ActionConflict {
			src = tr.S("plan.candidates", len(a.Candidates))
		}
		lines = append(lines, fmt.Sprintf("  %s  %s  %s  %s",
			tr.ActionLabel(string(a.Type)), a.Name, tr.ToolLabel(a.Tool), src))
		n++
	}
	return lines
}
