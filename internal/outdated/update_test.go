package outdated

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestApplyUpdatesSkillLockAndSkipDeleted(t *testing.T) {
	var ran []string
	report := &Result{
		Entries: []Entry{
			{Name: "code-review", Status: StatusOutdated, Provenance: ProvSkillLock},
			{Name: "write-a-skill", Status: StatusOutdated, Provenance: ProvSkillLock,
				Message: "upstream path missing on mattpocock/skills (deleted or renamed)"},
			{Name: "tdd", Status: StatusCurrent, Provenance: ProvSkillLock},
		},
	}
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Stdout: &bytes.Buffer{},
		Stdin:  strings.NewReader(""),
		RunCmd: func(dir, name string, args ...string) (string, error) {
			ran = append(ran, name+" "+strings.Join(args, " "))
			return "ok", nil
		},
	})
	if ur.Updated != 1 {
		t.Fatalf("updated=%d items=%+v", ur.Updated, ur.Items)
	}
	if ur.Skipped != 1 {
		t.Fatalf("skipped=%d", ur.Skipped)
	}
	if len(ran) != 1 || !strings.Contains(ran[0], "skills update") {
		t.Fatalf("ran=%v", ran)
	}
	// deleted should not be in npx names
	if strings.Contains(ran[0], "write-a-skill") {
		t.Fatalf("should not update deleted skill: %s", ran[0])
	}
}

func TestApplyUpdatesKnownOnce(t *testing.T) {
	// dws is catalog-driven; three copies must produce one UpdateSpec run.
	report := &Result{
		Entries: []Entry{
			{Name: "dws", Status: StatusOutdated, Provenance: ProvKnown, Path: "/a/dws",
				Remote: "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli"},
			{Name: "dws", Status: StatusOutdated, Provenance: ProvKnown, Path: "/b/dws",
				Remote: "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli"},
			{Name: "dws", Status: StatusOutdated, Provenance: ProvKnown, Path: "/c/dws",
				Remote: "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli"},
		},
	}
	var nSetup, nUpgrade int
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Stdout: &bytes.Buffer{},
		LookPath: func(file string) (string, error) {
			if file == "dws" {
				return "/usr/local/bin/dws", nil
			}
			return "", os.ErrNotExist
		},
		RunCmd: func(dir, name string, args ...string) (string, error) {
			line := name + " " + strings.Join(args, " ")
			if strings.Contains(line, "skill setup") {
				nSetup++
			}
			if strings.Contains(line, "upgrade") {
				nUpgrade++
			}
			return "ok", nil
		},
	})
	if len(ur.Items) != 1 {
		t.Fatalf("want 1 known update item, got %+v", ur.Items)
	}
	if ur.Items[0].Skipped || !ur.Items[0].OK {
		t.Fatalf("item=%+v", ur.Items[0])
	}
	if ur.Items[0].Action != "dws" {
		t.Fatalf("action=%s", ur.Items[0].Action)
	}
	if nSetup != 1 || nUpgrade != 1 {
		t.Fatalf("setup=%d upgrade=%d (want 1 each)", nSetup, nUpgrade)
	}
}

func TestApplyUpdatesKnownMissingBin(t *testing.T) {
	report := &Result{
		Entries: []Entry{
			{Name: "dws", Status: StatusOutdated, Provenance: ProvKnown},
		},
	}
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Stdout: &bytes.Buffer{},
		LookPath: func(file string) (string, error) {
			return "", os.ErrNotExist
		},
		RunCmd: func(dir, name string, args ...string) (string, error) {
			t.Fatal("should not run when bin missing")
			return "", nil
		},
	})
	if len(ur.Items) != 1 || !ur.Items[0].Skipped {
		t.Fatalf("%+v", ur.Items)
	}
	if !strings.Contains(ur.Items[0].Message, "dws") {
		t.Fatalf("msg=%s", ur.Items[0].Message)
	}
}

func TestApplyUpdatesGit(t *testing.T) {
	report := &Result{
		Entries: []Entry{
			{Name: "myskill", Path: "/tmp/foo", Status: StatusOutdated, Provenance: ProvGit},
		},
	}
	var gotDir string
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Stdout: &bytes.Buffer{},
		RunCmd: func(dir, name string, args ...string) (string, error) {
			gotDir = dir
			if name != "git" || args[0] != "pull" {
				t.Fatalf("%s %v", name, args)
			}
			return "Already up to date.", nil
		},
	})
	if ur.Updated != 1 || gotDir != "/tmp/foo" {
		t.Fatalf("%+v dir=%s", ur, gotDir)
	}
}

func TestApplyUpdatesCancel(t *testing.T) {
	report := &Result{
		Entries: []Entry{
			{Name: "x", Status: StatusOutdated, Provenance: ProvSkillLock},
		},
	}
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    false,
		Stdout: &bytes.Buffer{},
		Stdin:  strings.NewReader("n\n"),
		RunCmd: func(dir, name string, args ...string) (string, error) {
			t.Fatal("should not run")
			return "", nil
		},
	})
	if ur.Updated != 0 || len(ur.Items) != 0 {
		t.Fatalf("%+v", ur)
	}
}

func TestChildStreamWritersRedirect(t *testing.T) {
	var buf bytes.Buffer
	outW, errW := childStreamWriters(&buf)
	if outW != &buf || errW != &buf {
		t.Fatalf("redirected stdout should fold stderr into the same writer")
	}
	outW, errW = childStreamWriters(os.Stdout)
	if outW != os.Stdout || errW != os.Stderr {
		t.Fatalf("TTY stdout should keep stderr on os.Stderr")
	}
}

func TestRunCmdStreamingWritesOnlyToProvidedWriters(t *testing.T) {
	var out, errBuf bytes.Buffer
	got, err := runCmdStreaming(&out, &errBuf, "", "/bin/echo", "hello-asm-update")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "hello-asm-update") {
		t.Fatalf("captured=%q", got)
	}
	if !strings.Contains(out.String(), "hello-asm-update") {
		t.Fatalf("stdout writer=%q", out.String())
	}
}

func TestApplyUpdatesNameFilter(t *testing.T) {
	report := &Result{
		Entries: []Entry{
			{Name: "a", Status: StatusOutdated, Provenance: ProvSkillLock},
			{Name: "b", Status: StatusOutdated, Provenance: ProvSkillLock},
		},
	}
	var ran string
	_ = ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Names:  []string{"b"},
		Stdout: &bytes.Buffer{},
		RunCmd: func(dir, name string, args ...string) (string, error) {
			ran = strings.Join(args, " ")
			return "ok", nil
		},
	})
	if !strings.Contains(ran, "b") || strings.Contains(ran, " a ") {
		// args like: --yes skills update b -g -y
		if !strings.Contains(ran, "update b") && !strings.Contains(ran, "update b ") {
			// check list contains b only as skill name after update
			parts := strings.Split(ran, " ")
			foundA, foundB := false, false
			for _, p := range parts {
				if p == "a" {
					foundA = true
				}
				if p == "b" {
					foundB = true
				}
			}
			if foundA || !foundB {
				t.Fatalf("ran=%q", ran)
			}
		}
	}
}
