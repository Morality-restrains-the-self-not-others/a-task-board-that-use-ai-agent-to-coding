package main

import (
	"context"
	"testing"
)

func TestRenewalDeductsQuotaNotBalance(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := generateSnowflakeID()
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	const startQuota int64 = 3
	const startBalance int64 = 9999
	if _, err := db.Exec(`UPDATE billing_account SET task_post_quota = ?, balance = ? WHERE id = ?`,
		startQuota, startBalance, acc.ID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	taskID := "task_renew_quota_only"
	result, err := consumeTaskPostRenewal(ctx, tenantID, taskID, "ws1", "u1", "p1", "", "")
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if got := asTestInt64(result["cost_cents"]); got != 0 {
		t.Fatalf("cost_cents=%d want 0 (续存不得扣费)", got)
	}
	if got := asTestInt64(result["quota_consumed"]); got != 1 {
		t.Fatalf("quota_consumed=%d want 1", got)
	}

	var quota, balance int64
	if err := db.QueryRow(`SELECT task_post_quota, balance FROM billing_account WHERE id = ?`, acc.ID).
		Scan(&quota, &balance); err != nil {
		t.Fatalf("read account: %v", err)
	}
	if quota != startQuota-1 {
		t.Fatalf("quota=%d want %d", quota, startQuota-1)
	}
	if balance != startBalance {
		t.Fatalf("balance=%d want %d (续存不得扣钱包)", balance, startBalance)
	}

	var amount int64
	var unitType string
	err = db.QueryRow(`
		SELECT bt.amount, u.unit_type
		FROM billing_transaction bt
		JOIN billing_unit u ON bt.billing_unit_id = u.id
		WHERE bt.task_id = ?`, taskID).Scan(&amount, &unitType)
	if err != nil {
		t.Fatalf("read txn: %v", err)
	}
	if amount != 0 {
		t.Fatalf("txn.amount=%d want 0", amount)
	}
	if unitType != "task_post_quota" {
		t.Fatalf("unit_type=%q want task_post_quota (不得用标价的 server_start_renewal)", unitType)
	}
}

func TestRenewalRejectsWhenQuotaEmpty(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := generateSnowflakeID()
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	const startBalance int64 = 8888
	if _, err := db.Exec(`UPDATE billing_account SET task_post_quota = 0, balance = ? WHERE id = ?`,
		startBalance, acc.ID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err = consumeTaskPostRenewal(ctx, tenantID, "task_renew_empty", "ws1", "u1", "p1", "", "")
	if err == nil {
		t.Fatal("expected insufficient quota error")
	}
	if _, ok := err.(*InsufficientBalanceError); !ok {
		t.Fatalf("err type %T want InsufficientBalanceError: %v", err, err)
	}

	var quota, balance int64
	if err := db.QueryRow(`SELECT task_post_quota, balance FROM billing_account WHERE id = ?`, acc.ID).
		Scan(&quota, &balance); err != nil {
		t.Fatalf("read account: %v", err)
	}
	if quota != 0 {
		t.Fatalf("quota=%d want 0", quota)
	}
	if balance != startBalance {
		t.Fatalf("balance=%d want %d", balance, startBalance)
	}
}

func asTestInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return -1
	}
}
