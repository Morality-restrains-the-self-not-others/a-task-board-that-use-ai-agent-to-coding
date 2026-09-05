package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransactionsListExposesGrantChangeDisplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000001)
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 10},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000001/billing/transactions/?page=1&page_size=5", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var page map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	results, _ := page["results"].([]interface{})
	if len(results) == 0 {
		t.Fatalf("empty results: %s", rr.Body.String())
	}
	row, _ := results[0].(map[string]interface{})
	if row["points_source_type_display"] != "后台赠送" {
		t.Fatalf("points_source_type_display=%v", row["points_source_type_display"])
	}
	if row["change_display"] != "任务帖 +10 帖" {
		t.Fatalf("change_display=%v", row["change_display"])
	}
	if row["description"] != "管理员后台赠送：任务帖 +10 帖" {
		t.Fatalf("description=%v", row["description"])
	}
	changes, _ := row["resource_changes"].([]interface{})
	if len(changes) != 1 {
		t.Fatalf("resource_changes=%v", row["resource_changes"])
	}
	ch, _ := changes[0].(map[string]interface{})
	if ch["resource_type"] != ResourceTypeTaskPost || ch["display"] != "任务帖 +10 帖" {
		t.Fatalf("resource_changes[0]=%v", ch)
	}
}

func TestTransactionsListEnrichesLegacyGenericGrantDescription(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000002)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', '管理员后台赠送资源', 'admin_grant:legacy', ?, 0)`,
		generateSnowflakeID(), acc.ID, now,
	); err != nil {
		t.Fatalf("insert txn: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at)
		VALUES (?, ?, ?, 7, 7, '', '2099-12-31 23:59:59', ?)`,
		generateSnowflakeID(), tenantID, ResourceTypeTaskPost, now,
	); err != nil {
		t.Fatalf("insert grant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000002/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(list) == 0 {
		t.Fatal("expected transaction")
	}
	row := list[0]
	if row["points_source_type_display"] != "后台赠送" {
		t.Fatalf("points_source_type_display=%v", row["points_source_type_display"])
	}
	if row["change_display"] != "任务帖 +7 帖" {
		t.Fatalf("change_display=%v", row["change_display"])
	}
	if row["description"] != "管理员后台赠送：任务帖 +7 帖" {
		t.Fatalf("description=%v", row["description"])
	}
}

// 同秒多次 admin_grant 时，enrich 必须按 billing_transaction_id 精确归属，
// 不能把同一秒的另一笔赠送串到当前流水上（OPT-20260818-026 / OPT-20260819-015）。
func TestTransactionsListGrantFKEnrichMatchesOwnTxn(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000004)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	now := utcNow()
	t1 := generateSnowflakeID()
	t2 := generateSnowflakeID()
	g1 := generateSnowflakeID()
	g2 := generateSnowflakeID()
	for _, ins := range [][]interface{}{
		{
			`INSERT INTO billing_transaction (
				id, account_id, transaction_type, amount, balance_before, balance_after,
				points_source_type, description, transaction_id, created_at, usage_amount
			) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', '管理员后台赠送资源', ?, ?, 0)`,
			t1, acc.ID, "admin_grant:t1", now,
		},
		{
			`INSERT INTO billing_transaction (
				id, account_id, transaction_type, amount, balance_before, balance_after,
				points_source_type, description, transaction_id, created_at, usage_amount
			) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', '管理员后台赠送资源', ?, ?, 0)`,
			t2, acc.ID, "admin_grant:t2", now,
		},
		{
			`INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, billing_transaction_id)
			 VALUES (?, ?, ?, 10, 10, '', '2099-12-31 23:59:59', ?, ?)`,
			g1, tenantID, ResourceTypeTaskPost, now, t1,
		},
		{
			`INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at, billing_transaction_id)
			 VALUES (?, ?, ?, 5, 5, '', '2099-12-31 23:59:59', ?, ?)`,
			g2, tenantID, ResourceTypeTaskPost, now, t2,
		},
	} {
		q := ins[0].(string)
		if _, err := db.Exec(q, ins[1:]...); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000004/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(list))
	}
	byTxnID := map[string]map[string]interface{}{}
	for _, row := range list {
		txnID, _ := row["transaction_id"].(string)
		byTxnID[txnID] = row
	}
	row1 := byTxnID["admin_grant:t1"]
	row2 := byTxnID["admin_grant:t2"]
	if row1 == nil || row2 == nil {
		t.Fatalf("missing txn rows: %v", list)
	}
	if row1["change_display"] != "任务帖 +10 帖" {
		t.Fatalf("t1 change_display=%v, want 任务帖 +10 帖", row1["change_display"])
	}
	if row2["change_display"] != "任务帖 +5 帖" {
		t.Fatalf("t2 change_display=%v, want 任务帖 +5 帖", row2["change_display"])
	}
}

// adminGrantResources 写路径必须同时写入 billing_resource_grant.billing_transaction_id
// 与 billing_transaction.related_order_id，供 enrich 稳定关联（OPT-20260818-026）。
func TestAdminGrantWritesStableFK(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000005)
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 3},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	var txnID int64
	var relatedOrderID sql.NullInt64
	if err := db.QueryRow(
		`SELECT id, related_order_id FROM billing_transaction WHERE account_id = ?`,
		acc.ID,
	).Scan(&txnID, &relatedOrderID); err != nil {
		t.Fatalf("load txn: %v", err)
	}
	if !relatedOrderID.Valid || relatedOrderID.Int64 <= 0 {
		t.Fatalf("related_order_id not set on admin_grant txn")
	}
	var grantTxnID int64
	if err := db.QueryRow(
		`SELECT billing_transaction_id FROM billing_resource_grant WHERE tenant_id = ?`,
		tenantID,
	).Scan(&grantTxnID); err != nil {
		t.Fatalf("load grant: %v", err)
	}
	if grantTxnID != txnID {
		t.Fatalf("grant billing_transaction_id=%d want %d", grantTxnID, txnID)
	}
}

func TestTransactionsListQuotaConsumptionChangeDisplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9410000003)
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
		) VALUES (?, ?, 'consumption', 0, 0, 0, 'quota_consumption', ?, 1, '创建任务帖消耗配额', 'task_post_quota:x', ?)`,
		generateSnowflakeID(), acc.ID, unitID, utcNow(),
	); err != nil {
		t.Fatalf("insert txn: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000003/billing/transactions/", nil)
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
	row := list[0]
	if row["points_source_type_display"] != "配额消耗" {
		t.Fatalf("points_source_type_display=%v", row["points_source_type_display"])
	}
	if row["change_display"] != "-1 智能体任务" {
		t.Fatalf("change_display=%v", row["change_display"])
	}
}
