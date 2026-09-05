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
	// Schema is fully managed via dataMigrate/taskTenantService/*.sql files.
	// Applied only via 9999 init / `migrate` CLI — not on business openDB.
	return nil
}

func repoRoot() string {
	exe, _ := os.Executable()
	candidate := filepath.Dir(exe)
	for {
		if _, err := os.Stat(filepath.Join(candidate, "taskTenantService")); err == nil {
			return candidate
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
		candidate = parent
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; d != "/" && d != "."; d = filepath.Dir(d) {
			if _, err := os.Stat(filepath.Join(d, "taskTenantService")); err == nil {
				return d
			}
		}
	}
	return "."
}

// runDataMigrate executes all .sql files from dataMigrate/taskTenantService/
func runDataMigrate(repoRoot string) error {
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskTenantService")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		log.Printf("[taskTenantService] dataMigrate dir not found: %v (skipping)", err)
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
			log.Printf("[taskTenantService] dataMigrate: %s (skipped)", name)
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
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
		log.Printf("[taskTenantService] dataMigrate: %s (applied)", name)
	}
	return nil
}
