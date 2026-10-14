package config

import (
	"testing"
	"time"
)

// setSupabase sets the Supabase variables that Load requires.
func setSupabase(t *testing.T) {
	t.Helper()
	t.Setenv("SUPABASE_URL", "http://127.0.0.1:54321")
	t.Setenv("SUPABASE_JWKSURL", "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json")
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Chdir(t.TempDir()) // no .env file here
	setSupabase(t)
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("DATABASE_MAXCONNLIFETIME", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Supabase.URL != "http://127.0.0.1:54321" {
		t.Fatalf("unexpected supabase config: %+v", cfg.Supabase)
	}
	if cfg.App.Port != 9090 || cfg.Database.URL != "postgres://x" || cfg.Database.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Chdir(t.TempDir())
	setSupabase(t)
	t.Setenv("PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoadRequiresSupabase(t *testing.T) {
	tests := map[string]map[string]string{
		"missing url":     {"SUPABASE_JWKSURL": "http://x/jwks"},
		"missing jwksurl": {"SUPABASE_URL": "http://x"},
	}
	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected error for missing Supabase setting")
			}
		})
	}
}
