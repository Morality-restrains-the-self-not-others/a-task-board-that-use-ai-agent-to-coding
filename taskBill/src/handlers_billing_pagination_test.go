package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransactionsListPagedBody(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	var unitID int64
	if err := db.QueryRow(`SELECT id FROM billing_unit WHERE unit_type = 'server_start' LIMIT 1`).Scan(&unitID); err != nil {
		t.Fatalf("unit: %v", err)
	}
	for i := 0; i < 3; i++ {
		_, err = db.Exec(`
			INSERT INTO billing_transaction (
				id, account_id, transaction_type, amount, balance_before, balance_after,
				points_source_type, billing_unit_id, usage_amount, description, transaction_id, created_at
			) VALUES (?, ?, 'consumption', 1, 10, 9, 'consumption', ?, 1, 'row', ?, NOW())`,
			generateSnowflakeID(), acc.ID, unitID, "txn-page-"+string(rune('a'+i)),
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562496/billing/transactions/list_filtered/?page=1&page_size=2",
		nil,
	)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, true)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (want paged object)", err)
	}
	if int(body["total"].(float64)) < 3 {
		t.Fatalf("total=%v", body["total"])
	}
	if int(body["page_size"].(float64)) != 2 {
		t.Fatalf("page_size=%v", body["page_size"])
	}
	results, ok := body["results"].([]interface{})
	if !ok || len(results) != 2 {
		t.Fatalf("results=%v", body["results"])
	}
	row0, _ := results[0].(map[string]interface{})
	bu, _ := row0["billing_unit"].(map[string]interface{})
	if bu == nil || bu["name"] == nil || bu["name"] == "" {
		t.Fatalf("expected nested billing_unit.name, got %#v", row0["billing_unit"])
	}
}
