package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setSupabase sets the Supabase and S3 variables that Load requires.
func setSupabase(t *testing.T) {
	t.Helper()
	t.Setenv("SUPABASE_URL", "http://127.0.0.1:54321")
	t.Setenv("SUPABASE_JWKSURL", "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json")
	setS3(t)
}

func setS3(t *testing.T) {
	t.Helper()
	t.Setenv("S3_ENDPOINT", "http://127.0.0.1:9000")
	t.Setenv("S3_BUCKET", "punto-vecinal")
	t.Setenv("S3_ACCESSKEY", "rustfsadmin")
	t.Setenv("S3_SECRETKEY", "rustfsadmin")
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
			setS3(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected error for missing Supabase setting")
			}
		})
	}
}

func TestLoadS3Defaults(t *testing.T) {
	t.Chdir(t.TempDir())
	setSupabase(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	s3 := cfg.S3
	if s3.Endpoint != "http://127.0.0.1:9000" || s3.Bucket != "punto-vecinal" || s3.AccessKey != "rustfsadmin" || s3.SecretKey != "rustfsadmin" {
		t.Fatalf("unexpected S3 config: %+v", s3)
	}
	if s3.Region != "us-east-1" {
		t.Errorf("Region = %q, want the default us-east-1", s3.Region)
	}
	if s3.ForcePathStyle {
		t.Error("ForcePathStyle should default to false")
	}
	if s3.PublicEndpoint != s3.Endpoint {
		t.Errorf("PublicEndpoint = %q, want it to fall back to Endpoint", s3.PublicEndpoint)
	}
}

func TestLoadS3Overrides(t *testing.T) {
	t.Chdir(t.TempDir())
	setSupabase(t)
	t.Setenv("S3_PUBLICENDPOINT", "http://localhost:9000")
	t.Setenv("S3_REGION", "sa-east-1")
	t.Setenv("S3_FORCEPATHSTYLE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.PublicEndpoint != "http://localhost:9000" || cfg.S3.Region != "sa-east-1" || !cfg.S3.ForcePathStyle {
		t.Fatalf("unexpected S3 config: %+v", cfg.S3)
	}
}

func TestLoadRequiresS3(t *testing.T) {
	for _, name := range []string{"S3_ENDPOINT", "S3_BUCKET", "S3_ACCESSKEY", "S3_SECRETKEY"} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setSupabase(t)
			t.Setenv(name, "")
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("err = %v, want one naming %s", err, name)
			}
		})
	}
}

func TestLoadReadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	env := "PORT=9191\n" +
		"SUPABASE_URL=http://file\n" +
		"SUPABASE_JWKSURL=http://file/jwks\n" +
		"S3_ENDPOINT=http://file:9000\n" +
		"S3_BUCKET=bucket\n" +
		"S3_ACCESSKEY=key\n" +
		"S3_SECRETKEY=secret\n" +
		"S3_FORCEPATHSTYLE=true\n" +
		"DATABASE_MAXCONNLIFETIME=15m\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("S3_BUCKET", "from-env") // real variables win over .env

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 9191 || cfg.Supabase.URL != "http://file" || cfg.Database.MaxConnLifetime != 15*time.Minute {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.S3.Endpoint != "http://file:9000" || !cfg.S3.ForcePathStyle || cfg.S3.Bucket != "from-env" {
		t.Fatalf("unexpected S3 config: %+v", cfg.S3)
	}
}
