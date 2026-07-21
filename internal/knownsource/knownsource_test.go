package knownsource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchDWS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "NOTICE"), []byte("DingTalk Workspace CLI (dws)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: dws\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, ok := Match("dws", dir)
	if !ok {
		t.Fatal("expected match")
	}
	if s.OwnerRepo != "DingTalk-Real-AI/dingtalk-workspace-cli" {
		t.Fatalf("repo=%s", s.OwnerRepo)
	}
	if s.FolderPath() != "skills/mono" {
		t.Fatalf("folder=%s", s.FolderPath())
	}
}

func TestMatchDWSRejectsRandom(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: dws\n---\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Match("dws", dir); ok {
		t.Fatal("should not match without NOTICE/fingerprint")
	}
}

func TestGitBlobSHA1(t *testing.T) {
	// echo -n 'hello' | git hash-object --stdin
	// blob 5\0hello → known value
	got := GitBlobSHA1Bytes([]byte("hello"))
	want := "b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestLookupDWSHasUpdateSpec(t *testing.T) {
	s, ok := Lookup("dws")
	if !ok {
		t.Fatal("Lookup dws")
	}
	if s.Update == nil || len(s.Update.Steps) < 2 {
		t.Fatalf("Update=%+v", s.Update)
	}
	if !s.Update.OncePerMachine {
		t.Fatal("OncePerMachine expected")
	}
	if s.OutdatedHint() == "" || s.ActionID() != "dws" {
		t.Fatalf("hint=%q action=%q", s.OutdatedHint(), s.ActionID())
	}
}

func TestLookupMissing(t *testing.T) {
	if _, ok := Lookup("no-such-skill"); ok {
		t.Fatal("expected miss")
	}
}
