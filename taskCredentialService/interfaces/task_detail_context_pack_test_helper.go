package interfaces

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
)

// isMySQLDiskFull reports whether err is a MySQL storage-full failure
// (e.g. "disk is full" while creating a table/tablespace on a full tmpfs).
// Such conditions are transient infra failures, not code regressions.
func isMySQLDiskFull(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	switch mysqlErr.Number {
	case 3675, // ER_DISK_FULL (InnoDB): "Create table/tablespace ... failed, as disk is full"
		1021: // ER_DISK_FULL (legacy): "Disk full ... waiting for someone to free some space"
		return true
	}
	return strings.Contains(strings.ToLower(mysqlErr.Message), "disk is full")
}

// setupMySQLTestDB creates a dedicated MySQL test database and returns a *sql.DB handle.
func setupMySQLTestDB(t *testing.T) *sql.DB {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("TASKCRED_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}

	dbName := "test_iface_" + strings.ToLower(strings.NewReplacer(
		"/", "_", "-", "_", "(", "", ")", "", "*", "", "#", "",
	).Replace(t.Name()))
	if len(dbName) > 64 {
		dbName = dbName[:64]
	}

	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true&timeout=2s"

	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open admin: %v", err)
	}
	defer adminDB.Close()

	// MySQL 不可达或磁盘满时跳过而非失败（夜间全量巡检常遇环境未就绪即开跑），
	// 避免把基础设施故障误报为单测回归。
	if err := adminDB.Ping(); err != nil {
		t.Skipf("MySQL test DB unavailable: %v", err)
	}

	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		if isMySQLDiskFull(err) {
			t.Skipf("MySQL test DB storage full, skipping: %v", err)
		}
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
