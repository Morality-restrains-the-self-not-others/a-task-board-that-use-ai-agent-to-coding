package dbload

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const schemaRevTable = "_dbload_schema_rev"

// HashSQLDir returns a stable sha256 of all *.sql files in dir (name + contents).
// Used as the test-schema template revision so adding a migration invalidates the clone source.
func HashSQLDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("HashSQLDir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "", fmt.Errorf("HashSQLDir read %s: %w", name, err)
		}
		_, _ = fmt.Fprintf(h, "%s\n", name)
		_, _ = h.Write(body)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func templateDBName(namePrefix, rev string) string {
	clean := strings.ReplaceAll(namePrefix, "-", "_")
	short := rev
	if len(short) > 12 {
		short = short[:12]
	}
	name := "tpl_" + clean + "_" + short
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func configurePool(db *sql.DB) {
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
}

func ensureAdminDSN(dsn string) string {
	dsn = extractAdminDSN(dsn)
	if !strings.Contains(dsn, "charset=") {
		dsn += "&charset=utf8mb4"
		dsn = strings.ReplaceAll(dsn, "?&", "?")
	}
	return dsn
}

// OpenTestMySQLClonedFromDir hashes migrateRelDir under monorepoRoot and clones
// a per-revision template instead of replaying every dataMigrate file per test.
func OpenTestMySQLClonedFromDir(serviceKey, monorepoRoot, migrateRelDir string, migrate func(dsn string) error) (string, func(), error) {
	rev, err := HashSQLDir(filepath.Join(monorepoRoot, migrateRelDir))
	if err != nil {
		return "", nil, err
	}
	return OpenTestMySQLCloned(serviceKey, monorepoRoot, rev, migrate)
}

// OpenTestMySQLCloned creates a unique test database by cloning a migrated template.
// migrate runs only when the template for schemaRev is missing or stale.
func OpenTestMySQLCloned(serviceKey, monorepoRoot, schemaRev string, migrate func(dsn string) error) (string, func(), error) {
	base, err := ResolveMySQLDSN(serviceKey, monorepoRoot)
	if err != nil {
		return "", nil, fmt.Errorf("OpenTestMySQLCloned: resolve DSN: %w", err)
	}
	prefix := strings.ReplaceAll(serviceKey, "-", "_")
	return PrepareClonedTestDB(base, prefix, schemaRev, migrate)
}

// PrepareClonedTestDB clones from a GET_LOCK-guarded template database.
// namePrefix distinguishes services (and unit-test templates) in template DB names.
func PrepareClonedTestDB(baseDSN, namePrefix, schemaRev string, migrate func(dsn string) error) (dsn string, cleanup func(), err error) {
	if migrate == nil {
		return "", nil, errors.New("PrepareClonedTestDB: migrate is required")
	}
	if strings.TrimSpace(schemaRev) == "" {
		return "", nil, errors.New("PrepareClonedTestDB: schemaRev is required")
	}
	if strings.TrimSpace(namePrefix) == "" {
		return "", nil, errors.New("PrepareClonedTestDB: namePrefix is required")
	}

	adminDSN := ensureAdminDSN(baseDSN)
	admin, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return "", nil, fmt.Errorf("PrepareClonedTestDB: open admin: %w", err)
	}
	defer admin.Close()
	configurePool(admin)

	lockName := "dbload_tpl_" + strings.ReplaceAll(namePrefix, "-", "_")
	if len(lockName) > 64 {
		lockName = lockName[:64]
	}
	if err := withMySQLLock(admin, lockName, func() error {
		tpl := templateDBName(namePrefix, schemaRev)
		return ensureTemplate(admin, baseDSN, tpl, schemaRev, migrate)
	}); err != nil {
		return "", nil, err
	}

	tpl := templateDBName(namePrefix, schemaRev)
	testDBName := strings.ReplaceAll(namePrefix, "-", "_") + "_test_" + randomSuffix(8)
	if len(testDBName) > 64 {
		testDBName = testDBName[:64]
	}
	if err := cloneMySQLSchema(admin, tpl, testDBName); err != nil {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS `" + testDBName + "`")
		return "", nil, err
	}
	return replaceDBName(baseDSN, testDBName), makeTestDBCleanup(baseDSN, testDBName), nil
}

func withMySQLLock(db *sql.DB, name string, fn func() error) error {
	var got sql.NullInt64
	if err := db.QueryRow("SELECT GET_LOCK(?, 180)", name).Scan(&got); err != nil {
		return fmt.Errorf("GET_LOCK %s: %w", name, err)
	}
	if !got.Valid || got.Int64 != 1 {
		return fmt.Errorf("GET_LOCK %s not acquired", name)
	}
	defer db.Exec("SELECT RELEASE_LOCK(?)", name)
	return fn()
}

func ensureTemplate(admin *sql.DB, baseDSN, tpl, rev string, migrate func(string) error) error {
	var n int
	if err := admin.QueryRow(
		"SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME=?", tpl,
	).Scan(&n); err != nil {
		return fmt.Errorf("lookup template %s: %w", tpl, err)
	}
	if n == 1 {
		var existing string
		err := admin.QueryRow("SELECT rev FROM `" + tpl + "`.`" + schemaRevTable + "` LIMIT 1").Scan(&existing)
		if err == nil && existing == rev {
			return nil
		}
		if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + tpl + "`"); err != nil {
			return fmt.Errorf("drop stale template %s: %w", tpl, err)
		}
	}
	if _, err := admin.Exec("CREATE DATABASE `" + tpl + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return fmt.Errorf("create template %s: %w", tpl, err)
	}
	tplDSN := replaceDBName(baseDSN, tpl)
	log.Printf("[dbload] migrating schema template %s", tpl)
	if err := migrate(tplDSN); err != nil {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS `" + tpl + "`")
		return fmt.Errorf("migrate template %s: %w", tpl, err)
	}
	createRev := "CREATE TABLE `" + tpl + "`.`" + schemaRevTable + "` (" +
		"rev VARCHAR(64) NOT NULL PRIMARY KEY" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"
	if _, err := admin.Exec(createRev); err != nil {
		return fmt.Errorf("create rev table: %w", err)
	}
	if _, err := admin.Exec("INSERT INTO `"+tpl+"`.`"+schemaRevTable+"` (rev) VALUES (?)", rev); err != nil {
		return fmt.Errorf("insert rev: %w", err)
	}
	return nil
}

func listSchemaObjects(conn *sql.Conn, ctx context.Context, schema, tableType string) ([]string, error) {
	rows, err := conn.QueryContext(ctx,
		`SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_TYPE=? ORDER BY TABLE_NAME`,
		schema, tableType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// cloneMySQLSchema 把 src 模板克隆为 dst 测试库。
//
// OPT-20260824-053：整个克隆流程必须钉在单一 *sql.Conn 上执行——
// `SET SESSION FOREIGN_KEY_CHECKS=0` 与 `USE dst` 都是会话级状态，若 pool 把
// CREATE/INSERT 分到其它连接，含外键的表按字母序复制时会偶发外键违例
// （taskAuth 全量测试「首轮 FAIL 后全绿」的间歇失败根因）。钉单连接同时
// 避免 `USE dst` 的默认库污染泄漏回 pool 影响其它测试未限定库名的 SQL。
// 收尾把默认库重置为 information_schema（恒存在），连接回池后不再指向
// 已删除/他用的测试库。
func cloneMySQLSchema(admin *sql.DB, src, dst string) error {
	ctx := context.Background()
	conn, err := admin.Conn(ctx)
	if err != nil {
		return fmt.Errorf("clone acquire conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+dst+"`"); err != nil {
		return fmt.Errorf("drop dest %s: %w", dst, err)
	}
	if _, err := conn.ExecContext(ctx, "CREATE DATABASE `"+dst+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return fmt.Errorf("create dest %s: %w", dst, err)
	}
	clonedOK := false
	defer func() {
		if clonedOK {
			return
		}
		_, _ = conn.ExecContext(ctx, "USE `information_schema`")
		if _, dropErr := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+dst+"`"); dropErr != nil {
			log.Printf("[dbload] WARN clone rollback: drop dest %s failed: %v", dst, dropErr)
		}
	}()
	if _, err := conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}
	// SHOW CREATE TABLE 输出的 DDL 用未限定表名，必须设默认库为 dst。
	if _, err := conn.ExecContext(ctx, "USE `"+dst+"`"); err != nil {
		return fmt.Errorf("use dest %s: %w", dst, err)
	}
	// 收尾重置会话状态：FK 回 1、默认库指回恒存在的 information_schema，
	// 使连接回池后不残留 dst 会话污染（LIFO：先重置 USE 再重置 FK）。
	defer conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=1")
	defer conn.ExecContext(ctx, "USE `information_schema`")

	var srcExists int
	if err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME=?", src,
	).Scan(&srcExists); err != nil {
		return fmt.Errorf("lookup source schema %s: %w", src, err)
	}
	if srcExists == 0 {
		return fmt.Errorf("clone source schema %s does not exist", src)
	}

	tables, err := listSchemaObjects(conn, ctx, src, "BASE TABLE")
	if err != nil {
		return err
	}
	for _, table := range tables {
		var name, create string
		q := "SHOW CREATE TABLE `" + src + "`.`" + table + "`"
		if err := conn.QueryRowContext(ctx, q).Scan(&name, &create); err != nil {
			return fmt.Errorf("SHOW CREATE TABLE %s.%s: %w", src, table, err)
		}
		if _, err := conn.ExecContext(ctx, create); err != nil {
			return fmt.Errorf("create table %s.%s: %w", dst, table, err)
		}
		copySQL, err := copyTableInsertSQL(conn, ctx, src, dst, table)
		if err != nil {
			return err
		}
		if copySQL == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, copySQL); err != nil {
			return fmt.Errorf("copy table %s: %w", table, err)
		}
	}
	views, err := listSchemaObjects(conn, ctx, src, "VIEW")
	if err != nil {
		return err
	}
	for _, view := range views {
		var name, create, cs, coll string
		q := "SHOW CREATE VIEW `" + src + "`.`" + view + "`"
		if err := conn.QueryRowContext(ctx, q).Scan(&name, &create, &cs, &coll); err != nil {
			return fmt.Errorf("SHOW CREATE VIEW %s.%s: %w", src, view, err)
		}
		if _, err := conn.ExecContext(ctx, create); err != nil {
			return fmt.Errorf("create view %s.%s: %w", dst, view, err)
		}
	}
	clonedOK = true
	return nil
}

func quoteMySQLIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// listCopyableColumns omits VIRTUAL/STORED GENERATED columns. INSERT SELECT *
// specifies those columns and MySQL rejects them (Error 3105), which broke
// cloning task-auth after auth_user_role.company_id_key (048).
func listCopyableColumns(conn *sql.Conn, ctx context.Context, schema, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT COLUMN_NAME FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		  AND EXTRA NOT LIKE '%VIRTUAL GENERATED%'
		  AND EXTRA NOT LIKE '%STORED GENERATED%'
		ORDER BY ORDINAL_POSITION`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("list copyable columns %s.%s: %w", schema, table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func copyTableInsertSQL(conn *sql.Conn, ctx context.Context, src, dst, table string) (string, error) {
	cols, err := listCopyableColumns(conn, ctx, src, table)
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", nil
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteMySQLIdent(c)
	}
	list := strings.Join(quoted, ", ")
	return "INSERT INTO " + quoteMySQLIdent(dst) + "." + quoteMySQLIdent(table) +
		" (" + list + ") SELECT " + list + " FROM " + quoteMySQLIdent(src) + "." + quoteMySQLIdent(table), nil
}
