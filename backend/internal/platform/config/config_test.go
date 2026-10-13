package config

import (
	"testing"
	"time"
)

func TestLoadReadsEnvironment(t *testing.T) {
	t.Chdir(t.TempDir()) // no .env file here
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("DATABASE_MAXCONNLIFETIME", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 9090 || cfg.Database.URL != "postgres://x" || cfg.Database.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid port")
	}
}
