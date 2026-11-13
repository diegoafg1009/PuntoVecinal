// Package config loads the application settings from the environment, using
// an optional .env file as a fallback for local development.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      App
	Database Database
	Supabase Supabase
	S3       S3
	Voyage   Voyage
}

type App struct {
	Env  string
	Port int
}

type Database struct {
	URL             string
	TransactionPool bool
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

type Supabase struct {
	URL     string
	JWKSURL string
}

// S3 holds the settings of the S3-compatible object storage (RustFS locally,
// any S3 provider in production).
type S3 struct {
	// Endpoint is the API endpoint the backend talks to.
	Endpoint string
	// PublicEndpoint is the endpoint put in signed links, for when browsers
	// reach the server by another address. Defaults to Endpoint.
	PublicEndpoint string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	// ForcePathStyle addresses objects as host/bucket/key instead of
	// bucket.host/key.
	ForcePathStyle bool
}

type Voyage struct {
	APIKey string
}

// Load reads the configuration. Real environment variables take precedence
// over the values in the .env file, which is optional.
func Load() (Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// viper keeps .env keys flat (SUPABASE_URL becomes "supabase_url"), so
	// they never match the nested keys below. Read the file on its own and
	// use its values as defaults, which real environment variables override.
	file := viper.New()
	file.SetConfigFile(".env")
	file.SetConfigType("env")
	if err := file.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("read .env: %w", err)
		}
	}

	// Every key needs a default so AutomaticEnv resolves it on Unmarshal.
	defaults := map[string]any{
		"app.env":                  "development",
		"app.port":                 8080,
		"database.url":             "",
		"database.transactionpool": false,
		"database.maxconns":        0,
		"database.minconns":        0,
		"database.maxconnlifetime": "0s",
		"database.maxconnidletime": "0s",
		"database.connecttimeout":  "0s",
		"supabase.url":             "",
		"supabase.jwksurl":         "",
		"s3.endpoint":              "",
		"s3.publicendpoint":        "",
		"s3.region":                "us-east-1",
		"s3.bucket":                "",
		"s3.accesskey":             "",
		"s3.secretkey":             "",
		"s3.forcepathstyle":        false,
		"voyage.apikey":            "",
	}
	for k, d := range defaults {
		envName := strings.ReplaceAll(k, ".", "_")
		if k == "app.port" {
			envName = "port"
		}
		if file.IsSet(envName) {
			d = file.Get(envName)
		}
		v.SetDefault(k, d)
	}

	// The platform convention is a bare PORT rather than APP_PORT.
	if err := v.BindEnv("app.port", "PORT"); err != nil {
		return Config{}, fmt.Errorf("bind PORT: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.S3.PublicEndpoint == "" {
		cfg.S3.PublicEndpoint = cfg.S3.Endpoint
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.App.Port < 1 || c.App.Port > 65535 {
		return fmt.Errorf("invalid PORT: %d", c.App.Port)
	}
	if c.Supabase.URL == "" {
		return errors.New("SUPABASE_URL is required")
	}
	if c.Supabase.JWKSURL == "" {
		return errors.New("SUPABASE_JWKSURL is required")
	}
	for _, setting := range []struct{ name, value string }{
		{"S3_ENDPOINT", c.S3.Endpoint},
		{"S3_BUCKET", c.S3.Bucket},
		{"S3_ACCESSKEY", c.S3.AccessKey},
		{"S3_SECRETKEY", c.S3.SecretKey},
	} {
		if setting.value == "" {
			return fmt.Errorf("%s is required", setting.name)
		}
	}
	return nil
}
