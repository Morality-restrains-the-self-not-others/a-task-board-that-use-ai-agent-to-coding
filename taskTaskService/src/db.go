package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"mysqlmeta"
)

var db *sql.DB

func openDB(dsn string) error {
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		return err
	}
	// Schema/seed: dataMigrate/taskTaskService via 9999 init / `migrate` CLI only.
	return nil
}

func columnExists(table, column string) bool {
	var n int
	_ = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table, column).Scan(&n)
	return n > 0
}

// All DDL and data migrations are exclusively managed via dataMigrate/taskTaskService/*.sql
// files, applied idempotently by runDataMigrate() at startup.
//
// ensureCoreTablesExist (below) is a safety net that detects and repairs externally-dropped
// tables by resetting the migration log so the schema SQL re-applies. It is NOT a migration.
//
// Legacy Go-level migrations have been fully migrated to SQL files.
// See git history for the original Go implementations.

// ensureCoreTablesExist checks that the tasks table exists. If it doesn't but
// data_migrate_log records a prior 001_schema.sql application, the table was
// dropped externally (e.g. manual DROP TABLE, partial db reset). In that case
// we clear the stale log entries so runDataMigrate() re-applies the schema.
// This prevents silent HTTP 500s caused by missing tables despite the migration
// log claiming they were created.
func ensureCoreTablesExist() error {
	// Check if the core tasks table exists.
	var tableExists int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tasks'`).Scan(&tableExists)
	if tableExists > 0 {
		return nil
	}

	// tasks table missing — check data_migrate_log. If the log table doesn't
	// exist yet (fresh DB), let runDataMigrate() handle everything normally.
	var logTableExists int
	_ = db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'data_migrate_log'`).Scan(&logTableExists)
	if logTableExists == 0 {
		return nil
	}

	var logged int
	_ = db.QueryRow(`SELECT COUNT(*) FROM data_migrate_log WHERE step_key = '001_schema.sql'`).Scan(&logged)
	if logged == 0 {
		// Log table exists but no 001_schema.sql record — fresh start, nothing to fix.
		return nil
	}

	log.Printf("[taskTaskService] WARNING: tasks table missing despite 001_schema.sql logged — resetting migration log to re-apply schema")
	if _, err := db.Exec(`DELETE FROM data_migrate_log`); err != nil {
		return fmt.Errorf("reset data_migrate_log: %w", err)
	}
	return nil
}

// All legacy Go-level migrations have been moved to dataMigrate/taskTaskService/*.sql:
//   003_comments_backfill_user_id.sql + 003_comments_rebuild_drop_user_id.sql (formerly migrateCommentsLegacyUserID)
//   004_rhythm_windows_migrate_legacy.sql + 004_rhythm_windows_rebuild_rhythms.sql (formerly migrateLegacyRhythmWindows)
// dropLegacyAITaskCommentsIfEmpty removed — SQLite-specific, no-op on MySQL.
// See git history for the original Go implementations.

// repoRoot resolves the monorepo root for migration paths.
// OPT-20260901-019: delegate to findMonorepoRoot (confload) so clone-run honors
// CONF_ROOT / DEPLOY_ROOT — a cwd walk stops at envs/current/taskTaskService
// whose conf/base.yaml has no sibling conf-local secrets.
func repoRoot() string {
	root, err := findMonorepoRoot()
	if err != nil {
		return "."
	}
	return root
}

// runDataMigrate executes all .sql files from dataMigrate/taskTaskService/
func runDataMigrate(repoRoot string) error {
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskTaskService")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		log.Printf("[taskTaskService] dataMigrate dir not found: %v (skipping)", err)
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS data_migrate_log (
		step_key VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64) NOT NULL DEFAULT ''
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("create data_migrate_log: %w", err)
	}

	for _, name := range files {
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, name).Scan(&exists); err == nil {
			log.Printf("[taskTaskService] dataMigrate: %s (skipped)", name)
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			log.Printf("[taskTaskService] dataMigrate: read %s failed: %v (skipping)", name, err)
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}
		if _, err := tx.Exec(mysqlmeta.StripMySQLClientMeta(string(raw))); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO data_migrate_log (step_key, applied_at) VALUES (?, NOW())`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
		log.Printf("[taskTaskService] dataMigrate: %s (applied)", name)
	}
	return nil
}
