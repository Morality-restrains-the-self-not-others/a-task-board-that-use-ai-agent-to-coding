package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransactionsAndUsagesNestBillingUnitObject(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	now := utcNow()
	txnID := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 30, 100, 70, 'consumption', NULL, NULL, NULL, NULL, ?, 1, '启动消耗', 'txn-nest-1', ?)`,
		txnID, acc.ID, unitID, now,
	)
	if err != nil {
		t.Fatalf("insert txn: %v", err)
	}
	usageID := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO billing_usage (
			id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
			description, usage_time
		) VALUES (?, ?, ?, 1, NULL, NULL, NULL, NULL, '启动消耗', ?)`,
		usageID, acc.ID, unitID, now,
	)
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	path := "/api/tenant/850256677331562496/billing/transactions/list_filtered/"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("transactions status=%d body=%s", rr.Code, rr.Body.String())
	}
	var txns []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &txns); err != nil {
		t.Fatalf("decode txns: %v", err)
	}
	if len(txns) == 0 {
		t.Fatal("expected at least one transaction")
	}
	bu, ok := txns[0]["billing_unit"].(map[string]interface{})
	if !ok {
		t.Fatalf("billing_unit want object, got %#v", txns[0]["billing_unit"])
	}
	if bu["name"] != "智能体任务" {
		t.Fatalf("billing_unit.name=%v", bu["name"])
	}
	if bu["unit"] != "帖/12个月" {
		t.Fatalf("billing_unit.unit=%v", bu["unit"])
	}
	if bu["id"] != formatID(unitID) {
		t.Fatalf("billing_unit.id=%v want %s", bu["id"], formatID(unitID))
	}

	ureq := httptest.NewRequest(http.MethodGet, "/api/tenant/850256677331562496/billing/usages/", nil)
	urr := httptest.NewRecorder()
	handleUsagesList(urr, ureq)
	if urr.Code != http.StatusOK {
		t.Fatalf("usages status=%d body=%s", urr.Code, urr.Body.String())
	}
	var usages []map[string]interface{}
	if err := json.Unmarshal(urr.Body.Bytes(), &usages); err != nil {
		t.Fatalf("decode usages: %v", err)
	}
	if len(usages) == 0 {
		t.Fatal("expected at least one usage")
	}
	ubu, ok := usages[0]["billing_unit"].(map[string]interface{})
	if !ok {
		t.Fatalf("usage billing_unit want object, got %#v", usages[0]["billing_unit"])
	}
	if ubu["name"] != "智能体任务" || ubu["unit"] != "帖/12个月" {
		t.Fatalf("usage billing_unit=%#v", ubu)
	}
}

func TestNestedBillingUnitMissingRowStillReturnsID(t *testing.T) {
	got := nestedBillingUnit(42, map[int64]map[string]interface{}{})
	if got["id"] != "42" {
		t.Fatalf("got %#v", got)
	}
	if _, hasName := got["name"]; hasName {
		t.Fatalf("orphan unit should not invent name: %#v", got)
	}
}
