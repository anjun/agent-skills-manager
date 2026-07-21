package outdated

import (
	"bytes"
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

func TestApplyUpdatesDWS(t *testing.T) {
	report := &Result{
		Entries: []Entry{
			{Name: "dws", Status: StatusOutdated, Provenance: ProvKnown,
				Remote: "https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli"},
		},
	}
	var cmds []string
	ur := ApplyUpdates(report, UpdateOptions{
		Yes:    true,
		Stdout: &bytes.Buffer{},
		RunCmd: func(dir, name string, args ...string) (string, error) {
			cmds = append(cmds, name+" "+strings.Join(args, " "))
			return "ok", nil
		},
	})
	// LookPath for dws may or may not find real binary; if found, skill setup is called
	if len(ur.Items) != 1 {
		t.Fatalf("%+v", ur.Items)
	}
	it := ur.Items[0]
	if it.Action != "dws" {
		t.Fatalf("action=%s", it.Action)
	}
	// either setup ran, or skip if no dws
	if !it.OK && !it.Skipped {
		t.Fatalf("item=%+v cmds=%v", it, cmds)
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
