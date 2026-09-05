package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func staffAdminOrderReq(t *testing.T, method, path string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	return req
}

func decodeJSONMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return body
}

func TestHandleSystemAdminGetOrderReturnsProfitSharing(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 96101
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXPS1", "420000ps1")
	seedProfitSharingRecord(t, orderID, tenantID)
	if err := upsertProfitSharingReceiverRow("referrer-1", "wxapp", "secret-openid", psReceiverRegistered, "", true); err != nil {
		t.Fatal(err)
	}

	req := staffAdminOrderReq(t, http.MethodGet, fmt.Sprintf("/api/system-admin/orders/%d/", orderID))
	rec := httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	raw, ok := body["profit_sharing"].([]interface{})
	if !ok || len(raw) != 1 {
		t.Fatalf("profit_sharing=%v", body["profit_sharing"])
	}
	row, _ := raw[0].(map[string]interface{})
	if row["receiver_user_id"] != "referrer-1" {
		t.Fatalf("receiver=%v", row["receiver_user_id"])
	}
	if row["amount_yuan"] != "0.50" {
		t.Fatalf("amount_yuan=%v", row["amount_yuan"])
	}
	// OPT-20260826-012：订单详情 profit_sharing[] 与管理端队列一致下发 app_id/openid
	// （接收方成对身份）；原始 referrer_openid 快照键仍不暴露。
	if _, has := row["referrer_openid"]; has {
		t.Fatal("raw referrer_openid key must not appear")
	}
	if row["app_id"] != "wxapp" {
		t.Fatalf("app_id=%v want wxapp (receiver pair)", row["app_id"])
	}
	if row["openid"] != "secret-openid" {
		t.Fatalf("openid=%v want receiver pair openid, not snapshot", row["openid"])
	}
}

func TestHandleSystemAdminGetOrderEmptyProfitSharing(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 96102
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXPS2", "420000ps2")

	req := staffAdminOrderReq(t, http.MethodGet, fmt.Sprintf("/api/system-admin/orders/%d/", orderID))
	rec := httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	raw, ok := body["profit_sharing"].([]interface{})
	if !ok || len(raw) != 0 {
		t.Fatalf("want empty slice got %v", body["profit_sharing"])
	}
}

func TestHandleSystemAdminGetOrderAuth(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/1/", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/system-admin/orders/1/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec = httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member status=%d want 403", rec.Code)
	}
}

func TestHandleSystemAdminGetOrderNotFound(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/orders/999999999999999999/")
	rec := httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetOrderOmitsProfitSharingKey(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 96103
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXPS3", "420000ps3")
	seedProfitSharingRecord(t, orderID, tenantID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tenant/%d/billing/orders/%d/", tenantID, orderID), nil)
	rec := httptest.NewRecorder()
	handleGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if _, has := body["profit_sharing"]; has {
		t.Fatalf("tenant GET must omit profit_sharing, got %v", body["profit_sharing"])
	}
}
