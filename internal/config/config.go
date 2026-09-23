// Package config loads runtime configuration from the environment.
//
// Every value the process needs is read exactly once, at startup, into a
// Config value. Nothing else in the codebase calls os.Getenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env identifies the deployment environment.
type Env string

const (
	EnvDev     Env = "dev"
	EnvStaging Env = "staging"
	EnvProd    Env = "prod"
)

// Config is the fully resolved runtime configuration.
type Config struct {
	Env            Env
	HTTPAddr       string
	LogLevel       string
	LogFormat      string
	ShutdownGrace  time.Duration
	DatabaseURL    string
	DBMaxConns     int32
	DBMinConns     int32
	JWTSigningKey  string
	JWTIssuer      string
	AccessTokenTTL time.Duration
	ContentDir     string
}

// Load reads configuration from the environment and validates it.
//
// Validation is strict by design: a misconfigured process should fail at
// startup with a precise message rather than at the first request.
func Load() (Config, error) {
	c := Config{
		Env:           Env(str("LODESTAR_ENV", "dev")),
		HTTPAddr:      str("LODESTAR_HTTP_ADDR", ":8080"),
		LogLevel:      str("LODESTAR_LOG_LEVEL", "info"),
		LogFormat:     str("LODESTAR_LOG_FORMAT", "json"),
		DatabaseURL:   str("DATABASE_URL", ""),
		JWTSigningKey: str("LODESTAR_JWT_SIGNING_KEY", ""),
		JWTIssuer:     str("LODESTAR_JWT_ISSUER", "lodestar"),
		ContentDir:    str("LODESTAR_CONTENT_DIR", "./content"),
	}

	var errs []error
	var err error

	if c.ShutdownGrace, err = dur("LODESTAR_SHUTDOWN_GRACE", 15*time.Second); err != nil {
		errs = append(errs, err)
	}
	if c.AccessTokenTTL, err = dur("LODESTAR_ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		errs = append(errs, err)
	}
	if c.DBMaxConns, err = i32("LODESTAR_DB_MAX_CONNS", 10); err != nil {
		errs = append(errs, err)
	}
	if c.DBMinConns, err = i32("LODESTAR_DB_MIN_CONNS", 2); err != nil {
		errs = append(errs, err)
	}

	switch c.Env {
	case EnvDev, EnvStaging, EnvProd:
	default:
		errs = append(errs, fmt.Errorf("LODESTAR_ENV: %q is not one of dev|staging|prod", c.Env))
	}

	// Outside dev these are hard requirements; there is no built-in fallback
	// secret anywhere in this codebase.
	if c.Env != EnvDev {
		if c.JWTSigningKey == "" {
			errs = append(errs, errors.New("LODESTAR_JWT_SIGNING_KEY: required outside dev"))
		}
		if c.LogFormat != "json" {
			errs = append(errs, errors.New("LODESTAR_LOG_FORMAT: must be json outside dev"))
		}
	}
	if c.DBMinConns > c.DBMaxConns {
		errs = append(errs, errors.New("LODESTAR_DB_MIN_CONNS must be <= LODESTAR_DB_MAX_CONNS"))
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return c, nil
}

// IsDev reports whether the process runs in the development environment.
func (c Config) IsDev() bool { return c.Env == EnvDev }

func str(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func dur(key string, def time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	v, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func i32(key string, def int32) (int32, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return int32(v), nil
}
