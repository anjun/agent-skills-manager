package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/anj/agent-skills-manager/embedded"
	"github.com/anj/agent-skills-manager/internal/apply"
	"github.com/anj/agent-skills-manager/internal/config"
	"github.com/anj/agent-skills-manager/internal/doctor"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/i18n"
	"github.com/anj/agent-skills-manager/internal/outdated"
	"github.com/anj/agent-skills-manager/internal/plan"
	"github.com/anj/agent-skills-manager/internal/progress"
	"github.com/anj/agent-skills-manager/internal/remove"
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/selfupdate"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	"github.com/anj/agent-skills-manager/internal/vendorlist"
)

// version is the CLI version (override with -ldflags "-X main.version=...").
var version = "0.3.0"

func main() {
	cfg := config.Default()
	// migrate legacy state early (best-effort)
	_ = config.MigrateLegacyState(cfg)

	// global flags may appear before or after the command
	rest := parseGlobal(cfg, os.Args[1:])
	if len(rest) == 0 {
		usage(cfg)
		os.Exit(2)
	}
	cmd := rest[0]
	args := rest[1:]

	vl, err := vendorlist.Load(embedded.VendorSkillsYAML, cfg)
	if err != nil {
		fail(cfg, err)
	}
	// Grok: prefer live `grok inspect` over static name denylist (can take a few seconds)
	prog := progress.New(cfg.JSON)
	if !cfg.NoProbe && cfg.ToolEnabled("grok") && needsGrokProbe(cmd) {
		prog.Statusf("%s", cfg.T().S("progress.probe_grok"))
	}
	if needsGrokProbe(cmd) {
		vl.AttachGrokProbe(cfg)
	}
	prog.Done()

	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage(cfg)
	case "scan":
		runScan(cfg, vl)
	case "plan":
		runPlan(cfg, vl, args)
	case "apply":
		runApply(cfg, vl, args)
	case "sync":
		// W2: rescan → plan → apply (same conflict rules as apply)
		runSync(cfg, vl, args)
	case "undo":
		id := "last"
		if len(args) > 0 {
			id = args[0]
		}
		runUndo(cfg, id)
	case "doctor":
		runDoctor(cfg, vl)
	case "outdated", "check-update", "check-updates":
		runOutdated(cfg, vl, args)
	case "remove", "rm", "uninstall":
		runRemove(cfg, vl, args)
	case "disable", "off":
		runSkillGate(cfg, false, args)
	case "enable", "on":
		runSkillGate(cfg, true, args)
	case "gates", "gate":
		runGates(cfg, args)
	case "upgrade", "self-update", "selfupdate":
		runUpgrade(cfg, args)
	default:
		fmt.Fprintf(os.Stderr, cfg.T().S("main.unknown_cmd"), cmd)
		usage(cfg)
		os.Exit(2)
	}
}

func parseGlobal(cfg *config.Config, args []string) []string {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--yes" || a == "-y":
			cfg.Yes = true
		case a == "--json":
			cfg.JSON = true
		case a == "--all":
			cfg.ShowAll = true
		case a == "--no-project":
			cfg.NoProject = true
		case a == "--no-probe":
			cfg.NoProbe = true
		case a == "--en":
			cfg.Lang = i18n.EN
		case a == "--zh":
			cfg.Lang = i18n.ZH
		case a == "--lang" && i+1 < len(args):
			i++
			cfg.Lang = i18n.Parse(args[i])
		case strings.HasPrefix(a, "--lang="):
			cfg.Lang = i18n.Parse(strings.TrimPrefix(a, "--lang="))
		case a == "--strategy" && i+1 < len(args):
			i++
			cfg.Strategy = args[i]
		case len(a) > 11 && a[:11] == "--strategy=":
			cfg.Strategy = a[11:]
		case a == "--only" && i+1 < len(args):
			i++
			cfg.OnlyTools = config.ParseCSV(args[i])
		case a == "--exclude" && i+1 < len(args):
			i++
			cfg.ExcludeTools = config.ParseCSV(args[i])
		case a == "--extra-root" && i+1 < len(args):
			i++
			cfg.ExtraRoots = append(cfg.ExtraRoots, args[i])
		case a == "--project-root" && i+1 < len(args):
			i++
			cfg.ProjectRoots = append(cfg.ProjectRoots, args[i])
		default:
			rest = append(rest, a)
		}
	}
	return rest
}

func runScan(cfg *config.Config, vl *vendorlist.List) {
	res, err := scan.Run(cfg, vl)
	if err != nil {
		fail(cfg, err)
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}
	printScanHuman(cfg, vl, res)
}

func printScanHuman(cfg *config.Config, vl *vendorlist.List, res *scan.Result) {
	tr := cfg.T()
	byClass := map[scan.Class][]scan.Entry{}
	for _, e := range res.Entries {
		byClass[e.Class] = append(byClass[e.Class], e)
	}
	for c := range byClass {
		sort.Slice(byClass[c], func(i, j int) bool {
			if byClass[c][i].Name != byClass[c][j].Name {
				return byClass[c][i].Name < byClass[c][j].Name
			}
			return byClass[c][i].Path < byClass[c][j].Path
		})
	}

	cC := len(byClass[scan.ClassCanonical])
	cU := len(byClass[scan.ClassUser])
	cV := len(byClass[scan.ClassVendor])
	cP := len(byClass[scan.ClassProject])
	cW := len(res.WireGaps)

	fmt.Println(tr.S("scan.summary_title"))
	fmt.Println(tr.S("scan.sum_canonical", cC))
	fmt.Println(tr.S("scan.sum_user", cU))
	fmt.Println(tr.S("scan.sum_unwired", cW))
	fmt.Println(tr.S("scan.sum_project", cP))
	fmt.Println(tr.S("scan.sum_vendor", cV))
	fmt.Println(tr.S("scan.sum_total", len(res.Entries)))
	fmt.Println()

	type section struct {
		class scan.Class
		title string
		hint  string
	}
	sections := []section{
		{scan.ClassCanonical, tr.S("scan.sec_canonical"), tr.S("scan.sec_canonical_h")},
		{scan.ClassUser, tr.S("scan.sec_user"), tr.S("scan.sec_user_h")},
		{scan.ClassProject, tr.S("scan.sec_project"), tr.S("scan.sec_project_h")},
		{scan.ClassVendor, tr.S("scan.sec_vendor"), tr.S("scan.sec_vendor_h")},
	}

	for _, sec := range sections {
		entries := byClass[sec.class]
		if len(entries) == 0 {
			continue
		}
		fmt.Println(tr.S("scan.sec_header", sec.title, len(entries), sec.hint))

		if sec.class == scan.ClassVendor && !cfg.ShowAll {
			byTool := map[string]int{}
			for _, e := range entries {
				byTool[e.Tool]++
			}
			tools := make([]string, 0, len(byTool))
			for t := range byTool {
				tools = append(tools, t)
			}
			sort.Strings(tools)
			parts := make([]string, 0, len(tools))
			for _, t := range tools {
				parts = append(parts, fmt.Sprintf("%s=%d", t, byTool[t]))
			}
			fmt.Println(tr.S("scan.vendor_collapsed", strings.Join(parts, "  ")))
			fmt.Println(tr.S("scan.vendor_tip"))
			fmt.Println()
			continue
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintf(w, "  %s\t%s\t%s\n", tr.S("scan.col_name"), tr.S("scan.col_tool"), tr.S("scan.col_path"))
		for _, e := range entries {
			pathCol := shortPath(e.Path)
			if e.Symlink && e.Target != "" {
				pathCol = pathCol + "  → " + shortPath(e.Target)
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\n", e.Name, tr.ToolLabel(e.Tool), pathCol)
		}
		_ = w.Flush()
		fmt.Println()
	}

	// Skills already in agents but missing/wrong tool symlinks (manual agents drops).
	if cW > 0 {
		fmt.Println(tr.S("scan.sec_header", tr.S("scan.sec_unwired"), cW, tr.S("scan.sec_unwired_h")))
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			tr.S("scan.col_name"), tr.S("scan.col_missing"), tr.S("scan.col_wrong"), tr.S("scan.col_path"))
		for _, g := range res.WireGaps {
			miss := strings.Join(g.Missing, ",")
			if miss == "" {
				miss = "-"
			}
			wrong := strings.Join(g.Wrong, ",")
			if wrong == "" {
				wrong = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", g.Name, miss, wrong, shortPath(g.AgentsPath))
		}
		_ = w.Flush()
		fmt.Println()
	}

	if cU > 0 {
		fmt.Println(tr.S("scan.next_plan"))
	} else if cW > 0 {
		fmt.Println(tr.S("scan.next_wire"))
	}
	if cP == 0 && !cfg.NoProject {
		fmt.Println(tr.S("scan.project_note1"))
		fmt.Println(tr.S("scan.project_note2"))
	}
	printProbeNote(cfg, vl)
}

func printProbeNote(cfg *config.Config, vl *vendorlist.List) {
	if vl == nil {
		return
	}
	tr := cfg.T()
	if cfg.NoProbe {
		fmt.Println(tr.S("probe.disabled"))
		return
	}
	if !cfg.ToolEnabled("grok") {
		return
	}
	if vl.GrokProbeOK {
		fmt.Println(tr.S("probe.grok_ok", len(vl.GrokBundledName)))
	} else if vl.GrokProbeErr != "" {
		fmt.Println(tr.S("probe.grok_fail", vl.GrokProbeErr))
	}
}

func shortPath(p string) string {
	return fsutil.RelHome(p)
}

func runPlan(cfg *config.Config, vl *vendorlist.List, args []string) {
	tr := cfg.T()
	outPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--out" && i+1 < len(args) {
			outPath = args[i+1]
			i++
		}
	}
	res, err := scan.Run(cfg, vl)
	if err != nil {
		fail(cfg, err)
	}
	p := plan.Build(cfg, res)
	if outPath != "" {
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			fail(cfg, err)
		}
		if err := os.WriteFile(outPath, b, 0o644); err != nil {
			fail(cfg, err)
		}
		fmt.Println(tr.S("plan.wrote", outPath))
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(p)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		tr.S("plan.col_type"), tr.S("plan.col_name"), tr.S("plan.col_tool"),
		tr.S("plan.col_note"), tr.S("plan.col_source"))
	for _, a := range p.Actions {
		src := shortPath(a.SourcePath)
		if a.Type == plan.ActionConflict {
			src = tr.S("plan.candidates", len(a.Candidates))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tr.ActionLabel(string(a.Type)), a.Name, tr.ToolLabel(a.Tool), tr.NoteLabel(a.Note), src)
	}
	_ = w.Flush()
	fmt.Println(tr.S("plan.summary", p.Summary.Migrate, p.Summary.Link, p.Summary.Skip, p.Summary.Conflict))
	fmt.Println(tr.S("plan.no_change"))
}

func runApply(cfg *config.Config, vl *vendorlist.List, args []string) {
	planPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--plan" && i+1 < len(args) {
			planPath = args[i+1]
			i++
		}
	}
	var p *plan.Plan
	if planPath != "" {
		b, err := os.ReadFile(planPath)
		if err != nil {
			fail(cfg, err)
		}
		p = &plan.Plan{}
		if err := json.Unmarshal(b, p); err != nil {
			fail(cfg, err)
		}
	} else {
		res, err := scan.Run(cfg, vl)
		if err != nil {
			fail(cfg, err)
		}
		p = plan.Build(cfg, res)
	}
	if _, err := apply.Run(cfg, p, os.Stdin); err != nil {
		fail(cfg, err)
	}
}

// runSync is the personal happy path: rescan → show plan → apply.
// Conflict resolution is identical to apply (C1). Use -y for non-interactive
// (still requires --strategy when conflicts exist and not a TTY).
func runSync(cfg *config.Config, vl *vendorlist.List, args []string) {
	tr := cfg.T()
	// reject leftover flags we don't support on sync
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Print(tr.S("sync.help"))
			return
		}
		if strings.HasPrefix(a, "-") {
			fail(cfg, fmt.Errorf("%s", tr.S("sync.bad_flag", a)))
		}
	}

	res, err := scan.Run(cfg, vl)
	if err != nil {
		fail(cfg, err)
	}
	p := plan.Build(cfg, res)

	if cfg.JSON {
		// machine mode: just apply (plan embedded in apply flow via result)
		if _, err := apply.Run(cfg, p, os.Stdin); err != nil {
			fail(cfg, err)
		}
		return
	}

	fmt.Println(tr.S("sync.banner"))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		tr.S("plan.col_type"), tr.S("plan.col_name"), tr.S("plan.col_tool"),
		tr.S("plan.col_note"), tr.S("plan.col_source"))
	for _, a := range p.Actions {
		src := shortPath(a.SourcePath)
		if a.Type == plan.ActionConflict {
			src = tr.S("plan.candidates", len(a.Candidates))
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			tr.ActionLabel(string(a.Type)), a.Name, tr.ToolLabel(a.Tool), tr.NoteLabel(a.Note), src)
	}
	_ = w.Flush()
	fmt.Println(tr.S("plan.summary", p.Summary.Migrate, p.Summary.Link, p.Summary.Skip, p.Summary.Conflict))
	fmt.Println(tr.S("sync.plan_done"))

	if p.Summary.Migrate == 0 && p.Summary.Link == 0 && p.Summary.Conflict == 0 {
		fmt.Println(tr.S("sync.nothing"))
		return
	}

	if _, err := apply.Run(cfg, p, os.Stdin); err != nil {
		fail(cfg, err)
	}
}

func runUndo(cfg *config.Config, id string) {
	tx, err := apply.LoadTx(cfg, id)
	if err != nil {
		fail(cfg, err)
	}
	if err := apply.Undo(cfg, tx); err != nil {
		fail(cfg, err)
	}
}

func runDoctor(cfg *config.Config, vl *vendorlist.List) {
	tr := cfg.T()
	prog := progress.New(cfg.JSON)
	prog.Statusf("%s", tr.S("progress.doctor"))
	issues, err := doctor.Run(cfg, vl)
	prog.Done()
	if err != nil {
		fail(cfg, err)
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(issues)
		return
	}
	if len(issues) == 0 {
		fmt.Println(tr.S("doctor.ok"))
		return
	}
	for _, is := range issues {
		fmt.Printf("%-6s  %s\n        %s\n", tr.SeverityLabel(is.Severity), shortPath(is.Path), is.Message)
	}
	fmt.Print(tr.S("doctor.count", len(issues)))
	fmt.Println()
}

func runOutdated(cfg *config.Config, vl *vendorlist.List, args []string) {
	tr := cfg.T()
	doUpdate := false
	var names []string
	for _, a := range args {
		switch a {
		case "--update", "--fix", "-u":
			doUpdate = true
		case "--yes", "-y":
			cfg.Yes = true
		case "--all":
			cfg.ShowAll = true
		case "--help", "-h":
			fmt.Print(tr.S("outdated.help"))
			return
		default:
			if strings.HasPrefix(a, "-") {
				fail(cfg, fmt.Errorf("unknown outdated flag: %s (use --update / -u / -y / --all)", a))
			}
			names = append(names, a)
		}
	}

	prog := progress.New(cfg.JSON)
	res, err := outdated.RunOpts(cfg, vl, outdated.RunOptions{Progress: prog})
	prog.Done()
	if err != nil {
		fail(cfg, err)
	}

	if cfg.JSON && !doUpdate {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}

	fmt.Println(tr.S("outdated.title"))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		tr.S("outdated.col_status"), tr.S("outdated.col_name"), tr.S("outdated.col_prov"),
		tr.S("outdated.col_local"), tr.S("outdated.col_remote"), tr.S("outdated.col_msg"))
	for _, e := range res.Entries {
		if e.Status == outdated.StatusUnknown && !cfg.ShowAll {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			statusLabel(tr, e.Status), e.Name, string(e.Provenance),
			e.LocalRef, e.RemoteRef, e.Message)
	}
	_ = w.Flush()
	fmt.Println(tr.S("outdated.summary",
		res.Summary.Outdated, res.Summary.Current, res.Summary.Unknown, res.Summary.Error, res.Summary.Total))
	if !cfg.ShowAll && res.Summary.Unknown > 0 {
		fmt.Printf("  (%d unknown hidden; pass --all to list)\n", res.Summary.Unknown)
	}

	if !doUpdate {
		fmt.Println(tr.S("outdated.note"))
		return
	}

	fmt.Println(tr.S("outdated.update_begin"))
	ur := outdated.ApplyUpdates(res, outdated.UpdateOptions{
		Yes:   cfg.Yes,
		Names: names,
	})
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"check": res, "update": ur})
		return
	}
	for _, it := range ur.Items {
		mark := "ok"
		if it.Skipped {
			mark = "skip"
		} else if !it.OK {
			mark = "fail"
		}
		fmt.Printf("  [%s] %s (%s): %s\n", mark, it.Name, it.Action, it.Message)
	}
	fmt.Println(tr.S("outdated.update_summary", ur.Updated, ur.Skipped, ur.Failed))
}

func statusLabel(tr *i18n.T, s outdated.Status) string {
	switch s {
	case outdated.StatusOutdated:
		return tr.S("outdated.status_outdated")
	case outdated.StatusCurrent:
		return tr.S("outdated.status_current")
	case outdated.StatusUnknown:
		return tr.S("outdated.status_unknown")
	case outdated.StatusError:
		return tr.S("outdated.status_error")
	case outdated.StatusSkipped:
		return tr.S("outdated.status_skipped")
	default:
		return string(s)
	}
}

func runRemove(cfg *config.Config, vl *vendorlist.List, args []string) {
	tr := cfg.T()
	opt := remove.Options{Yes: cfg.Yes, Stdin: os.Stdin, Stdout: os.Stdout}
	var names []string
	for _, a := range args {
		switch a {
		case "--yes", "-y":
			opt.Yes = true
		case "--keep-lock":
			opt.KeepLock = true
		case "--project":
			opt.IncludeProject = true
		case "--help", "-h":
			fmt.Print(tr.S("remove.help"))
			return
		default:
			if strings.HasPrefix(a, "-") {
				fail(cfg, fmt.Errorf("unknown remove flag: %s", a))
			}
			names = append(names, a)
		}
	}
	if len(names) == 0 {
		fail(cfg, fmt.Errorf("%s", tr.S("remove.need_name")))
	}
	opt.Names = names

	res, err := remove.Run(cfg, vl, opt)
	if err != nil {
		fail(cfg, err)
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}
	if res.Aborted {
		return
	}
	fmt.Println(tr.S("remove.done", len(res.Removed), len(res.Failed), len(res.LockModified)))
	for _, p := range res.Removed {
		fmt.Printf("  - %s\n", fsutil.RelHome(p))
	}
	for _, f := range res.Failed {
		fmt.Printf("  ! %s\n", f)
	}
	for _, p := range res.LockModified {
		fmt.Printf("  lock: %s\n", fsutil.RelHome(p))
	}
}

// runSkillGate handles enable/disable (global or --tool).
func runSkillGate(cfg *config.Config, enable bool, args []string) {
	tr := cfg.T()
	var tools []string
	var names []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			if enable {
				fmt.Print(tr.S("gate.enable_help"))
			} else {
				fmt.Print(tr.S("gate.disable_help"))
			}
			return
		case a == "--tool" || a == "-t":
			if i+1 >= len(args) {
				fail(cfg, fmt.Errorf("%s", tr.S("gate.need_tool")))
			}
			i++
			tools = append(tools, args[i])
		case strings.HasPrefix(a, "--tool="):
			tools = append(tools, strings.TrimPrefix(a, "--tool="))
		case strings.HasPrefix(a, "-"):
			fail(cfg, fmt.Errorf("%s", tr.S("gate.bad_flag", a)))
		default:
			names = append(names, a)
		}
	}
	if len(names) == 0 {
		fail(cfg, fmt.Errorf("%s", tr.S("gate.need_name")))
	}

	store, err := skillgate.Open(cfg)
	if err != nil {
		fail(cfg, err)
	}

	type row struct {
		Name string             `json:"name"`
		Res  *skillgate.Result  `json:"result"`
		Err  string             `json:"error,omitempty"`
	}
	var rows []row
	for _, name := range names {
		var (
			r   *skillgate.Result
			err error
		)
		if enable {
			r, err = store.Enable(name, tools)
		} else {
			r, err = store.Disable(name, tools)
		}
		if err != nil {
			rows = append(rows, row{Name: name, Err: err.Error()})
			continue
		}
		rows = append(rows, row{Name: name, Res: r})
	}

	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
		// non-zero exit if any error
		for _, r := range rows {
			if r.Err != "" {
				os.Exit(1)
			}
		}
		return
	}

	failed := 0
	for _, r := range rows {
		if r.Err != "" {
			failed++
			fmt.Printf("❌ %s: %s\n", r.Name, r.Err)
			continue
		}
		res := r.Res
		mark := "✓"
		if res.Already {
			mark = "·"
		}
		fmt.Printf("%s %s — %s\n", mark, res.Name, res.Message)
		for _, p := range res.Removed {
			fmt.Printf("    − %s\n", shortPath(p))
		}
		for _, p := range res.Linked {
			fmt.Printf("    + %s\n", shortPath(p))
		}
		if res.Parked != "" {
			fmt.Printf("    parked → %s\n", shortPath(res.Parked))
		}
		if res.Restored != "" {
			fmt.Printf("    restored → %s\n", shortPath(res.Restored))
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func runGates(cfg *config.Config, args []string) {
	tr := cfg.T()
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Print(tr.S("gate.list_help"))
			return
		}
	}
	store, err := skillgate.Open(cfg)
	if err != nil {
		fail(cfg, err)
	}
	names := store.List()
	type item struct {
		Name      string   `json:"name"`
		GlobalOff bool     `json:"global_off"`
		ToolsOff  []string `json:"tools_off,omitempty"`
	}
	var items []item
	for _, n := range names {
		e := store.Get(n)
		items = append(items, item{Name: n, GlobalOff: e.GlobalOff, ToolsOff: e.ToolsOff})
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
		return
	}
	if len(items) == 0 {
		fmt.Println(tr.S("gate.list_empty"))
		return
	}
	fmt.Println(tr.S("gate.list_title"))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\n", tr.S("gate.col_name"), tr.S("gate.col_global"), tr.S("gate.col_tools"))
	for _, it := range items {
		g := tr.S("gate.no")
		if it.GlobalOff {
			g = tr.S("gate.yes")
		}
		tools := "-"
		if len(it.ToolsOff) > 0 {
			tools = strings.Join(it.ToolsOff, ",")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", it.Name, g, tools)
	}
	_ = w.Flush()
	fmt.Println(tr.S("gate.list_hint"))
}

func runUpgrade(cfg *config.Config, args []string) {
	tr := cfg.T()
	checkOnly := false
	for _, a := range args {
		switch a {
		case "--check", "-c":
			checkOnly = true
		case "--yes", "-y":
			cfg.Yes = true
		}
	}
	opt := selfupdate.Options{
		Current:   version,
		CheckOnly: checkOnly,
		Yes:       cfg.Yes,
	}
	prog := progress.New(cfg.JSON)
	// Preview first when not --check and not --yes on TTY
	if !checkOnly && !cfg.Yes {
		prog.Statusf("%s", tr.S("progress.upgrade_check"))
		preview, err := selfupdate.Run(context.Background(), selfupdate.Options{
			Current:   version,
			CheckOnly: true,
		})
		prog.Done()
		if err != nil {
			fail(cfg, err)
		}
		if preview.UpToDate {
			if cfg.JSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(preview)
				return
			}
			fmt.Println(tr.S("upgrade.current", preview.Latest))
			return
		}
		fmt.Println(tr.S("upgrade.available", preview.Current, preview.Latest, preview.Asset, preview.Repo))
		fmt.Print(tr.S("upgrade.confirm"))
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			fmt.Println(tr.S("upgrade.aborted"))
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if checkOnly {
		prog.Statusf("%s", tr.S("progress.upgrade_check"))
	} else {
		prog.Statusf("%s", tr.S("progress.upgrade_download"))
	}
	res, err := selfupdate.Run(ctx, opt)
	prog.Done()
	if err != nil {
		fail(cfg, err)
	}
	if cfg.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}
	if res.Updated {
		fmt.Println(tr.S("upgrade.done", res.Current, res.Latest))
		return
	}
	if res.UpToDate {
		fmt.Println(tr.S("upgrade.current", res.Latest))
		return
	}
	// check-only with update available
	fmt.Println(tr.S("upgrade.available", res.Current, res.Latest, res.Asset, res.Repo))
	if checkOnly {
		fmt.Println(tr.S("upgrade.check_hint"))
	}
}

func usage(cfg *config.Config) {
	fmt.Fprintf(os.Stderr, cfg.T().S("main.usage"), version)
}

func fail(cfg *config.Config, err error) {
	fmt.Fprintln(os.Stderr, cfg.T().S("main.error_prefix"), err)
	os.Exit(1)
}

// needsGrokProbe: skip live grok inspect on meta commands that never classify skills.
func needsGrokProbe(cmd string) bool {
	switch cmd {
	case "version", "-v", "--version", "help", "-h", "--help",
		"upgrade", "self-update", "selfupdate":
		return false
	default:
		return true
	}
}
