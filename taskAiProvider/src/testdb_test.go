package main

// 共享测试 DB helper（OPT-20260806-063：合并 handlers_test.go testApp 与
// sso_exchange_db_test.go openSSOAppWithTestDB 两套独立的测试库初始化/迁移应用逻辑，
// 避免长期维护漂移 — 迁移目录变更只改这一处）。

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	dbload "dbload"
	"taskAiProvider/infrastructure"
)

// testApp 委托共享 helper（OPT-20260806-063：统一 openTestApp，
// 测试库初始化/迁移应用逻辑见 testdb_test.go，避免双份实现漂移）。
func testApp(t *testing.T) *App {
	t.Helper()
	return openTestApp(t)
}

// openTestApp 构建基于真实 LoadConfig（真源 SSOJwtSecret 解析，清空 env 覆盖）
// + 独立测试 MySQL 库的 App。SSO 成功路径（ExchangeBridge 建号/兑换）必须在此
// 基础上验证，避免 testAppMinimal（硬编码密钥 + DB=nil）掩盖配置链路回归。
func openTestApp(t *testing.T) *App {
	t.Helper()
	root, err := infrastructure.FindMonorepoRoot()
	if err != nil {
		t.Skipf("monorepo root: %v", err)
	}
	// 清空 env 覆盖，让 LoadConfig 回退到 conf/core/sso/config.yaml 真源
	// （LoadConfig 以 TrimSpace(Getenv) != "" 判定，置空等价于未设置）。
	t.Setenv("TASK2APP_SSO_JWT_SECRET", "")
	t.Setenv("DJANGO_SECRET_KEY", "")
	cfg, err := infrastructure.LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	dsn, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"ai-provider", root, "dataMigrate/taskAiProvider",
		func(migrateDSN string) error {
			return applyAIMigrationsErr(migrateDSN, root)
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	db, err := infrastructure.OpenDB(dsn)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	cfg.VendorDocsDir = t.TempDir()
	cfg.VendorDocsBackend = "local"
	app := NewApp(cfg, db)
	app.AutoRunExtractor = noopAutoRunExtractor{}
	app.SkillsExtractor = noopImageSkillsExtractor{}
	app.ExtractSync = true
	// 单测默认跳过真实 taskAuth SMS gate（由专用用例覆盖未验证路径）
	prev := vendorContactVerifiedFn
	vendorContactVerifiedFn = func(_ *App, _ context.Context, _ string) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { vendorContactVerifiedFn = prev })
	return app
}

// applyAIMigrations 对测试库应用 dataMigrate/taskAiProvider/*.sql
// （与 taskAuth runDataMigrateFromDir 同机制：排序执行 + multiStatements）。
func applyAIMigrations(t *testing.T, dsn string, root string) {
	t.Helper()
	if err := applyAIMigrationsErr(dsn, root); err != nil {
		t.Fatal(err)
	}
}

func applyAIMigrationsErr(dsn, root string) error {
	migDir := filepath.Join(root, "dataMigrate", "taskAiProvider")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migrations in %s", migDir)
	}
	if !strings.Contains(dsn, "multiStatements=true") {
		if strings.Contains(dsn, "?") {
			dsn += "&multiStatements=true"
		} else {
			dsn += "?multiStatements=true"
		}
	}
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(5 * time.Minute)
	conn.SetConnMaxIdleTime(2 * time.Minute)
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			return err
		}
		if _, err := conn.Exec(string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}
