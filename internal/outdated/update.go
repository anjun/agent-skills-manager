package outdated

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anj/agent-skills-manager/internal/knownsource"
)

// UpdateOptions controls ApplyUpdates.
type UpdateOptions struct {
	// Yes skips confirmation prompts.
	Yes bool
	// Names limits updates to these skill names (empty = all outdated).
	Names []string
	// Stdin for confirmations (default os.Stdin).
	Stdin io.Reader
	// Stdout for progress (default os.Stdout).
	Stdout io.Writer
	// RunCmd runs external commands; nil uses a real exec with timeout.
	// dir is working directory ("" = process cwd).
	RunCmd func(dir, name string, args ...string) (combinedOut string, err error)
	// LookPath finds a binary on PATH; nil uses exec.LookPath (injectable for tests).
	LookPath func(file string) (string, error)
}

// UpdateItem is one attempted update.
type UpdateItem struct {
	Name       string     `json:"name"`
	Provenance Provenance `json:"provenance"`
	Action     string     `json:"action"` // npx-skills | known/cli id | git-pull | skip
	OK         bool       `json:"ok"`
	Skipped    bool       `json:"skipped,omitempty"`
	Message    string     `json:"message,omitempty"`
}

// UpdateResult is the outcome of ApplyUpdates.
type UpdateResult struct {
	Items   []UpdateItem `json:"items"`
	Updated int          `json:"updated"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
}

// ApplyUpdates tries to refresh skills that are StatusOutdated.
//
//	skill-lock → npx skills update <names...> -g -y
//	known      → knownsource.Catalog[name].Update steps (e.g. dws CLI)
//	git        → git -C <path> pull --ff-only
//	deleted upstream / unknown → skip with message
func ApplyUpdates(report *Result, opt UpdateOptions) *UpdateResult {
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Stdin == nil {
		opt.Stdin = os.Stdin
	}
	if opt.RunCmd == nil {
		opt.RunCmd = defaultRunCmd
	}
	if opt.LookPath == nil {
		opt.LookPath = exec.LookPath
	}

	out := &UpdateResult{}
	want := map[string]bool{}
	for _, n := range opt.Names {
		n = strings.TrimSpace(strings.ToLower(n))
		if n != "" {
			want[n] = true
		}
	}

	var targets []Entry
	for _, e := range report.Entries {
		if e.Status != StatusOutdated {
			continue
		}
		if len(want) > 0 && !want[strings.ToLower(e.Name)] {
			continue
		}
		targets = append(targets, e)
	}
	if len(targets) == 0 {
		fmt.Fprintln(opt.Stdout, "没有可更新的 skill（Status=outdated）。")
		return out
	}

	fmt.Fprintf(opt.Stdout, "将尝试更新 %d 个 skill：\n", len(targets))
	for _, e := range targets {
		fmt.Fprintf(opt.Stdout, "  • %s  (%s)\n", e.Name, e.Provenance)
	}
	if !opt.Yes {
		fmt.Fprint(opt.Stdout, "继续？[y/N] ")
		line, _ := bufio.NewReader(opt.Stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			fmt.Fprintln(opt.Stdout, "已取消。")
			return out
		}
	}

	var (
		lockNames []string
		lockSeen  = map[string]bool{}
		gitOnes   []Entry
		gitSeen   = map[string]bool{}
		// known updates keyed by catalog name; value is first entry for reporting
		knownOnes []knownUpdateJob
		knownSeen = map[string]bool{}
		skips     []Entry
	)
	for _, e := range targets {
		key := strings.ToLower(e.Name)
		switch {
		case isDeletedUpstream(e):
			skips = append(skips, e)
		case e.Provenance == ProvSkillLock:
			if lockSeen[key] {
				continue
			}
			lockSeen[key] = true
			lockNames = append(lockNames, e.Name)
		case e.Provenance == ProvGit:
			if gitSeen[key] {
				continue
			}
			gitSeen[key] = true
			gitOnes = append(gitOnes, e)
		case e.Provenance == ProvKnown:
			src, ok := knownsource.Lookup(e.Name)
			if !ok || src.Update == nil || len(src.Update.Steps) == 0 {
				skips = append(skips, e)
				continue
			}
			// OncePerMachine (default for multi-home installers): one run per catalog name
			once := src.Update.OncePerMachine
			if once && knownSeen[key] {
				continue
			}
			if !knownSeen[key] {
				knownSeen[key] = true
				knownOnes = append(knownOnes, knownUpdateJob{Entry: e, Source: src})
			} else if !once {
				knownOnes = append(knownOnes, knownUpdateJob{Entry: e, Source: src})
			}
		default:
			skips = append(skips, e)
		}
	}

	if len(lockNames) > 0 {
		out.Items = append(out.Items, updateSkillLockBatch(lockNames, opt)...)
	}
	for _, e := range gitOnes {
		out.Items = append(out.Items, updateGit(e, opt))
	}
	for _, job := range knownOnes {
		out.Items = append(out.Items, updateKnown(job.Entry, job.Source, opt))
	}
	for _, e := range skips {
		msg := e.Message
		if isDeletedUpstream(e) {
			msg = "上游已删除或改名，跳过自动更新（可手动删除本地 skill 或从 skill-lock 移除）"
		} else if strings.TrimSpace(msg) == "" {
			msg = "无自动更新路径"
		}
		out.Items = append(out.Items, UpdateItem{
			Name: e.Name, Provenance: e.Provenance, Action: "skip",
			Skipped: true, Message: msg,
		})
	}

	for _, it := range out.Items {
		switch {
		case it.Skipped:
			out.Skipped++
		case it.OK:
			out.Updated++
		default:
			out.Failed++
		}
	}
	return out
}

type knownUpdateJob struct {
	Entry  Entry
	Source knownsource.Source
}

func isDeletedUpstream(e Entry) bool {
	m := strings.ToLower(e.Message)
	return strings.Contains(m, "upstream path missing") ||
		strings.Contains(m, "deleted or renamed") ||
		strings.Contains(m, "appear to have been deleted")
}

func updateSkillLockBatch(names []string, opt UpdateOptions) []UpdateItem {
	fmt.Fprintf(opt.Stdout, "→ npx skills update %s -g -y\n", strings.Join(names, " "))
	args := append([]string{"--yes", "skills", "update"}, names...)
	args = append(args, "-g", "-y")
	_, err := opt.RunCmd("", "npx", args...)
	msg := "npx skills update " + strings.Join(names, " ") + " -g -y"
	if err != nil {
		fmt.Fprintln(opt.Stdout, "  具名更新失败，尝试全量: npx skills update -g -y")
		_, err2 := opt.RunCmd("", "npx", "--yes", "skills", "update", "-g", "-y")
		if err2 != nil {
			items := make([]UpdateItem, 0, len(names))
			for _, n := range names {
				items = append(items, UpdateItem{
					Name: n, Provenance: ProvSkillLock, Action: "npx-skills",
					OK: false, Message: err2.Error(),
				})
			}
			return items
		}
		msg = "npx skills update -g -y (full)"
	}
	items := make([]UpdateItem, 0, len(names))
	for _, n := range names {
		items = append(items, UpdateItem{
			Name: n, Provenance: ProvSkillLock, Action: "npx-skills",
			OK: true, Message: msg,
		})
	}
	return items
}

func updateGit(e Entry, opt UpdateOptions) UpdateItem {
	fmt.Fprintf(opt.Stdout, "→ git -C %s pull --ff-only\n", e.Path)
	out, err := opt.RunCmd(e.Path, "git", "pull", "--ff-only")
	it := UpdateItem{Name: e.Name, Provenance: ProvGit, Action: "git-pull"}
	if err != nil {
		it.OK = false
		it.Message = err.Error()
		if strings.TrimSpace(out) != "" {
			it.Message = strings.TrimSpace(out)
		}
		return it
	}
	it.OK = true
	it.Message = strings.TrimSpace(out)
	if it.Message == "" {
		it.Message = "git pull --ff-only ok"
	}
	return it
}

// updateKnown runs Catalog UpdateSpec steps (generic; dws is just one catalog entry).
func updateKnown(e Entry, src knownsource.Source, opt UpdateOptions) UpdateItem {
	spec := src.Update
	it := UpdateItem{
		Name:       e.Name,
		Provenance: ProvKnown,
		Action:     src.ActionID(),
	}
	if spec == nil || len(spec.Steps) == 0 {
		it.Skipped = true
		it.Message = "known source has no UpdateSpec"
		return it
	}

	if bin := strings.TrimSpace(spec.RequiresBin); bin != "" {
		if _, err := opt.LookPath(bin); err != nil {
			it.OK = false
			it.Skipped = true
			msg := "未找到 " + bin + " 命令"
			if h := strings.TrimSpace(spec.InstallHint); h != "" {
				msg += "。可: " + h
			}
			it.Message = msg
			return it
		}
	}

	for _, step := range spec.Steps {
		if len(step.Argv) == 0 {
			continue
		}
		label := step.Label
		if label == "" {
			label = strings.Join(step.Argv, " ")
		}
		fmt.Fprintln(opt.Stdout, "→", label)
		_, err := opt.RunCmd("", step.Argv[0], step.Argv[1:]...)
		if err != nil {
			if step.ContinueOnError {
				fmt.Fprintln(opt.Stdout, "  步骤失败，继续:", err)
				continue
			}
			it.OK = false
			it.Message = err.Error()
			return it
		}
	}

	it.OK = true
	it.Message = spec.SuccessMessage
	if it.Message == "" {
		it.Message = "known source update ok (" + src.OwnerRepo + ")"
	}
	return it
}

func defaultRunCmd(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var buf bytes.Buffer
	// stream to user + capture
	cmd.Stdout = io.MultiWriter(&buf, os.Stdout)
	cmd.Stderr = io.MultiWriter(&buf, os.Stderr)
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if out != "" {
			return out, fmt.Errorf("%s: %w", truncateMsg(out, 240), err)
		}
		return "", err
	}
	return out, nil
}

func truncateMsg(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
