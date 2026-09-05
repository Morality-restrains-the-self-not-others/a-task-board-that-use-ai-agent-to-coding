package main

import (
	"testing"

	dbload "dbload"
)

func setupProjectTestDB(t *testing.T) {
	t.Helper()
	testDSN, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-project", repoRoot(), "dataMigrate/taskProjectService",
		func(dsn string) error {
			if err := openDB(dsn); err != nil {
				return err
			}
			defer db.Close()
			return runDataMigrate(repoRoot())
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
}
