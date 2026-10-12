package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "movies")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("DB_NAME", "movies")
	t.Setenv("DB_PORT", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("ALLOWED_ORIGIN", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPort != 5432 || cfg.AppPort != 8080 {
		t.Errorf("ports = (%d, %d), want (5432, 8080)", cfg.DBPort, cfg.AppPort)
	}
	if cfg.DBMaxOpenConns != 25 || cfg.DBMaxIdleConns != 25 {
		t.Errorf("pool sizes = (%d, %d), want (25, 25)", cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime != 30*time.Minute || cfg.DBConnMaxIdleTime != 5*time.Minute {
		t.Errorf("pool durations = (%s, %s), want (30m, 5m)", cfg.DBConnMaxLifetime, cfg.DBConnMaxIdleTime)
	}
	if cfg.DBSSLMode != "disable" || cfg.AllowedOrigin != "*" {
		t.Errorf("defaults = (%q, %q), want (%q, %q)", cfg.DBSSLMode, cfg.AllowedOrigin, "disable", "*")
	}
	if got, want := cfg.PostgresDSN(), "postgres://movies:test-password@localhost:5432/movies?sslmode=disable"; got != want {
		t.Errorf("PostgresDSN() = %q, want %q", got, want)
	}
}

func TestLoadMissingRequiredVariable(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "movies")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "movies")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("Load() error = %v, want missing DB_PASSWORD error", err)
	}
}
