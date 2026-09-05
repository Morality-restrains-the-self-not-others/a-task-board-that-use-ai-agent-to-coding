package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	dbload "dbload"

	_ "github.com/go-sql-driver/mysql"
	"mysqlmeta"
)

var (
	budgetDB   *sql.DB
	budgetDBMu sync.RWMutex
)

func resolveBudgetDBDSN(repoRoot string) string {
	if v := os.Getenv("TASK_BUDGET_MYSQL_DSN"); v != "" {
		return v
	}
	dsn, err := dbload.ResolveMySQLDSN("task-budget", repoRoot)
	if err == nil && dsn != "" {
		return dsn
	}
	return ""
}

func openBudgetDB(dsn, repoRoot string) error {
	budgetDBMu.Lock()
	defer budgetDBMu.Unlock()
	if budgetDB != nil {
		_ = budgetDB.Close()
		budgetDB = nil
	}
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open budget db: %w", err)
	}
	conn.SetMaxOpenConns(8)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(5 * time.Minute)
	conn.SetConnMaxIdleTime(2 * time.Minute)
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("ping budget db: %w", err)
	}
	// Budget schema/seed: dataMigrate/taskBudget via 9999 init / migrate CLI only.
	budgetDB = conn
	log.Printf("[taskCloudService] budget db opened (mysql)")
	return nil
}

// runBudgetDataMigrate executes .sql files from dataMigrate/taskBudget/ against the budget database.
func runBudgetDataMigrate(conn *sql.DB, repoRoot string) error {
	if repoRoot == "" {
		return nil // test mode — skip dataMigrate
	}
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskBudget")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		log.Printf("[taskCloudService] budget dataMigrate dir not found: %v (skipping)", err)
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

	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS data_migrate_log (
		step_key VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64) NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("budget data_migrate_log: %w", err)
	}

	for _, name := range files {
		var exists int
		if err := conn.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, name).Scan(&exists); err == nil {
			log.Printf("[taskCloudService] budget dataMigrate: %s (skipped)", name)
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return fmt.Errorf("budget read %s: %w", name, err)
		}
		// Wrap in a transaction so partial failures don't leave the DB inconsistent.
		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("budget begin tx for %s: %w", name, err)
		}
		if _, err := tx.Exec(mysqlmeta.StripMySQLClientMeta(string(raw))); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("budget migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO data_migrate_log (step_key, applied_at) VALUES (?, NOW())`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("budget record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("budget commit %s: %w", name, err)
		}
		log.Printf("[taskCloudService] budget dataMigrate: %s (applied)", name)
	}
	return nil
}

func closeBudgetDB() {
	budgetDBMu.Lock()
	defer budgetDBMu.Unlock()
	if budgetDB != nil {
		_ = budgetDB.Close()
		budgetDB = nil
	}
}

func getBudgetDB() *sql.DB {
	budgetDBMu.RLock()
	defer budgetDBMu.RUnlock()
	return budgetDB
}

// openBudgetDBForTest opens an isolated DB connection for unit tests.
// Passes the real repoRoot so dataMigrate SQL files (CREATE TABLE, etc.)
// are applied to the empty test database created by OpenTestMySQL.
func openBudgetDBForTest(dsn string) error {
	if err := openBudgetDB(dsn, repoRoot()); err != nil {
		return err
	}
	return runBudgetDataMigrate(budgetDB, repoRoot())
}

// ensureBudgetLedgerSchema verifies that dataMigrate ran successfully on the budget
// database by checking for the existence of core budget tables. DDL is the single
// source of truth in dataMigrate/taskBudget/001_schema.sql.
func ensureBudgetLedgerSchema(conn *sql.DB) error {
	coreTables := []string{
		"cloud_workspace_model_budget_default",
		"cloud_task_model_budget",
		"cloud_tenant_budget_permission",
	}
	for _, table := range coreTables {
		var count int
		if err := conn.QueryRow(
			`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`,
			table,
		).Scan(&count); err != nil {
			return fmt.Errorf("ensureBudgetLedgerSchema: cannot check table %s: %w", table, err)
		}
		if count == 0 {
			return fmt.Errorf(
				"budget table %q is missing — dataMigrate may not have run. "+
					"Run: cd dataMigrate/taskBudget && mysql < 001_schema.sql",
				table,
			)
		}
	}
	return nil
}
