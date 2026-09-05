package main

import (
	"mysqlmeta"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// OPT-20260808-025: auth_oidc_client.managed_by 管理权属语义。
// 设计文档 §4.1-4.3.1（2026-08-08-task-chrome-plugin-oidc-extension-id-admin-design.md）。

// TestOidcMigration031ManagedByColumn — 031 migration 幂等性：
//   - 列存在（重跑 guarded_add_column_031 不报错）
//   - 行存在/行缺失（UPDATE 幂等）
func TestOidcMigration031ManagedByColumn(t *testing.T) {
	oidcTestSetup(t)
	defer db.Close()

	// 列存在 + 默认值 bootstrap
	var colType string
	if err := db.QueryRow(`
		SELECT DATA_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'auth_oidc_client' AND COLUMN_NAME = 'managed_by'`,
	).Scan(&colType); err != nil {
		t.Fatalf("managed_by 列缺失: %v", err)
	}
	if !strings.EqualFold(colType, "varchar") {
		t.Fatalf("managed_by 应为 varchar, got %s", colType)
	}

	// 无 chrome-extension 行时：直接重跑 031 SQL（列存在 + 行缺失场景）→ 不报错。
	// 与 runDataMigrateFromDir 相同预处理（strip DELIMITER + 自定义终止符），
	// 由 dbload 打开带 multiStatements 的 DSN。
	mig, err := os.ReadFile(filepath.Join(repoRoot(), "dataMigrate/taskAuth/031_oidc_client_managed_by.sql"))
	if err != nil {
		t.Fatalf("读取 031 migration: %v", err)
	}
	if _, err := db.Exec(mysqlmeta.StripMySQLClientMeta(string(mig))); err != nil {
		t.Fatalf("031 重跑（列存在/行缺失）报错: %v", err)
	}

	// 行存在场景：插入 chrome-extension 行后再重跑 → 置 admin 且不报错
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, created_at, updated_at)
		VALUES ('9001', 'chrome-extension', 'x', 'chrome-extension', '["http://127.0.0.1:9000/cb"]', NOW(), NOW())`); err != nil {
		t.Fatalf("插入 chrome-extension 行: %v", err)
	}
	if _, err := db.Exec(mysqlmeta.StripMySQLClientMeta(string(mig))); err != nil {
		t.Fatalf("031 重跑（行存在）报错: %v", err)
	}
	var managedBy string
	if err := db.QueryRow(`SELECT managed_by FROM auth_oidc_client WHERE client_id = 'chrome-extension'`).Scan(&managedBy); err != nil {
		t.Fatalf("查询 chrome-extension managed_by: %v", err)
	}
	if managedBy != "admin" {
		t.Fatalf("chrome-extension 应被 031 置为 admin, got %s", managedBy)
	}
}

// TestEnsureOidcClientAdminManagedRowNotOverwritten — managed_by='admin' 行：
// seed 仅 INSERT 缺失，不 UPDATE 覆盖管理员对 redirect_uris 的改库。
func TestEnsureOidcClientAdminManagedRowNotOverwritten(t *testing.T) {
	oidcTestSetup(t)
	defer db.Close()

	adminURIs := `["http://admin-changed.example.com/cb"]`
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, managed_by, created_at, updated_at)
		VALUES ('9002', 'admin-client', 'secret-hash', 'admin-client', ?, 'admin', NOW(), NOW())`, adminURIs); err != nil {
		t.Fatalf("插入 admin 托管行: %v", err)
	}

	if err := ensureOidcClient("admin-client", "other-secret", "admin-client", `["http://conf.example.com/cb"]`); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}

	row, err := loadOidcClient("admin-client")
	if err != nil || row == nil {
		t.Fatalf("loadOidcClient: err=%v row=%v", err, row)
	}
	if row.RedirectURIs != adminURIs {
		t.Fatalf("admin 托管行被 seed 覆盖: %s → %s", adminURIs, row.RedirectURIs)
	}
	if row.ManagedBy != "admin" {
		t.Fatalf("managed_by 应保持 admin, got %s", row.ManagedBy)
	}
}

// TestEnsureOidcClientBootstrapRowStillSelfHeals — managed_by='bootstrap'（默认）行：
// 维持原自愈 UPDATE（回归）。
func TestEnsureOidcClientBootstrapRowStillSelfHeals(t *testing.T) {
	oidcTestSetup(t)
	defer db.Close()

	confURIs := `["http://conf.example.com/cb"]`
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, created_at, updated_at)
		VALUES ('9003', 'bootstrap-client', 'secret-hash', 'bootstrap-client', '["http://old.example.com/cb"]', NOW(), NOW())`); err != nil {
		t.Fatalf("插入 bootstrap 托管行: %v", err)
	}

	if err := ensureOidcClient("bootstrap-client", "other-secret", "bootstrap-client", confURIs); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}

	row, err := loadOidcClient("bootstrap-client")
	if err != nil || row == nil {
		t.Fatalf("loadOidcClient: err=%v row=%v", err, row)
	}
	if row.RedirectURIs != confURIs {
		t.Fatalf("bootstrap 托管行应被自愈覆盖: %s → %s", row.RedirectURIs, confURIs)
	}
	if row.ManagedBy != "bootstrap" {
		t.Fatalf("managed_by 应保持 bootstrap, got %s", row.ManagedBy)
	}
}

// TestEnsureOidcClientInsertDefaultsBootstrap — INSERT 新行 managed_by 显式写
// 'bootstrap'（兼容列 default）。
func TestEnsureOidcClientInsertDefaultsBootstrap(t *testing.T) {
	oidcTestSetup(t)
	defer db.Close()

	if err := ensureOidcClient("brand-new-client", "new-secret", "brand-new-client", `["http://new.example.com/cb"]`); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}

	row, err := loadOidcClient("brand-new-client")
	if err != nil || row == nil {
		t.Fatalf("loadOidcClient: err=%v row=%v", err, row)
	}
	if row.ManagedBy != "bootstrap" {
		t.Fatalf("新行 managed_by 应默认 bootstrap, got %s", row.ManagedBy)
	}
}
