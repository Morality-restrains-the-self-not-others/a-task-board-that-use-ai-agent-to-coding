package main

import (
	"context"
	"testing"
)

// TestConsumeTaskPostQuotaTxWithNullExpiresGrant 回归测试：
// MySQL 8.0.46 严格模式（STRICT_TRANS_TABLES）下，旧 SQL 用 expires_at = ” 与
// datetime 列比较抛 Error 1292（Incorrect datetime value），且 UPDATE 目标表
// 直接出现在子查询 FROM 抛 Error 1093。修复（IS NULL 判定 + 派生表物化）后，
// 永不过期（expires_at IS NULL）的赠送 grant 应能被正常消耗。
func TestConsumeTaskPostQuotaTxWithNullExpiresGrant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	tenantID := generateSnowflakeID()
	now := utcNow()

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if _, err := db.Exec(`UPDATE billing_account SET task_post_quota = 5 WHERE id = ?`, acc.ID); err != nil {
		t.Fatalf("seed quota: %v", err)
	}
	grantID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at)
		VALUES (?, ?, 'task_post', 2, 2, 'test', NULL, ?)`,
		grantID, tenantID, now,
	); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM billing_resource_grant WHERE id = ?`, grantID)
		_, _ = db.Exec(`DELETE FROM billing_account WHERE id = ?`, acc.ID)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()
	if err := consumeTaskPostQuotaTx(ctx, tx, tenantID); err != nil {
		t.Fatalf("consumeTaskPostQuotaTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var remaining, quota int64
	if err := db.QueryRow(`SELECT remaining FROM billing_resource_grant WHERE id = ?`, grantID).Scan(&remaining); err != nil {
		t.Fatalf("read grant: %v", err)
	}
	if err := db.QueryRow(`SELECT task_post_quota FROM billing_account WHERE id = ?`, acc.ID).Scan(&quota); err != nil {
		t.Fatalf("read quota: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("grant remaining: expected 1, got %d", remaining)
	}
	if quota != 4 {
		t.Fatalf("account quota: expected 4, got %d", quota)
	}
}
