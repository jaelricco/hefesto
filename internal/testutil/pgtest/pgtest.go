//go:build integration

// Package pgtest gives integration tests a real, migrated Postgres.
//
// By default it starts postgres:16-alpine with testcontainers — once per test
// binary — and applies every migration in db/migrations to a template
// database. Each call to New then creates a fresh database from that
// template, so tests are isolated from each other and cheap to set up.
//
// HEFESTO_TEST_DATABASE_URL, if set, points at an existing Postgres 16 server
// to use instead of a container (for environments without Docker). The role
// must be allowed to create databases. It is still a real Postgres; nothing
// here mocks SQL.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Image is the Postgres the tests run against; it matches Compose and prod.
const Image = "postgres:16-alpine"

var (
	once     sync.Once
	server   *url.URL // admin connection to the server
	template string
	setupErr error
	counter  atomic.Int64
)

// MigrationsDir is the absolute path of db/migrations.
func MigrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}

// RepoRoot is the absolute path of the repository.
func RepoRoot() string {
	return filepath.Join(MigrationsDir(), "..", "..")
}

// New returns a pool on a fresh, fully migrated database that is dropped when
// the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool, _ := NewWithURL(t)
	return pool
}

// NewWithURL is New, also returning the database's connection URL.
func NewWithURL(t testing.TB) (*pgxpool.Pool, string) {
	t.Helper()
	once.Do(setup)
	if setupErr != nil {
		t.Fatalf("pgtest setup: %v", setupErr)
	}

	name := fmt.Sprintf("t_%d_%d", os.Getpid(), counter.Add(1))
	ctx := context.Background()
	if err := admin(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, template)); err != nil {
		t.Fatalf("creating test database: %v", err)
	}
	dbURL := withDB(name)

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		_ = admin(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	})
	return pool, dbURL
}

// NewEmpty returns the URL of a fresh database with no migrations applied,
// for tests of the migrations themselves.
func NewEmpty(t testing.TB) string {
	t.Helper()
	once.Do(setup)
	if setupErr != nil {
		t.Fatalf("pgtest setup: %v", setupErr)
	}
	name := fmt.Sprintf("e_%d_%d", os.Getpid(), counter.Add(1))
	if err := admin(context.Background(), "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating empty database: %v", err)
	}
	t.Cleanup(func() {
		_ = admin(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
	})
	return withDB(name)
}

// Migrate applies every migration to the database at dbURL.
func Migrate(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("opening %s: %w", dbURL, err)
	}
	defer func() { _ = db.Close() }()
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db, MigrationsDir()); err != nil {
		return fmt.Errorf("migrating: %w", err)
	}
	return nil
}

func setup() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	raw := os.Getenv("HEFESTO_TEST_DATABASE_URL")
	if raw == "" {
		c, err := tcpostgres.Run(ctx, Image,
			tcpostgres.WithDatabase("hefesto"),
			tcpostgres.WithUsername("hefesto"),
			tcpostgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).WithStartupTimeout(time.Minute)),
		)
		if err != nil {
			setupErr = fmt.Errorf("starting %s (set HEFESTO_TEST_DATABASE_URL to use an existing server): %w", Image, err)
			return
		}
		// The container lives as long as the test binary; Ryuk reaps it.
		raw, err = c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setupErr = fmt.Errorf("container connection string: %w", err)
			return
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		setupErr = fmt.Errorf("parsing database url: %w", err)
		return
	}
	server = u

	template = "tmpl_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin(ctx, "CREATE DATABASE "+template); err != nil {
		setupErr = fmt.Errorf("creating template database: %w", err)
		return
	}
	if err := Migrate(withDB(template)); err != nil {
		setupErr = err
	}
}

func withDB(name string) string {
	u := *server
	u.Path = "/" + name
	return u.String()
}

func admin(ctx context.Context, stmt string) error {
	conn, err := pgx.Connect(ctx, server.String())
	if err != nil {
		return fmt.Errorf("admin connection: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("%s: %w", stmt, err)
	}
	return nil
}

// Run is for TestMain: it runs the tests, then drops the template database so
// a shared server is left as it was found.
func Run(m *testing.M) int {
	code := m.Run()
	if template != "" && server != nil {
		_ = admin(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", template))
	}
	return code
}
