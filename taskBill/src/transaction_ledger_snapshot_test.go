package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestTransactionsListLedgerSnapshotYuan(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000004)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 30, 100, 70, 'consumption', ?, 1, '启动消耗', 'txn-ledger-cash', ?)`,
		generateSnowflakeID(), acc.ID, unitID, utcNow(),
	); err != nil {
		t.Fatalf("insert txn: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000004/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("expected transaction")
	}
	snap, ok := list[0]["ledger_snapshot"].(map[string]interface{})
	if !ok {
		t.Fatalf("ledger_snapshot=%v", list[0]["ledger_snapshot"])
	}
	if snap["balance_before_yuan"] != "1.00" || snap["balance_after_yuan"] != "0.70" {
		t.Fatalf("yuan fields=%v", snap)
	}
	display, _ := snap["display"].(string)
	if display == "" || !containsAll(display, "1.00", "0.70") {
		t.Fatalf("display=%q", display)
	}
	if list[0]["change_display"] != "-0.30 元" {
		t.Fatalf("change_display=%v", list[0]["change_display"])
	}
}

func TestTransactionsListTaskPostRemainingReplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000005)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("task_post_quota", "任务帖配额消耗", 0)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	if _, err := db.Exec(`UPDATE billing_account SET task_post_quota = 8 WHERE id = ?`, acc.ID); err != nil {
		t.Fatalf("quota: %v", err)
	}

	grantTime := "2026-08-01 10:00:00"
	c1Time := "2026-08-01 11:00:00"
	c2Time := "2026-08-01 12:00:00"
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', '管理员后台赠送资源', 'admin_grant:replay', ?, 0)`,
		generateSnowflakeID(), acc.ID, grantTime,
	); err != nil {
		t.Fatalf("insert grant txn: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at)
		VALUES (?, ?, ?, 10, 8, '', '2099-12-31 23:59:59', ?)`,
		generateSnowflakeID(), tenantID, ResourceTypeTaskPost, grantTime,
	); err != nil {
		t.Fatalf("insert grant: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 0, 0, 0, 'quota_consumption', ?, 1, '消耗1', 'task_post_quota:c1', ?)`,
		generateSnowflakeID(), acc.ID, unitID, c1Time,
	); err != nil {
		t.Fatalf("insert c1: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 0, 0, 0, 'quota_consumption', ?, 1, '消耗2', 'task_post_quota:c2', ?)`,
		generateSnowflakeID(), acc.ID, unitID, c2Time,
	); err != nil {
		t.Fatalf("insert c2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000005/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len=%d body=%s", len(list), rr.Body.String())
	}
	// newest first: c2 remaining 8, c1 remaining 9, grant remaining 10
	want := []int64{8, 9, 10}
	for i, w := range want {
		snap, _ := list[i]["ledger_snapshot"].(map[string]interface{})
		display, _ := snap["display"].(string)
		if !strings.Contains(display, "任务帖剩余 "+strconv.FormatInt(w, 10)) {
			t.Fatalf("row %d display=%q want remaining %d", i, display, w)
		}
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
