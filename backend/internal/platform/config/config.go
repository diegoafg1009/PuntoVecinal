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
	Spaces   Spaces
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

// Spaces holds the DigitalOcean Spaces (S3-compatible) settings.
type Spaces struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type Voyage struct {
	APIKey string
}

// Load reads the configuration. Real environment variables take precedence
// over the values in the .env file, which is optional.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

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
		"spaces.endpoint":          "",
		"spaces.region":            "",
		"spaces.bucket":            "",
		"spaces.accesskey":         "",
		"spaces.secretkey":         "",
		"voyage.apikey":            "",
	}
	for k, d := range defaults {
		v.SetDefault(k, d)
	}

	// The platform convention is a bare PORT rather than APP_PORT.
	if err := v.BindEnv("app.port", "PORT"); err != nil {
		return Config{}, fmt.Errorf("bind PORT: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("read .env: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
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
	return nil
}
