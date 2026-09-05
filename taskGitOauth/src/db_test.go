package main

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// setupMySQLTestDB creates a dedicated MySQL test database and returns a *sql.DB handle.
func setupMySQLTestDB(t *testing.T) *sql.DB {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("GITOAUTH_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}

	dbName := "test_gitoa_" + strings.ToLower(strings.NewReplacer(
		"/", "_", "-", "_", "(", "", ")", "", "*", "", "#", "",
	).Replace(t.Name()))
	if len(dbName) > 64 {
		dbName = dbName[:64]
	}

	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"

	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open admin: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create test db %s: %v", dbName, err)
	}

	dsn := baseDSN + dbName + "?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		adminDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`")
		t.Fatalf("open test db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		cleanDB, _ := sql.Open("mysql", adminDSN)
		if cleanDB != nil {
			cleanDB.Exec("DROP DATABASE IF EXISTS `" + dbName + "`")
			cleanDB.Close()
		}
	})

	return db
}
