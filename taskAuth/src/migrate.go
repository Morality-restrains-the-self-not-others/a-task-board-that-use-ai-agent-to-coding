package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"mysqlmeta"
)

func runDataMigrateFromDir(dsn string, repoRoot string) error {
	migDir := filepath.Join(repoRoot, "dataMigrate", "taskAuth")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
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
		return fmt.Errorf("no migrations in %s", migDir)
	}

	// Ensure multiStatements so multi-statement SQL files (e.g. stored procedure +
	// CALL + DROP) execute atomically. Without this, only the first statement runs
	// and the migration silently fails.
	if !strings.Contains(dsn, "multiStatements=true") {
		if strings.Contains(dsn, "?") {
			dsn += "&multiStatements=true"
		} else {
			dsn += "?multiStatements=true"
		}
	}
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS data_migrate_log (
		step_key VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64) NOT NULL DEFAULT ''
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("data_migrate_log: %w", err)
	}

	// Backfill from legacy taskauth_schema_migrations into data_migrate_log
	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM data_migrate_log`).Scan(&count); err == nil && count == 0 {
		rows, err := conn.Query(`SELECT name, applied_at FROM taskauth_schema_migrations ORDER BY name`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var name, appliedAt string
				if rows.Scan(&name, &appliedAt) == nil {
					_, _ = conn.Exec(`INSERT IGNORE INTO data_migrate_log (step_key, applied_at) VALUES (?, ?)`, name, appliedAt)
				}
			}
		}
	}

	for _, name := range files {
		var exists int
		if err := conn.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, name).Scan(&exists); err == nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return err
		}
		sqlText, err := renderBootstrapAdminSQL(string(raw), repoRoot)
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := conn.Exec(mysqlmeta.StripMySQLClientMeta(sqlText)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		_, err = conn.Exec(`INSERT INTO data_migrate_log (step_key, applied_at) VALUES (?, NOW())`, name)
		if err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}
