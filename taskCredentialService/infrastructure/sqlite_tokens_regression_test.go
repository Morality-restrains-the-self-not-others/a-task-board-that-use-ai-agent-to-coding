package infrastructure

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// setupInfraTestDB 为 infrastructure 包测试创建独立 MySQL 测试库（taskapp 专用用户，
// 与根包 token_mysql_test_helper 同源，避免跨包依赖）。
func setupInfraTestDB(t *testing.T) *sql.DB {
	t.Helper()
	baseDSN := strings.TrimSpace(os.Getenv("TASKCRED_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}
	dbName := "test_infra_" + strings.ToLower(strings.NewReplacer(
		"/", "_", "-", "_", "(", "", ")", "", "*", "", "#", "",
	).Replace(t.Name()))
	if len(dbName) > 64 {
		dbName = dbName[:64]
	}
	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true&timeout=2s"
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open admin mysql: %v", err)
	}
	defer adminDB.Close()
	// MySQL 不可达时跳过而非失败（夜间全量巡检常遇环境未就绪即开跑），
	// 避免把基础设施故障误报为单测回归；与 sqlite_business_live_test.go 一致。
	if err := adminDB.Ping(); err != nil {
		t.Skipf("MySQL test DB unavailable: %v", err)
	}
	if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + dbName); err != nil {
		t.Fatalf("drop test db: %v", err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE " + dbName + " DEFAULT CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS " + dbName)
	})
	db, err := sql.Open("mysql", baseDSN+dbName+"?charset=utf8mb4&parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestScanTokenNullExpires 回归（OPT-20260809-024）：container_access_token_expires_at
// 为 NULL 时 scanToken 必须容忍（映射为空串），FindByAccessToken 不得报
// converting NULL to string。修复前该场景将令牌误判 TOKEN_NOT_FOUND →
// exchange-refresh 401 TOKEN_ACCESS_INVALID → 容器 TOKEN_BOOTSTRAP_FAILED。
func TestScanTokenNullExpires(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE credential_container_tokens (
			id VARCHAR(64) PRIMARY KEY, task_id VARCHAR(64) NOT NULL, company_id VARCHAR(64) NOT NULL DEFAULT '',
			workspace_id VARCHAR(64) NOT NULL DEFAULT '',
			comment_id VARCHAR(64) NOT NULL DEFAULT '',
			container_access_token VARCHAR(512) NOT NULL DEFAULT '',
			container_access_token_expires_at VARCHAR(64) NULL,
			container_refresh_token VARCHAR(512) NOT NULL DEFAULT '',
			server_url VARCHAR(1024) NOT NULL DEFAULT '',
			business_api_endpoint VARCHAR(1024) NOT NULL DEFAULT '',
			container_vscode_url VARCHAR(1024) NOT NULL DEFAULT '',
			authorization_id VARCHAR(64) NOT NULL DEFAULT '',
			image_id VARCHAR(64) NOT NULL DEFAULT '',
			instance_type VARCHAR(128) NOT NULL DEFAULT '',
			region_id VARCHAR(64) NOT NULL DEFAULT '',
			zone_id VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO credential_container_tokens (id, task_id, company_id, workspace_id,
			container_access_token, container_access_token_expires_at, container_refresh_token)
		VALUES (?, ?, ?, ?, ?, NULL, ?)
	`, "tok-null-exp", "task-null-exp", "tenant-1", "ws-1", "access-null-exp", "refresh-1"); err != nil {
		t.Fatalf("insert with NULL expires: %v", err)
	}

	repo := NewSQLiteTokenRepository(db)
	found, err := repo.FindByAccessToken("access-null-exp")
	if err != nil {
		t.Fatalf("FindByAccessToken with NULL expires: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.TaskID != "task-null-exp" {
		t.Errorf("expected task-null-exp, got %s", found.TaskID)
	}
	if found.ContainerAccessTokenExpiresAt != "" {
		t.Errorf("expected empty expires, got %q", found.ContainerAccessTokenExpiresAt)
	}
}

// TestSQLiteBusinessTableNames 回归（OPT-20260809-024）：业务表已改名
// （tasks→task_tasks、git_identities→task_git_identities），查询必须跟随，
// 否则 task-detail 502 TASK_SNAPSHOT_MISSING、repo 身份全部缺失。
func TestSQLiteBusinessTableNames(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE task_tasks (
			id VARCHAR(64) PRIMARY KEY, tenant_id VARCHAR(64) NOT NULL DEFAULT '',
			workspace_id VARCHAR(64) NOT NULL DEFAULT '', title VARCHAR(512) NOT NULL DEFAULT '',
			description TEXT, auto_run TINYINT DEFAULT 0, owner_id VARCHAR(64) NOT NULL DEFAULT '',
			installed_image_id VARCHAR(64) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		CREATE TABLE task_git_identities (
			id VARCHAR(64) PRIMARY KEY, user_id VARCHAR(64) NOT NULL DEFAULT '',
			git_user_name VARCHAR(255) NOT NULL DEFAULT '', git_user_email VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		CREATE TABLE task_repo_identities (
			id VARCHAR(64) PRIMARY KEY, task_id VARCHAR(64) NOT NULL, repo_url VARCHAR(1024) NOT NULL,
			git_identity_id VARCHAR(64) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if _, err := db.Exec("INSERT INTO task_tasks (id, tenant_id, workspace_id, title, auto_run, owner_id) VALUES (?,?,?,?,1,?)",
		"task-tn", "tenant-1", "ws-1", "hello world", "873438061961179136"); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	if _, err := db.Exec("INSERT INTO task_git_identities (id, user_id, git_user_name, git_user_email) VALUES (?,?,?,?)",
		"git-id-1", "42", "alice", "alice@example.com"); err != nil {
		t.Fatalf("insert identity: %v", err)
	}
	if _, err := db.Exec("INSERT INTO task_repo_identities (id, task_id, repo_url, git_identity_id) VALUES (?,?,?,?)",
		"ri-1", "task-tn", "https://github.com/ruandao/somanyad", "git-id-1"); err != nil {
		t.Fatalf("insert repo identity: %v", err)
	}

	repo := NewSQLiteBusinessRepository(db, nil)
	snap, err := repo.FetchTaskSnapshot("task-tn", "")
	if err != nil {
		t.Fatalf("FetchTaskSnapshot: %v", err)
	}
	if snap == nil || snap.Title != "hello world" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	if !snap.AutoRun {
		t.Error("expected auto_run=true")
	}
	ids, err := repo.FetchRepoIdentities("task-tn", "")
	if err != nil {
		t.Fatalf("FetchRepoIdentities: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(ids))
	}
	if ids[0].UserID != 42 || ids[0].UserName != "alice" {
		t.Errorf("identity details mismatch: %+v", ids[0])
	}
}
