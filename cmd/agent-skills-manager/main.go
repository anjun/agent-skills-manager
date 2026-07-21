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
	"github.com/anj/agent-skills-manager/internal/scan"
	"github.com/anj/agent-skills-manager/internal/selfupdate"
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
	// Grok: prefer live `grok inspect` over static name denylist
	vl.AttachGrokProbe(cfg)

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
	case "undo":
		id := "last"
		if len(args) > 0 {
			id = args[0]
		}
		runUndo(cfg, id)
	case "doctor":
		runDoctor(cfg, vl)
	case "outdated", "check-update", "check-updates":
		runOutdated(cfg, vl)
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

	fmt.Println(tr.S("scan.summary_title"))
	fmt.Println(tr.S("scan.sum_canonical", cC))
	fmt.Println(tr.S("scan.sum_user", cU))
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

	if cU > 0 {
		fmt.Println(tr.S("scan.next_plan"))
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
	issues, err := doctor.Run(cfg, vl)
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

func runOutdated(cfg *config.Config, vl *vendorlist.List) {
	tr := cfg.T()
	res, err := outdated.Run(cfg, vl)
	if err != nil {
		fail(cfg, err)
	}
	if cfg.JSON {
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
		// hide pure unknown noise unless --all
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
	fmt.Println(tr.S("outdated.note"))
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
	// Preview first when not --check and not --yes on TTY
	if !checkOnly && !cfg.Yes {
		preview, err := selfupdate.Run(context.Background(), selfupdate.Options{
			Current:   version,
			CheckOnly: true,
		})
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
	res, err := selfupdate.Run(ctx, opt)
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
