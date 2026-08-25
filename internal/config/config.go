package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddr, DBPath, MigrationsPath string
	SessionTTL                       time.Duration
}

func Load() Config {
	ttl, err := time.ParseDuration(value("SESSION_TTL", "8h"))
	if err != nil {
		ttl = 8 * time.Hour
	}
	return Config{HTTPAddr: value("HTTP_ADDR", ":8080"), DBPath: value("DB_PATH", "lab.db"), MigrationsPath: value("MIGRATIONS_PATH", "migrations"), SessionTTL: ttl}
}
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
