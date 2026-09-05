package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	dbload "dbload"
	_ "github.com/go-sql-driver/mysql"
)

// openLegacyTestDB creates a test MySQL database and returns the DSN for use with
// both raw SQL setup and openDB (schema migration). Both operations share the same
// database so that openDB's dataMigrate can transform the legacy schema.
func openLegacyTestDB(t *testing.T) (dsn string, cleanup func()) {
	t.Helper()
	dsn, dbCleanup, err := dbload.OpenTestMySQL("task-task", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		dbCleanup()
		t.Fatalf("open legacy: %v", err)
	}
	db.Close()
	return dsn, func() {
		dbCleanup()
	}
}

// TestMigrateCommentsLegacyUserIDRebuild ensures production DBs that still have
// comments.user_id NOT NULL can create comments after openDB migrates the schema.
func TestMigrateCommentsLegacyUserIDRebuild(t *testing.T) {
	legacyDSN, cleanup := openLegacyTestDB(t)
	defer cleanup()
	legacy, err := sql.Open("mysql", legacyDSN)
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE comments (
			id VARCHAR(255) PRIMARY KEY,
			task_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
				created_by_id VARCHAR(255) DEFAULT '',
				mentions_json TEXT,
				execution_mode VARCHAR(64) DEFAULT 'wait_previous',
				depends_on_comment_ids TEXT,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO comments(id, task_id, user_id, created_by_id, content, created_at)
		VALUES('cmt_old', 'task_legacy', 'u_old', '', 'legacy row', '2026-01-01 00:00:00');
	`)
	if err != nil {
		legacy.Close()
		t.Fatalf("seed legacy comments: %v", err)
	}
	legacy.Close()

	if err := openDB(legacyDSN); err != nil {
		t.Fatalf("openDB (migrate): %v", err)
	}
	// openDB 不再自动迁移（dataMigrate 由 9999 init / migrate CLI 负责），
	// 测试须显式应用 schema 以完成 comments → task_comments 转换。
	if err := runDataMigrate(repoRoot()); err != nil {
		t.Fatalf("runDataMigrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var hasUserID int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='task_comments' AND COLUMN_NAME='user_id'`,
	).Scan(&hasUserID); err != nil {
		t.Fatalf("info schema: %v", err)
	}
	if hasUserID != 0 {
		t.Fatalf("expected legacy user_id column removed, still present")
	}

	var createdBy string
	if err := db.QueryRow(`SELECT created_by_id FROM task_comments WHERE id='cmt_old'`).Scan(&createdBy); err != nil {
		t.Fatalf("select migrated row: %v", err)
	}
	if createdBy != "u_old" {
		t.Fatalf("created_by_id=%q want u_old", createdBy)
	}

	// INSERT path used by handleCreateComment must succeed without user_id.
	_, err = db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,created_at)
		 VALUES(?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"cmt_new", "task_legacy", "u_new", "hello",
		`[{"type":"installed_image","id":"img-1","name":"N"}]`,
	)
	if err != nil {
		t.Fatalf("insert after migrate: %v", err)
	}
}

func TestCreateCommentWorksAfterLegacyCommentsMigration(t *testing.T) {
	legacyDSN, cleanup := openLegacyTestDB(t)
	defer cleanup()
	legacy, err := sql.Open("mysql", legacyDSN)
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE comments (
			id VARCHAR(255) PRIMARY KEY, task_id VARCHAR(255) NOT NULL, created_by_id VARCHAR(255) DEFAULT '', user_id VARCHAR(255) NOT NULL,
				mentions_json TEXT,
				execution_mode VARCHAR(64) DEFAULT 'wait_previous',
				depends_on_comment_ids TEXT,
			content TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		legacy.Close()
		t.Fatalf("seed: %v", err)
	}
	legacy.Close()

	if err := openDB(legacyDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	// 同上：显式应用 schema 迁移（legacy comments → task_comments 全量建表）。
	if err := runDataMigrate(repoRoot()); err != nil {
		t.Fatalf("runDataMigrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-1": "trae0630"})
	taskID := createTestTaskForComments(t)

	body := `{"content":"把项目跑起来","mentions":[{"type":"installed_image","id":"img-1","name":"trae0630"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatalf("expected comment id, got %#v", resp)
	}
}
