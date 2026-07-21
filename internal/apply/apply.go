package apply

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/plan"
)

// Step recorded for undo.
type Step struct {
	Op       string `json:"op"` // move|symlink|remove
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Backup   string `json:"backup,omitempty"` // path we moved aside
	LinkPath string `json:"link_path,omitempty"`
	LinkWas  string `json:"link_was,omitempty"` // previous target or "dir" or "none"
}

// Transaction is a durable apply log.
type Transaction struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Steps     []Step    `json:"steps"`
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
}

// Result of apply.
type Result struct {
	Tx       *Transaction
	Migrated int
	Linked   int
	Skipped  int
	Conflicts int
}

// Run executes a plan.
func Run(cfg *config.Config, p *plan.Plan, stdin *os.File) (*Result, error) {
	if err := config.EnsureStateDirs(cfg); err != nil {
		return nil, err
	}
	if err := fsutil.EnsureDir(cfg.AgentsSkills); err != nil {
		return nil, err
	}

	// resolve conflicts first into concrete migrate/link actions
	actions, err := resolveConflicts(cfg, p.Actions, stdin)
	if err != nil {
		return nil, err
	}

	// confirm
	moves := []plan.Action{}
	for _, a := range actions {
		if a.Type == plan.ActionMigrate {
			moves = append(moves, a)
		}
	}
	tr := cfg.T()
	if !cfg.Yes {
		if !config.IsTTY() {
			return nil, fmt.Errorf("%s", tr.S("apply.need_yes"))
		}
		fmt.Print(tr.S("apply.about", countType(actions, plan.ActionMigrate), countType(actions, plan.ActionLink)))
		fmt.Println()
		if len(moves) > 0 {
			fmt.Println(tr.S("apply.moves"))
			for _, m := range moves {
				fmt.Printf("  %s → %s\n", m.SourcePath, m.DestPath)
			}
		}
		fmt.Print(tr.S("apply.proceed"))
		line, _ := bufio.NewReader(stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			return nil, fmt.Errorf("%s", tr.S("apply.aborted"))
		}
	}

	tx := &Transaction{
		ID:        time.Now().Format("20060102-150405") + "-" + rand4(),
		CreatedAt: time.Now().UTC(),
	}
	res := &Result{Tx: tx}

	for i, a := range actions {
		switch a.Type {
		case plan.ActionSkip:
			res.Skipped++
			continue
		case plan.ActionConflict:
			// should have been resolved
			res.Conflicts++
			continue
		case plan.ActionMigrate:
			fmt.Println(tr.S("apply.step_migrate", i+1, a.Name))
			if err := doMigrate(tx, a); err != nil {
				tx.Error = err.Error()
				_ = saveTx(cfg, tx)
				return res, fmt.Errorf("%s: %w", tr.S("apply.err_migrate", a.Name, tx.ID), err)
			}
			res.Migrated++
		case plan.ActionLink:
			fmt.Println(tr.S("apply.step_link", i+1, a.Name, a.Tool))
			if err := doLink(tx, a); err != nil {
				// missing parent dir is ok skip
				if os.IsNotExist(err) {
					// ensure parent only if wire root exists
					parent := filepath.Dir(a.SourcePath)
					if st, e := os.Stat(parent); e != nil || !st.IsDir() {
						res.Skipped++
						continue
					}
				}
				tx.Error = err.Error()
				_ = saveTx(cfg, tx)
				return res, fmt.Errorf("%s: %w", tr.S("apply.err_link", a.Name, tx.ID), err)
			}
			res.Linked++
		}
	}
	tx.OK = true
	if err := saveTx(cfg, tx); err != nil {
		return res, err
	}
	fmt.Print(tr.S("apply.done", res.Migrated, res.Linked, res.Skipped, tx.ID))
	fmt.Println()
	return res, nil
}

func resolveConflicts(cfg *config.Config, actions []plan.Action, stdin *os.File) ([]plan.Action, error) {
	tr := cfg.T()
	var out []plan.Action
	reader := bufio.NewReader(stdin)
	for _, a := range actions {
		if a.Type != plan.ActionConflict {
			out = append(out, a)
			continue
		}
		if len(a.Candidates) == 0 {
			out = append(out, plan.Action{Type: plan.ActionSkip, Name: a.Name, Note: "empty conflict"})
			continue
		}
		// strategy non-interactive
		if cfg.Strategy != "" && cfg.Strategy != "interactive" {
			resolved, err := applyStrategy(cfg, a)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved...)
			continue
		}
		if !config.IsTTY() {
			return nil, fmt.Errorf("%s", tr.S("apply.conflict_need", a.Name))
		}
		fmt.Print(tr.S("apply.conflict_title", a.Name))
		fmt.Println()
		for i, c := range a.Candidates {
			fmt.Println(tr.S("apply.conflict_cand", i+1, c.Path, c.Tool, short(c.Hash)))
		}
		fmt.Print(tr.S("apply.conflict_prompt"))
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		switch {
		case line == "q":
			return nil, fmt.Errorf("%s", tr.S("apply.conflict_abort", a.Name))
		case line == "s":
			out = append(out, plan.Action{Type: plan.ActionSkip, Name: a.Name, Note: "conflict skipped"})
		case line == "r":
			// primary = first, others renamed
			primary := a.Candidates[0]
			dest := filepath.Join(cfg.AgentsSkills, a.Name)
			out = append(out, plan.Action{Type: plan.ActionMigrate, Name: a.Name, SourcePath: primary.Path, DestPath: dest, Hash: primary.Hash, Tool: primary.Tool})
			for i := 1; i < len(a.Candidates); i++ {
				c := a.Candidates[i]
				newName := a.Name + "@" + c.Tool
				nd := filepath.Join(cfg.AgentsSkills, newName)
				out = append(out, plan.Action{Type: plan.ActionMigrate, Name: newName, SourcePath: c.Path, DestPath: nd, Hash: c.Hash, Tool: c.Tool, Note: "renamed from conflict"})
			}
		case strings.HasPrefix(line, "k") || isDigit(line):
			idx := 0
			if isDigit(line) {
				fmt.Sscanf(line, "%d", &idx)
				idx--
			} else if len(line) > 1 {
				fmt.Sscanf(line[1:], "%d", &idx)
				idx--
			}
			if idx < 0 || idx >= len(a.Candidates) {
				idx = 0
			}
			c := a.Candidates[idx]
			dest := filepath.Join(cfg.AgentsSkills, a.Name)
			out = append(out, plan.Action{Type: plan.ActionMigrate, Name: a.Name, SourcePath: c.Path, DestPath: dest, Hash: c.Hash, Tool: c.Tool})
		default:
			// keep first
			c := a.Candidates[0]
			dest := filepath.Join(cfg.AgentsSkills, a.Name)
			out = append(out, plan.Action{Type: plan.ActionMigrate, Name: a.Name, SourcePath: c.Path, DestPath: dest, Hash: c.Hash, Tool: c.Tool})
		}
	}
	return out, nil
}

func applyStrategy(cfg *config.Config, a plan.Action) ([]plan.Action, error) {
	switch cfg.Strategy {
	case "skip":
		return []plan.Action{{Type: plan.ActionSkip, Name: a.Name, Note: "conflict skipped by strategy"}}, nil
	case "keep-agents":
		for _, c := range a.Candidates {
			if c.Tool == "agents" || strings.Contains(c.Path, ".agents/skills") {
				dest := filepath.Join(cfg.AgentsSkills, a.Name)
				return []plan.Action{{Type: plan.ActionMigrate, Name: a.Name, SourcePath: c.Path, DestPath: dest, Hash: c.Hash, Tool: c.Tool}}, nil
			}
		}
		return []plan.Action{{Type: plan.ActionSkip, Name: a.Name, Note: "keep-agents: no agents candidate"}}, nil
	case "keep-newest":
		// pick max mtime
		best := a.Candidates[0]
		bestT := time.Time{}
		for _, c := range a.Candidates {
			st, err := os.Stat(c.Path)
			if err != nil {
				continue
			}
			if st.ModTime().After(bestT) {
				bestT = st.ModTime()
				best = c
			}
		}
		dest := filepath.Join(cfg.AgentsSkills, a.Name)
		return []plan.Action{{Type: plan.ActionMigrate, Name: a.Name, SourcePath: best.Path, DestPath: dest, Hash: best.Hash, Tool: best.Tool}}, nil
	default:
		return nil, fmt.Errorf("%s", cfg.T().S("apply.unknown_strategy", cfg.Strategy))
	}
}

func doMigrate(tx *Transaction, a plan.Action) error {
	src := a.SourcePath
	dst := a.DestPath
	if fsutil.Abs(src) == fsutil.Abs(dst) {
		return nil
	}
	// if dst exists, backup aside
	if st, err := os.Lstat(dst); err == nil {
		bak := dst + ".bak-" + tx.ID
		if err := os.Rename(dst, bak); err != nil {
			return err
		}
		tx.Steps = append(tx.Steps, Step{Op: "move", From: dst, To: bak, Backup: bak})
		_ = st
	}
	// if src is under a tool root and is the only copy — move
	if err := fsutil.MoveDir(src, dst); err != nil {
		return err
	}
	tx.Steps = append(tx.Steps, Step{Op: "move", From: src, To: dst})
	return nil
}

func doLink(tx *Transaction, a plan.Action) error {
	linkPath := a.SourcePath
	target := a.DestPath
	parent := filepath.Dir(linkPath)
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		// create wire root if it's a known skills dir name
		if err := fsutil.EnsureDir(parent); err != nil {
			return err
		}
	}
	// if already correct symlink, skip
	if fsutil.IsSymlink(linkPath) {
		cur, err := fsutil.ReadLinkAbs(linkPath)
		if err == nil && fsutil.Abs(cur) == fsutil.Abs(target) {
			return nil
		}
		was, _ := os.Readlink(linkPath)
		if err := os.Remove(linkPath); err != nil {
			return err
		}
		tx.Steps = append(tx.Steps, Step{Op: "remove", LinkPath: linkPath, LinkWas: was})
	} else if st, err := os.Lstat(linkPath); err == nil {
		// real dir — move aside then link
		bak := linkPath + ".bak-" + tx.ID
		if err := os.Rename(linkPath, bak); err != nil {
			return err
		}
		tx.Steps = append(tx.Steps, Step{Op: "move", From: linkPath, To: bak, Backup: bak})
		_ = st
	}
	if err := fsutil.SymlinkRel(target, linkPath); err != nil {
		return err
	}
	tx.Steps = append(tx.Steps, Step{Op: "symlink", LinkPath: linkPath, To: target})
	return nil
}

func saveTx(cfg *config.Config, tx *Transaction) error {
	path := filepath.Join(cfg.StateDir, "transactions", tx.ID+".json")
	b, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadTx loads a transaction by id or "last".
func LoadTx(cfg *config.Config, id string) (*Transaction, error) {
	dir := filepath.Join(cfg.StateDir, "transactions")
	if id == "last" || id == "" {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		var newest string
		var newestT time.Time
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(newestT) {
				newestT = info.ModTime()
				newest = e.Name()
			}
		}
		if newest == "" {
			return nil, fmt.Errorf("%s", cfg.T().S("apply.no_tx"))
		}
		id = strings.TrimSuffix(newest, ".json")
	}
	b, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var tx Transaction
	if err := json.Unmarshal(b, &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// Undo reverses a transaction (best-effort reverse order).
func Undo(cfg *config.Config, tx *Transaction) error {
	for i := len(tx.Steps) - 1; i >= 0; i-- {
		s := tx.Steps[i]
		switch s.Op {
		case "symlink":
			_ = os.Remove(s.LinkPath)
		case "remove":
			// cannot restore easily without was path
			if s.LinkWas != "" && s.LinkPath != "" {
				_ = os.Symlink(s.LinkWas, s.LinkPath)
			}
		case "move":
			// reverse rename
			if s.To != "" && s.From != "" {
				// if From is free, move To back to From
				_ = os.RemoveAll(s.From) // if dangling link
				if err := os.Rename(s.To, s.From); err != nil {
					// try copy back
					fmt.Fprintf(os.Stderr, cfg.T().S("apply.undo_warn"), s.From, s.To, err)
				}
			}
		}
	}
	fmt.Println(cfg.T().S("apply.undo_done", tx.ID))
	return nil
}

func countType(actions []plan.Action, t plan.ActionType) int {
	n := 0
	for _, a := range actions {
		if a.Type == t {
			n++
		}
	}
	return n
}

func short(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

func isDigit(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func rand4() string {
	return fmt.Sprintf("%04x", time.Now().UnixNano()&0xffff)
}
