package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyState(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".local", "state", LegacyAppName)
	tx := filepath.Join(legacy, "transactions")
	if err := os.MkdirAll(tx, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tx, "x.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	cfg.Home = home
	cfg.StateDir = filepath.Join(home, ".local", "state", AppName)

	if err := MigrateLegacyState(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfg.StateDir, "transactions", "x.json")); err != nil {
		t.Fatalf("expected migrated file: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy dir should be gone after rename")
	}
}
