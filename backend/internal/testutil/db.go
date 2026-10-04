// Package testutil provides shared helpers for Postgres- and Redis-backed tests.
//
// Postgres integration tests get an isolated database per test package:
// because `go test ./...` runs packages concurrently, each test binary creates
// its own `temu_in_test_<pkg>_<random>` database from the admin connection in
// TEST_DATABASE_URL, migrates it, truncates it before every test, and drops it
// when the package finishes.
//
// Usage in a package with integration tests:
//
//	func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }
//
//	func TestSomething(t *testing.T) {
//		db := testutil.DB(t) // skips when TEST_DATABASE_URL is unset
//		...
//	}
//
// Integration tests must not call t.Parallel(): they share one database and
// every DB(t) call truncates all tables.
//
// The role in TEST_DATABASE_URL needs CREATEDB. With REQUIRE_INTEGRATION=1 a
// missing TEST_DATABASE_URL fails the test instead of skipping it.
package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"temu.in/backend/internal/database"
)

const (
	envDatabaseURL        = "TEST_DATABASE_URL"
	envRequireIntegration = "REQUIRE_INTEGRATION"
	dbNamePrefix          = "temu_in_test_"
)

var (
	mainInstalled bool

	setupOnce sync.Once
	setupErr  error
	pkgDB     *database.DB
	adminURL  string
	dbName    string
)

var nonIdent = regexp.MustCompile(`[^a-z0-9_]+`)

// Main runs the package's tests and drops the per-package database afterwards.
// Every package that calls DB must install it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }
func Main(m *testing.M) int {
	mainInstalled = true
	code := m.Run()
	if err := teardown(); err != nil {
		fmt.Fprintf(os.Stderr, "testutil: teardown: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

// DB returns a migrated, freshly truncated database for this test package.
// It skips the test when TEST_DATABASE_URL is unset (fails instead when
// REQUIRE_INTEGRATION=1).
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	rawURL := os.Getenv(envDatabaseURL)
	if rawURL == "" {
		if os.Getenv(envRequireIntegration) == "1" {
			t.Fatal("integration tests required but TEST_DATABASE_URL is unset")
		}
		t.Skip("TEST_DATABASE_URL not set")
	}
	if !mainInstalled {
		t.Fatal("testutil.Main must be installed via TestMain in this package")
	}

	setupOnce.Do(func() { setupErr = setup(rawURL) })
	if setupErr != nil {
		t.Fatalf("testutil: set up test database: %v", setupErr)
	}

	if err := truncateAll(pkgDB.DB); err != nil {
		t.Fatalf("testutil: truncate tables: %v", err)
	}
	return pkgDB.DB
}

// packageDBName derives the per-package database name from the test binary
// name (e.g. "booking.test" -> "temu_in_test_booking_<8 hex>").
func packageDBName(binary string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(binary), ".test")
	base = strings.TrimSuffix(base, ".test.exe")
	pkg := strings.Trim(nonIdent.ReplaceAllString(strings.ToLower(base), "_"), "_")
	if pkg == "" {
		pkg = "pkg"
	}
	// Postgres identifiers are limited to 63 bytes.
	if maxPkg := 63 - len(dbNamePrefix) - 1 - 8; len(pkg) > maxPkg {
		pkg = pkg[:maxPkg]
	}
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate database suffix: %w", err)
	}
	return dbNamePrefix + pkg + "_" + hex.EncodeToString(buf), nil
}

// withDatabase returns rawURL with its path replaced by name.
func withDatabase(rawURL, name string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}
	u.Path = "/" + name
	u.RawPath = ""
	return u.String(), nil
}

func openQuiet(dsn string) (*database.DB, error) {
	db, err := database.Connect(dsn)
	if err != nil {
		return nil, err
	}
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	return db, nil
}

func setup(rawURL string) error {
	name, err := packageDBName(os.Args[0])
	if err != nil {
		return err
	}
	pkgURL, err := withDatabase(rawURL, name)
	if err != nil {
		return err
	}

	admin, err := openQuiet(rawURL)
	if err != nil {
		return fmt.Errorf("connect admin database: %w", err)
	}
	defer admin.Close()

	// name is built only from [a-z0-9_], so quoting it is safe.
	if err := admin.Exec(`CREATE DATABASE "` + name + `"`).Error; err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	adminURL, dbName = rawURL, name

	db, err := openQuiet(pkgURL)
	if err != nil {
		return fmt.Errorf("connect test database: %w", err)
	}
	pkgDB = db
	if err := db.RunMigrations(); err != nil {
		return fmt.Errorf("migrate test database: %w", err)
	}
	return nil
}

func teardown() error {
	if dbName == "" {
		return nil
	}
	if pkgDB != nil {
		_ = pkgDB.Close()
		pkgDB = nil
	}
	admin, err := openQuiet(adminURL)
	if err != nil {
		return fmt.Errorf("connect admin database: %w", err)
	}
	defer admin.Close()

	if err := admin.Exec(
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()`,
		dbName,
	).Error; err != nil {
		return fmt.Errorf("terminate connections to %s: %w", dbName, err)
	}
	if err := admin.Exec(`DROP DATABASE IF EXISTS "` + dbName + `"`).Error; err != nil {
		return fmt.Errorf("drop database %s: %w", dbName, err)
	}
	dbName = ""
	return nil
}

func truncateAll(db *gorm.DB) error {
	var tables []string
	if err := db.Raw(
		`SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'schema_migrations' ORDER BY tablename`,
	).Scan(&tables).Error; err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	if len(tables) == 0 {
		return nil
	}
	quoted := make([]string, len(tables))
	for i, name := range tables {
		quoted[i] = `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
	return db.Exec("TRUNCATE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE").Error
}
