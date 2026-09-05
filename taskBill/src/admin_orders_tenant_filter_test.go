package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSystemAdminListOrdersFiltersTenantID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	tidA := int64(880301)
	tidB := int64(880302)
	idA := generateSnowflakeID()
	idB := generateSnowflakeID()
	seedWechatAccountOrder(t, idA, tidA, 1, "ORD-TA", "", "")
	seedWechatAccountOrder(t, idB, tidB, 1, "ORD-TB", "", "")

	req := staffAdminOrderReq(t, http.MethodGet, fmt.Sprintf("/api/system-admin/orders/?tenant_id=%d&limit=15", tidA))
	rec := httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	raw, _ := body["orders"].([]interface{})
	if len(raw) != 1 {
		t.Fatalf("want 1 order for tenant A, got %v", body)
	}
	row, _ := raw[0].(map[string]interface{})
	if row["order_number"] != "ORD-TA" {
		t.Fatalf("order=%v", row)
	}
}

func TestHandleSystemAdminListOrdersInvalidTenantID(t *testing.T) {
	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/orders/?tenant_id=not-a-number")
	rec := httptest.NewRecorder()
	handleSystemAdminListOrders(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid tenant_id got %d body=%s", rec.Code, rec.Body.String())
	}
}
