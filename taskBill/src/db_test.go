package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	dbload "dbload"
	"github.com/go-sql-driver/mysql"
	"mysqlmeta"
)

// runMigrations reads all .sql files from dataMigrate/taskBill/ and executes them
// against the target database in filename order. This is test infrastructure only
// — application code must not define runMigrations (see no-runmigrations-in-code memory).
func runMigrations(dsn, repoRoot string) error {
	migrationsDir := filepath.Join(repoRoot, "dataMigrate", "taskBill")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", migrationsDir, err)
	}

	var sqlFiles []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".sql") {
			sqlFiles = append(sqlFiles, n)
		}
	}
	sort.Strings(sqlFiles)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	for _, f := range sqlFiles {
		path := filepath.Join(migrationsDir, f)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		sql := string(content)
		if strings.TrimSpace(sql) == "" {
			continue
		}
		if _, err := db.Exec(mysqlmeta.StripMySQLClientMeta(sql)); err != nil {
			return fmt.Errorf("execute %s: %w", f, err)
		}
	}

	return nil
}

// setupMySQLTestDB clones a migrated schema template into a unique test database,
// opens the global `db` handle, and returns a cleanup function that drops the clone.
//
// Uses TASKBILL_MYSQL_TEST_DSN env var (host/user/password only, no db name)
// or defaults to registry.yaml credentials.
func setupMySQLTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"

	repoRoot, err := findMonorepoRoot()
	if err != nil {
		t.Fatalf("monorepo root not found — run tests from monorepo root: %v", err)
	}

	rev, err := dbload.HashSQLDir(filepath.Join(repoRoot, "dataMigrate", "taskBill"))
	if err != nil {
		t.Fatalf("hash migrations: %v", err)
	}
	dsn, cloneCleanup, err := prepareCloneDB(func() (string, func(), error) {
		return dbload.PrepareClonedTestDB(adminDSN, "task_bill", rev, func(migrateDSN string) error {
			return runMigrations(migrateDSN, repoRoot)
		})
	})
	if err != nil {
		t.Fatalf("clone test database: %v", err)
	}

	if err := openDB(dsn); err != nil {
		cloneCleanup()
		t.Fatalf("openDB: %v", err)
	}

	return func() {
		if db != nil {
			db.Close()
			db = nil
		}
		cloneCleanup()
	}
}

// prepareCloneDB runs a PrepareClonedTestDB-style clone, retrying bounded
// times only on transient InnoDB deadlocks / lock-wait timeouts (MySQL error
// 1213 "Deadlock found", 1205 "Lock wait timeout exceeded") — the errors
// MySQL explicitly answers with "try restarting transaction".
//
// The nightly sweep intermittently hit Error 1213 while cloning the
// partitioned billing_usage template (INSERT INTO dst SELECT * FROM src under
// REPEATABLE-READ), a one-off lock race that a retry clears. Non-transient
// failures (schema errors, connection refused, …) abort on the first attempt.
func prepareCloneDB(prepare func() (string, func(), error)) (dsn string, cleanup func(), err error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		dsn, cleanup, err = prepare()
		if err == nil {
			return dsn, cleanup, nil
		}
		lastErr = err
		var my *mysql.MySQLError
		if !errors.As(err, &my) || !isTransientCloneErr(my) {
			return "", nil, err
		}
		log.Printf("[taskBill] clone test DB attempt %d/%d hit transient MySQL error %d (%s), retrying",
			attempt, maxAttempts, my.Number, my.Message)
		time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
	}
	return "", nil, lastErr
}

func isTransientCloneErr(my *mysql.MySQLError) bool {
	return my.Number == 1213 || my.Number == 1205
}

func TestPrepareCloneDBRetriesTransientDeadlock(t *testing.T) {
	calls := 0
	dsn, cleanup, err := prepareCloneDB(func() (string, func(), error) {
		calls++
		if calls <= 2 {
			return "", nil, &mysql.MySQLError{
				Number:  1213,
				Message: "Deadlock found when trying to get lock; try restarting transaction",
			}
		}
		return "dsn-ok", func() {}, nil
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
	if dsn != "dsn-ok" {
		t.Fatalf("dsn=%q want dsn-ok", dsn)
	}
	cleanup()
}

func TestPrepareCloneDBRetriesLockWaitTimeout(t *testing.T) {
	calls := 0
	_, _, err := prepareCloneDB(func() (string, func(), error) {
		calls++
		if calls == 1 {
			return "", nil, &mysql.MySQLError{
				Number:  1205,
				Message: "Lock wait timeout exceeded; try restarting transaction",
			}
		}
		return "dsn-ok", func() {}, nil
	})
	if err != nil {
		t.Fatalf("expected lock-wait retry to succeed, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func TestPrepareCloneDBAbortsOnNonTransientError(t *testing.T) {
	calls := 0
	_, _, err := prepareCloneDB(func() (string, func(), error) {
		calls++
		return "", nil, fmt.Errorf("CREATE DATABASE permission denied")
	})
	if err == nil {
		t.Fatal("expected non-transient error to abort immediately")
	}
	if calls != 1 {
		t.Fatalf("expected no retry for non-transient error, got %d calls", calls)
	}
}

// TestSetupMySQLTestDBAppliesFinalSchema checks that cloned setup still lands on
// the post-migrate schema (038 order comments present; 018 dropped pricing package).
func TestSetupMySQLTestDBAppliesFinalSchema(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tableCount := func(table string) int {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?",
			table,
		).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		return n
	}
	if n := tableCount("billing_order_comment"); n != 1 {
		t.Fatalf("expected billing_order_comment (migration 038) to exist after setup, got %d", n)
	}
	if n := tableCount("billing_pricing_package"); n != 0 {
		t.Fatalf("expected billing_pricing_package to be dropped by migration 018, got %d", n)
	}
}
