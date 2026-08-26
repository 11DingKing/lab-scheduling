package config_test

import (
	"github.com/11DingKing/lab-scheduling/internal/config"
	"os"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	for _, k := range []string{"HTTP_ADDR", "DB_PATH", "MIGRATIONS_PATH", "SESSION_TTL"} {
		_ = os.Unsetenv(k)
	}
	cfg := config.Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatal(cfg.HTTPAddr)
	}
	if cfg.DBPath != "lab.db" {
		t.Fatal(cfg.DBPath)
	}
	if cfg.MigrationsPath != "migrations" {
		t.Fatal(cfg.MigrationsPath)
	}
	if cfg.SessionTTL != 8*time.Hour {
		t.Fatal(cfg.SessionTTL)
	}
}
func TestEnvironmentOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("MIGRATIONS_PATH", "/tmp/migrations")
	t.Setenv("SESSION_TTL", "30m")
	cfg := config.Load()
	if cfg.HTTPAddr != ":9090" || cfg.DBPath != "/tmp/test.db" || cfg.MigrationsPath != "/tmp/migrations" || cfg.SessionTTL != 30*time.Minute {
		t.Fatalf("%+v", cfg)
	}
}
func TestInvalidTTLUsesDefault(t *testing.T) {
	t.Setenv("SESSION_TTL", "bad")
	if got := config.Load().SessionTTL; got != 8*time.Hour {
		t.Fatal(got)
	}
}
