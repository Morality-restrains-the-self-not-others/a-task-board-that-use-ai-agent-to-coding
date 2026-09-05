// Package main — Go-based dataMigrate infrastructure for taskAuth.
// GoDataMigrateSteps are executed by RunGoDataMigrate() after openDB().
// Each step is tracked via data_migrate_log for idempotent execution.
package main

import "log"

type goMigrateStep struct {
	Key      string
	Apply    func() error
	Always   bool          // true: run every migrate (table-level idempotent seed)
	Checksum func() string // optional audit hash written to data_migrate_log.checksum
}

// GoDataMigrateSteps lists all Go-based bootstrap/seed steps.
// DDL is exclusively managed via dataMigrate/taskAuth/*.sql files — no Go DDL here.
// Steps run with global db available (called after openDB() in main.go).
var GoDataMigrateSteps = []goMigrateStep{
	{
		Key:      "010_oidc_bootstrap_clients",
		Apply:    seedOidcBootstrapClients,
		Always:   true, // conf 可追加 client；禁止因 step_key 已存在而跳过
		Checksum: oidcBootstrapClientChecksum,
	},
}

// RunGoDataMigrate executes all registered Go dataMigrate steps after openDB().
// SQL-style steps skip when data_migrate_log already has step_key.
// Always steps re-run every migrate (Apply must be table-level idempotent).
func RunGoDataMigrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS data_migrate_log (
		step_key VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64) NOT NULL DEFAULT ''
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}

	// Backfill from legacy taskauth_schema_migrations into data_migrate_log
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM data_migrate_log`).Scan(&count); err == nil && count == 0 {
		rows, err := db.Query(`SELECT name, applied_at FROM taskauth_schema_migrations ORDER BY name`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var name, appliedAt string
				if rows.Scan(&name, &appliedAt) == nil {
					_, _ = db.Exec(`INSERT IGNORE INTO data_migrate_log (step_key, applied_at) VALUES (?, ?)`, name, appliedAt)
				}
			}
		}
	}

	for _, step := range GoDataMigrateSteps {
		if !step.Always {
			var exists int
			if err := db.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, step.Key).Scan(&exists); err == nil {
				log.Printf("[taskAuth] dataMigrate: %s (skipped)", step.Key)
				continue
			}
		}
		if err := step.Apply(); err != nil {
			return err
		}
		checksum := ""
		if step.Checksum != nil {
			checksum = step.Checksum()
		}
		if _, err := db.Exec(`
			INSERT INTO data_migrate_log (step_key, applied_at, checksum) VALUES (?, NOW(), ?)
			ON DUPLICATE KEY UPDATE applied_at = NOW(), checksum = ?`,
			step.Key, checksum, checksum); err != nil {
			return err
		}
		if step.Always {
			log.Printf("[taskAuth] dataMigrate: %s (applied always checksum=%s)", step.Key, checksum)
		} else {
			log.Printf("[taskAuth] dataMigrate: %s (applied)", step.Key)
		}
	}
	return nil
}
