package dbload

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashSQLDirStableAndSensitiveToContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "002_b.sql"), []byte("CREATE TABLE b (id INT);"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "001_a.sql"), []byte("CREATE TABLE a (id INT);"), 0o644); err != nil {
		t.Fatal(err)
	}
	h1, err := HashSQLDir(dir)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	h2, err := HashSQLDir(dir)
	if err != nil {
		t.Fatalf("hash 2: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hash not stable: %s vs %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(h1))
	}
	if err := os.WriteFile(filepath.Join(dir, "003_c.sql"), []byte("CREATE TABLE c (id INT);"), 0o644); err != nil {
		t.Fatal(err)
	}
	h3, err := HashSQLDir(dir)
	if err != nil {
		t.Fatalf("hash 3: %v", err)
	}
	if h3 == h1 {
		t.Fatal("expected hash to change when a SQL file is added")
	}
}

func TestTemplateDBNameFitsMySQLLimit(t *testing.T) {
	name := templateDBName("task-cloud-service-with-a-very-long-key", strings.Repeat("abcdef", 20))
	if len(name) > 64 {
		t.Fatalf("template name %q length %d exceeds MySQL 64", name, len(name))
	}
	if !strings.HasPrefix(name, "tpl_") {
		t.Fatalf("expected tpl_ prefix, got %q", name)
	}
}

func TestCloneMySQLSchemaDropsDestWhenSourceMissing(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	if !MySQLAvailable(adminDSN) {
		t.Skip("MySQL not reachable")
	}

	admin, err := sql.Open("mysql", ensureAdminDSN(adminDSN))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	configurePool(admin)

	dst := "dbload_clone_fail_test_" + randomSuffix(8)
	src := "dbload_clone_missing_src_" + randomSuffix(8)
	err = cloneMySQLSchema(admin, src, dst)
	if err == nil {
		t.Fatal("expected clone to fail when source schema is missing")
	}

	var n int
	if qerr := admin.QueryRow(
		"SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME=?", dst,
	).Scan(&n); qerr != nil {
		t.Fatalf("lookup dest: %v", qerr)
	}
	if n != 0 {
		t.Fatalf("clone failure left dest database %s (count=%d)", dst, n)
	}
}

func TestPrepareClonedTestDBIsolatesMutations(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	if !MySQLAvailable(adminDSN) {
		t.Skip("MySQL not reachable")
	}

	migrate := func(dsn string) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		configurePool(db)
		if _, err := db.Exec(`CREATE TABLE widget (
			id INT NOT NULL PRIMARY KEY,
			n INT NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO widget (id, n) VALUES (1, 10)`)
		return err
	}

	rev := "clone-ut-widget-v1"
	dsn1, cleanup1, err := PrepareClonedTestDB(adminDSN, "dbload_clone_ut", rev, migrate)
	if err != nil {
		t.Fatalf("clone 1: %v", err)
	}
	defer cleanup1()
	dsn2, cleanup2, err := PrepareClonedTestDB(adminDSN, "dbload_clone_ut", rev, migrate)
	if err != nil {
		t.Fatalf("clone 2: %v", err)
	}
	defer cleanup2()

	db1, err := sql.Open("mysql", dsn1)
	if err != nil {
		t.Fatalf("open clone1: %v", err)
	}
	defer db1.Close()
	configurePool(db1)
	if _, err := db1.Exec("UPDATE widget SET n=99 WHERE id=1"); err != nil {
		t.Fatalf("mutate clone1: %v", err)
	}

	db2, err := sql.Open("mysql", dsn2)
	if err != nil {
		t.Fatalf("open clone2: %v", err)
	}
	defer db2.Close()
	configurePool(db2)
	var n int
	if err := db2.QueryRow("SELECT n FROM widget WHERE id=1").Scan(&n); err != nil {
		t.Fatalf("read clone2: %v", err)
	}
	if n != 10 {
		t.Fatalf("clone2 saw clone1 mutation: n=%d want 10", n)
	}
}

// OPT-20260824-053：含外键的 schema 克隆必须成功。克隆按字母序复制表，
// child 先于 parent 时依赖 `SET SESSION FOREIGN_KEY_CHECKS=0` 钉在同一连接；
// 若 pool 把 CREATE/INSERT 分到其它连接（FK 检查默认开），会偶发 errno 150。
func TestPrepareClonedTestDBClonesForeignKeys(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	if !MySQLAvailable(adminDSN) {
		t.Skip("MySQL not reachable")
	}

	migrate := func(dsn string) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		configurePool(db)
		// 两个表都建：克隆按字母序 child 先于 parent。
		if _, err := db.Exec(`CREATE TABLE z_parent (
			id INT NOT NULL PRIMARY KEY
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
			return err
		}
		if _, err := db.Exec(`CREATE TABLE a_child (
			id INT NOT NULL PRIMARY KEY,
			parent_id INT NULL,
			CONSTRAINT fk_a_child_parent FOREIGN KEY (parent_id) REFERENCES z_parent(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO z_parent (id) VALUES (1)`); err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO a_child (id, parent_id) VALUES (1, 1)`)
		return err
	}

	dsn, cleanup, err := PrepareClonedTestDB(adminDSN, "dbload_clone_fk", "clone-ut-fk-v1", migrate)
	if err != nil {
		t.Fatalf("clone with FK: %v", err)
	}
	defer cleanup()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open clone: %v", err)
	}
	defer db.Close()
	configurePool(db)
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM a_child WHERE id=1 AND parent_id=1").Scan(&n); err != nil {
		t.Fatalf("read clone FK: %v", err)
	}
	if n != 1 {
		t.Fatalf("cloned FK row count=%d want 1", n)
	}
	// FK 约束应被完整克隆（证明 CREATE 用的 DDL 未因跳过校验而丢失约束）
	var fk int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA=DATABASE() AND CONSTRAINT_NAME='fk_a_child_parent'",
	).Scan(&fk); err != nil {
		t.Fatalf("read FK constraint: %v", err)
	}
	if fk != 1 {
		t.Fatalf("cloned FK constraint count=%d want 1", fk)
	}
}

func TestPrepareClonedTestDBClonesGeneratedColumns(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	if !MySQLAvailable(adminDSN) {
		t.Skip("MySQL not reachable")
	}

	migrate := func(dsn string) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		configurePool(db)
		if _, err := db.Exec(`CREATE TABLE auth_role_gen (
			id BIGINT NOT NULL PRIMARY KEY,
			company_id VARCHAR(64) NULL,
			company_id_key VARCHAR(64)
				GENERATED ALWAYS AS (COALESCE(company_id, '')) STORED,
			UNIQUE KEY uk_company_key (company_id_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
			return err
		}
		_, err = db.Exec(`INSERT INTO auth_role_gen (id, company_id) VALUES (1, NULL), (2, 'acme')`)
		return err
	}

	dsn, cleanup, err := PrepareClonedTestDB(adminDSN, "dbload_clone_gen", "clone-ut-gen-v1", migrate)
	if err != nil {
		t.Fatalf("clone with generated column: %v", err)
	}
	defer cleanup()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open clone: %v", err)
	}
	defer db.Close()
	configurePool(db)
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM auth_role_gen").Scan(&n); err != nil {
		t.Fatalf("read clone generated: %v", err)
	}
	if n != 2 {
		t.Fatalf("cloned generated-column row count=%d want 2", n)
	}
	var key string
	if err := db.QueryRow("SELECT company_id_key FROM auth_role_gen WHERE id=1").Scan(&key); err != nil {
		t.Fatalf("read generated value: %v", err)
	}
	if key != "" {
		t.Fatalf("NULL company_id should generate '', got %q", key)
	}
}

func TestPrepareClonedTestDBRefreshesStaleTemplate(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TASKBILL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	if !MySQLAvailable(adminDSN) {
		t.Skip("MySQL not reachable")
	}

	migrateV1 := func(dsn string) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		configurePool(db)
		_, err = db.Exec(`CREATE TABLE only_v1 (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
		return err
	}
	migrateV2 := func(dsn string) error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		configurePool(db)
		_, err = db.Exec(`CREATE TABLE only_v2 (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
		return err
	}

	dsn1, cleanup1, err := PrepareClonedTestDB(adminDSN, "dbload_clone_rev", "rev-a", migrateV1)
	if err != nil {
		t.Fatalf("v1: %v", err)
	}
	defer cleanup1()
	dsn2, cleanup2, err := PrepareClonedTestDB(adminDSN, "dbload_clone_rev", "rev-b", migrateV2)
	if err != nil {
		t.Fatalf("v2: %v", err)
	}
	defer cleanup2()

	db1, err := sql.Open("mysql", dsn1)
	if err != nil {
		t.Fatal(err)
	}
	defer db1.Close()
	configurePool(db1)
	var n1 int
	if err := db1.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='only_v1'").Scan(&n1); err != nil {
		t.Fatal(err)
	}
	if n1 != 1 {
		t.Fatalf("v1 clone missing only_v1")
	}

	db2, err := sql.Open("mysql", dsn2)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	configurePool(db2)
	var n2 int
	if err := db2.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='only_v2'").Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n2 != 1 {
		t.Fatalf("v2 clone missing only_v2 after rev change")
	}
}
