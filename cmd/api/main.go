// Command api is the Hefesto HTTP server.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // IANA zones even on an image without tzdata

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/config"
	lhttp "github.com/jaelricco/hefesto/internal/http"
	"github.com/jaelricco/hefesto/internal/store"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local server and exit (used by the container healthcheck)")
	flag.Parse()

	if *healthcheck {
		os.Exit(probe())
	}

	if err := run(); err != nil {
		// The only place in the codebase that decides the process exit code.
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(newLogger(cfg))

	slog.Info("starting hefesto api",
		"version", version, "commit", commit, "env", cfg.Env, "addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := lhttp.RouterDeps{Version: version, Commit: commit}
	if cfg.DatabaseURL == "" {
		// Allowed so the binary can start without a database in dev; /readyz
		// then reports 503, which keeps it out of rotation anywhere real.
		slog.Warn("DATABASE_URL is not set; /readyz will report not ready")
	} else {
		openCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		pool, err := store.Open(openCtx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
		cancel()
		if err != nil {
			return fmt.Errorf("database: %w", err)
		}
		defer pool.Close()
		deps.DB = pool
		if err := wireAPI(ctx, cfg, &deps, store.New(pool)); err != nil {
			return err
		}
	}
	router := lhttp.NewRouter(deps)

	srv := &stdhttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received", "grace", cfg.ShutdownGrace)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	slog.Info("stopped cleanly")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if cfg.LogFormat == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(h).With("service", "hefesto-api")
}

// probe implements `api -healthcheck` so the container image needs no curl.
func probe() int {
	addr := os.Getenv("HEFESTO_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	client := &stdhttp.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1" + addr + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != stdhttp.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

// wireAPI builds the authenticated API: token issuer, Apple verifier, auth
// service, request schemas, and the background reaper for deleted accounts.
func wireAPI(ctx context.Context, cfg config.Config, deps *lhttp.RouterDeps, st *store.Store) error {
	key := []byte(cfg.JWTSigningKey)
	if len(key) == 0 && cfg.IsDev() {
		// Dev only (config refuses this elsewhere): tokens die with the process.
		key = make([]byte, 48)
		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("generating dev signing key: %w", err)
		}
		slog.Warn("HEFESTO_JWT_SIGNING_KEY is not set; using an ephemeral key, so tokens will not survive a restart")
	}
	issuer, err := auth.NewIssuer(key, cfg.JWTIssuer, cfg.AccessTokenTTL)
	if err != nil {
		return err
	}
	var apple *auth.AppleVerifier
	if len(cfg.AppleClientIDs) > 0 {
		apple = auth.NewAppleVerifier(cfg.AppleClientIDs, auth.NewJWKS(auth.AppleKeysURL, nil))
	}
	svc, err := auth.NewService(st, issuer, apple, auth.Config{
		RefreshTTL: cfg.RefreshTokenTTL,
		Argon2: auth.Argon2Params{
			MemoryKiB: cfg.Argon2MemoryKiB, Iterations: cfg.Argon2Iterations, Parallelism: cfg.Argon2Parallelism,
		},
	})
	if err != nil {
		return err
	}
	schemas, err := lhttp.LoadSchemas()
	if err != nil {
		return err
	}
	deps.Auth, deps.Store, deps.Schemas = svc, st, schemas
	deps.TrustProxy, deps.DeletionGrace = cfg.TrustProxyHeaders, cfg.AccountDeletionGrace

	go reapDeletedAccounts(ctx, st, cfg.AccountDeletionGrace)
	return nil
}

// reapDeletedAccounts hard-deletes accounts past their grace period, hourly.
// It is idempotent and advisory-locked, so running it in every API process
// is safe.
func reapDeletedAccounts(ctx context.Context, st *store.Store, grace time.Duration) {
	tick := time.NewTicker(time.Hour)
	defer tick.Stop()
	for {
		n, err := st.ReapDeletedUsers(ctx, grace)
		switch {
		case err != nil && ctx.Err() == nil:
			slog.Error("reaping deleted accounts failed", "error", err)
		case n > 0:
			slog.Info("reaped deleted accounts", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
