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

	RefreshTokenTTL      time.Duration
	Argon2MemoryKiB      uint32
	Argon2Iterations     uint32
	Argon2Parallelism    uint8
	AppleClientIDs       []string // accepted `aud` values for Sign in with Apple
	AccountDeletionGrace time.Duration
	TrustProxyHeaders    bool // take the client IP from X-Forwarded-For (behind Caddy)

	// Object storage for media. Media is off when S3Endpoint is empty, which
	// only dev allows.
	S3Endpoint       string
	S3PublicEndpoint string // the address clients reach, if it differs
	S3Region         string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3PathStyle      bool
	S3PresignTTL     time.Duration
}

// MediaEnabled reports whether object storage is configured.
func (c Config) MediaEnabled() bool { return c.S3Endpoint != "" }

// Load reads configuration from the environment and validates it.
//
// Validation is strict by design: a misconfigured process should fail at
// startup with a precise message rather than at the first request.
func Load() (Config, error) {
	c := Config{
		Env:           Env(str("HEFESTO_ENV", "dev")),
		HTTPAddr:      str("HEFESTO_HTTP_ADDR", ":8080"),
		LogLevel:      str("HEFESTO_LOG_LEVEL", "info"),
		LogFormat:     str("HEFESTO_LOG_FORMAT", "json"),
		DatabaseURL:   str("DATABASE_URL", ""),
		JWTSigningKey: str("HEFESTO_JWT_SIGNING_KEY", ""),
		JWTIssuer:     str("HEFESTO_JWT_ISSUER", "hefesto"),
		ContentDir:    str("HEFESTO_CONTENT_DIR", "./content"),
	}

	var errs []error
	var err error

	if c.ShutdownGrace, err = dur("HEFESTO_SHUTDOWN_GRACE", 15*time.Second); err != nil {
		errs = append(errs, err)
	}
	if c.AccessTokenTTL, err = dur("HEFESTO_ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		errs = append(errs, err)
	}
	if c.RefreshTokenTTL, err = dur("HEFESTO_REFRESH_TOKEN_TTL", 90*24*time.Hour); err != nil {
		errs = append(errs, err)
	}
	if c.AccountDeletionGrace, err = dur("HEFESTO_ACCOUNT_DELETION_GRACE", 30*24*time.Hour); err != nil {
		errs = append(errs, err)
	}
	var n int32
	if n, err = i32("HEFESTO_ARGON2_MEMORY_KIB", 64*1024); err != nil {
		errs = append(errs, err)
	}
	c.Argon2MemoryKiB = uint32(n) //nolint:gosec // validated below
	if n, err = i32("HEFESTO_ARGON2_ITERATIONS", 3); err != nil {
		errs = append(errs, err)
	}
	c.Argon2Iterations = uint32(n) //nolint:gosec // validated below
	if n, err = i32("HEFESTO_ARGON2_PARALLELISM", 2); err != nil {
		errs = append(errs, err)
	}
	if n < 1 || n > 255 {
		errs = append(errs, errors.New("HEFESTO_ARGON2_PARALLELISM: between 1 and 255"))
	} else {
		c.Argon2Parallelism = uint8(n)
	}
	if c.Argon2MemoryKiB < 8*1024 || c.Argon2Iterations < 1 {
		errs = append(errs, errors.New("HEFESTO_ARGON2_*: memory must be at least 8192 KiB and iterations at least 1"))
	}
	for _, id := range strings.Split(str("HEFESTO_APPLE_CLIENT_ID", ""), ",") {
		if id = strings.TrimSpace(id); id != "" {
			c.AppleClientIDs = append(c.AppleClientIDs, id)
		}
	}
	c.TrustProxyHeaders = str("HEFESTO_TRUST_PROXY_HEADERS", "false") == "true"
	c.S3Endpoint = str("HEFESTO_S3_ENDPOINT", "")
	c.S3PublicEndpoint = str("HEFESTO_S3_PUBLIC_ENDPOINT", "")
	c.S3Region = str("HEFESTO_S3_REGION", "us-east-1")
	c.S3Bucket = str("HEFESTO_S3_BUCKET", "hefesto-media")
	c.S3AccessKey = str("HEFESTO_S3_ACCESS_KEY", "")
	c.S3SecretKey = str("HEFESTO_S3_SECRET_KEY", "")
	c.S3PathStyle = str("HEFESTO_S3_USE_PATH_STYLE", "false") == "true"
	if c.S3PresignTTL, err = dur("HEFESTO_S3_PRESIGN_TTL", 15*time.Minute); err != nil {
		errs = append(errs, err)
	} else if c.S3PresignTTL < time.Minute || c.S3PresignTTL > 7*24*time.Hour {
		errs = append(errs, errors.New("HEFESTO_S3_PRESIGN_TTL: between 1m and 168h"))
	}
	if c.MediaEnabled() && (c.S3AccessKey == "" || c.S3SecretKey == "") {
		errs = append(errs, errors.New("HEFESTO_S3_ACCESS_KEY and HEFESTO_S3_SECRET_KEY: required with HEFESTO_S3_ENDPOINT"))
	}
	if c.DBMaxConns, err = i32("HEFESTO_DB_MAX_CONNS", 10); err != nil {
		errs = append(errs, err)
	}
	if c.DBMinConns, err = i32("HEFESTO_DB_MIN_CONNS", 2); err != nil {
		errs = append(errs, err)
	}

	switch c.Env {
	case EnvDev, EnvStaging, EnvProd:
	default:
		errs = append(errs, fmt.Errorf("HEFESTO_ENV: %q is not one of dev|staging|prod", c.Env))
	}

	// Outside dev these are hard requirements; there is no built-in fallback
	// secret anywhere in this codebase.
	if c.Env != EnvDev {
		if len(c.JWTSigningKey) < 32 {
			errs = append(errs, errors.New("HEFESTO_JWT_SIGNING_KEY: required outside dev, at least 32 characters"))
		}
		if c.LogFormat != "json" {
			errs = append(errs, errors.New("HEFESTO_LOG_FORMAT: must be json outside dev"))
		}
		if !c.MediaEnabled() {
			errs = append(errs, errors.New("HEFESTO_S3_ENDPOINT: required outside dev"))
		}
	}
	if c.DBMinConns > c.DBMaxConns {
		errs = append(errs, errors.New("HEFESTO_DB_MIN_CONNS must be <= HEFESTO_DB_MAX_CONNS"))
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
