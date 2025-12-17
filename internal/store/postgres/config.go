package postgres

import (
	"os"
	"strconv"
	"time"
)

// Config captures PostgreSQL connection tuning options.
type Config struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnIdleTime   time.Duration
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
}

// FromEnv builds a Config by reading well-known environment variables.
func FromEnv() Config {
	cfg := Config{
		URL: os.Getenv("DATABASE_URL"),
	}

	if v := parseEnvInt32("PG_MAX_CONNS"); v > 0 {
		cfg.MaxConns = v
	}
	if v := parseEnvInt32("PG_MIN_CONNS"); v > 0 {
		cfg.MinConns = v
	}
	if d := parseEnvDuration("PG_MAX_CONN_IDLE", time.Minute); d > 0 {
		cfg.MaxConnIdleTime = d
	}
	if d := parseEnvDuration("PG_MAX_CONN_LIFETIME", time.Hour); d > 0 {
		cfg.MaxConnLifetime = d
	}
	if d := parseEnvDuration("PG_HEALTHCHECK_PERIOD", 30*time.Second); d > 0 {
		cfg.HealthCheckPeriod = d
	}

	return cfg
}

func parseEnvInt32(key string) int32 {
	raw := os.Getenv(key)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}

func parseEnvDuration(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return d
}
