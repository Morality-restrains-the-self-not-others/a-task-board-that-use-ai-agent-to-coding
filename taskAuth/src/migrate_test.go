package main

import (
	"testing"

	dbload "dbload"
)

func TestRunMigrationsCreatesAuthTables(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)

	if err := runDataMigrateFromDir(testDSN, repoRoot()); err != nil {
		t.Fatalf("runDataMigrateFromDir: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	for _, table := range []string{
		"auth_django_content_type",
		"auth_user",
		"auth_login_method",
		"auth_customtoken",
		"auth_super_admin",
		"data_migrate_log",
	} {
		var name string
		err := db.QueryRow(
			`SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	if err := runDataMigrateFromDir(testDSN, repoRoot()); err != nil {
		t.Fatalf("second runDataMigrateFromDir: %v", err)
	}
}
