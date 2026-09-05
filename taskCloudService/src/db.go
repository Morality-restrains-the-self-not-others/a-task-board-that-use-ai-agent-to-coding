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
	if err != nil { return err }
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil { return err }
	// Schema/seed: dataMigrate/taskCloudService via 9999 init / `go run ./src migrate` only.
	// Business process must not auto-migrate (see 40_app_process_independent_of_db_migrate.md).
	if err := ensureRecommendedProvidersSeeded(); err != nil { log.Printf("[taskCloudService] verify recommended providers: %v", err) }
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

// All DDL and data migrations are now exclusively managed via dataMigrate/taskCloudService/*.sql
// files, applied via 9999 init / `go run ./src migrate` (not on business openDB).
// Legacy Go-level migrations have been fully migrated to SQL files.
// See git history for the original Go implementations.

// ensureFeatureParamsSchema verifies that dataMigrate ran successfully by checking
// for the existence of core feature-params tables. If tables are missing despite
// dataMigrate claiming they were created, the service refuses to start with a
// clear fatal message instead of silently creating tables via inline DDL.
//
// DDL is the single source of truth in dataMigrate/taskCloudService/001_schema.sql.
// This function is a safety net for the edge case where repoRoot resolution diverges
// and runDataMigrate() cannot find the dataMigrate directory.
func ensureFeatureParamsSchema(conn *sql.DB) error {
	coreTables := []string{
		"cloud_tenant_feature_params",
		"cloud_sub_token_providers",
		"cloud_recommended_llm_providers",
	}
	for _, table := range coreTables {
		var count int
		if err := conn.QueryRow(
			`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`,
			table,
		).Scan(&count); err != nil {
			return fmt.Errorf("ensureFeatureParamsSchema: cannot check table %s: %w", table, err)
		}
		if count == 0 {
			return fmt.Errorf(
				"feature-params table %q is missing — dataMigrate may not have run. "+
					"Run: cd dataMigrate/taskCloudService && mysql < 001_schema.sql",
				table,
			)
		}
	}
	return nil
}

// repoRoot resolves the monorepo root by delegating to the canonical
// findMonorepoRoot() in config.go. Uses a single shared implementation
// so that path resolution is always consistent between config loading
// and dataMigrate. See OPT-20260731-019.
func repoRoot() string {
	root, err := findMonorepoRoot()
	if err != nil || root == "" {
		return "."
	}
	return root
}

// runDataMigrate executes all .sql files from dataMigrate/taskCloudService/
func runDataMigrate(repoRoot string) error {
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskCloudService")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		log.Printf("[taskCloudService] dataMigrate dir not found: %v (skipping)", err)
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
	)`); err != nil {
		return fmt.Errorf("create data_migrate_log: %w", err)
	}

	for _, name := range files {
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, name).Scan(&exists); err == nil {
			log.Printf("[taskCloudService] dataMigrate: %s (skipped)", name)
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		// Wrap in a transaction so partial failures don't leave the DB inconsistent.
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
		log.Printf("[taskCloudService] dataMigrate: %s (applied)", name)
	}
	return nil
}
