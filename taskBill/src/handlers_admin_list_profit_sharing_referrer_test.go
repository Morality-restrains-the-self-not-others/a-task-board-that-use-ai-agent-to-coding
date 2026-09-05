package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedReferrerFlaggedOrder(t *testing.T, referrerID, referredID int64, status, settleAfter string) (orderID, psID int64) {
	t.Helper()
	tenantID := generateSnowflakeID()
	orderID = generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXREF"+formatID(orderID), "420000"+formatID(orderID))
	if _, err := db.Exec(`UPDATE billing_resource_order SET user_id = ? WHERE id = ?`, referredID, orderID); err != nil {
		t.Fatalf("set order buyer: %v", err)
	}
	if err := upsertReferralEdge(formatID(referrerID), formatID(referredID), tenantID, utcNow(), "CH", true); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	psID = seedProfitSharingRecordStatus(t, orderID, tenantID, status, settleAfter)
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_user_id = ? WHERE id = ?`, formatID(referrerID), psID); err != nil {
		t.Fatalf("set ps referrer: %v", err)
	}
	return orderID, psID
}

func TestHandleSystemAdminListProfitSharingByReferrerGraph(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	referrerA := generateSnowflakeID()
	u1 := generateSnowflakeID()
	u2 := generateSnowflakeID()
	u1Order, _ := seedReferrerFlaggedOrder(t, referrerA, u1, psStatusPending, "2026-08-20T00:00:00Z")
	u2Tenant := generateSnowflakeID()
	u2Order := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, u2Tenant, u2Order, u2, 2000)
	if err := upsertReferralEdge(formatID(referrerA), formatID(u2), u2Tenant, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	otherReferrer := generateSnowflakeID()
	otherBuyer := generateSnowflakeID()
	seedReferrerFlaggedOrder(t, otherReferrer, otherBuyer, psStatusPending, "2026-08-21T00:00:00Z")
	seedAdminProfitSharingRow(t, psStatusPending, "2026-08-22T00:00:00Z")
	if err := upsertProfitSharingReceiverRow(formatID(referrerA), "wxapp", "secret-openid", psReceiverRegistered, "", true); err != nil {
		t.Fatal(err)
	}

	req := staffAdminOrderReq(t, http.MethodGet,
		"/api/system-admin/profit-sharing/?referrer_user_id="+formatID(referrerA)+"&status=all")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if body["receiver_registration_status"] != psReceiverRegistered {
		t.Fatalf("receiver_registration_status=%v", body["receiver_registration_status"])
	}
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v want 1 (only flagged U1)", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["referred_user_id"] != formatID(u1) {
		t.Fatalf("referred_user_id=%v want %s", row["referred_user_id"], formatID(u1))
	}
	if row["order_number"] != "ORD"+formatID(u1Order) {
		t.Fatalf("order_number=%v", row["order_number"])
	}
	if _, has := row["referrer_openid"]; has {
		t.Fatal("referrer_openid key must not appear")
	}
	if row["app_id"] != "wxapp" {
		t.Fatalf("app_id=%v want wxapp", row["app_id"])
	}
	if row["openid"] != "secret-openid" {
		t.Fatalf("openid=%v want receiver pair, not snapshot", row["openid"])
	}
	if _, has := row["wechat_transaction_id"]; !has {
		t.Fatal("wechat_transaction_id must appear for platform staff")
	}
	if _, has := row["wechat_profit_sharing_id"]; !has {
		t.Fatal("wechat_profit_sharing_id must appear for platform staff")
	}
	if _, has := row["out_trade_no"]; !has {
		t.Fatal("out_trade_no must appear for platform staff")
	}
	if _, has := row["out_profit_sharing_no"]; !has {
		t.Fatal("out_profit_sharing_no must appear for platform staff")
	}
}

func TestHandleSystemAdminListProfitSharingByReferrerIncludesFinishedWhenAll(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, _ = seedReferrerFlaggedOrder(t, referrer, buyer, psStatusFinished, "2026-07-01T00:00:00Z")

	req := staffAdminOrderReq(t, http.MethodGet,
		"/api/system-admin/profit-sharing/?referrer_user_id="+formatID(referrer)+"&status=all")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body := decodeJSONMap(t, rec)
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("total=%v want 1 finished", body["total"])
	}

	req = staffAdminOrderReq(t, http.MethodGet,
		"/api/system-admin/profit-sharing/?referrer_user_id="+formatID(referrer))
	rec = httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body = decodeJSONMap(t, rec)
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("empty status with referrer should default all, total=%v", body["total"])
	}
}

func TestHandleSystemAdminListProfitSharingWithoutReferrerKeepsGlobalQueue(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	seedAdminProfitSharingRow(t, psStatusPending, "2026-08-01T00:00:00Z")
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-02T00:00:00Z")

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body := decodeJSONMap(t, rec)
	if int(body["total"].(float64)) != 2 {
		t.Fatalf("global queue total=%v want 2", body["total"])
	}
}

func TestHandleSystemAdminListProfitSharingReferrerForbidden(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/profit-sharing/?referrer_user_id=1", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestHandleSystemAdminListProfitSharingReferrerOmitsUnflaggedPaidOrder(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	seedPaidOrderForProfitSharing(t, tenantID, orderID, buyer, 1000)
	if err := upsertReferralEdge(formatID(referrer), formatID(buyer), tenantID, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	req := staffAdminOrderReq(t, http.MethodGet,
		fmt.Sprintf("/api/system-admin/profit-sharing/?referrer_user_id=%s&status=all", formatID(referrer)))
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	body := decodeJSONMap(t, rec)
	if int(body["total"].(float64)) != 0 {
		t.Fatalf("unflagged paid order leaked: %v", body)
	}
}

func TestHandleSystemAdminListProfitSharingOpenidFallsBackToReceiver(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = '' WHERE id = ?`, psID); err != nil {
		t.Fatalf("clear snapshot openid: %v", err)
	}
	if err := upsertProfitSharingReceiverRow(formatID(referrer), "wx-login-app", "oRECEIVER-openid", psReceiverRegistered, "", true); err != nil {
		t.Fatal(err)
	}

	req := staffAdminOrderReq(t, http.MethodGet,
		"/api/system-admin/profit-sharing/?referrer_user_id="+formatID(referrer)+"&status=all")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v want 1", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["app_id"] != "wx-login-app" {
		t.Fatalf("app_id=%v", row["app_id"])
	}
	if row["openid"] != "oRECEIVER-openid" {
		t.Fatalf("openid=%v want receiver fallback", row["openid"])
	}
}

func TestHandleSystemAdminListProfitSharingUsesReceiverPairNotStaleSnapshot(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	referrer := generateSnowflakeID()
	buyer := generateSnowflakeID()
	_, psID := seedReferrerFlaggedOrder(t, referrer, buyer, psStatusFailed, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = ? WHERE id = ?`,
		"osshz2WnJeZXaLGYtxwVmZwyf0Xg", psID); err != nil {
		t.Fatalf("stale web snapshot: %v", err)
	}
	if err := upsertProfitSharingReceiverRow(formatID(referrer), "wx31273ca77c89dffe", "oNsrS1BPzk9Lr7IvyVl9lHIQKwB8", psReceiverRegistered, "", true); err != nil {
		t.Fatal(err)
	}

	req := staffAdminOrderReq(t, http.MethodGet,
		"/api/system-admin/profit-sharing/?referrer_user_id="+formatID(referrer)+"&status=all")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v want 1", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["app_id"] != "wx31273ca77c89dffe" {
		t.Fatalf("app_id=%v want mp/pay appid", row["app_id"])
	}
	if row["openid"] != "oNsrS1BPzk9Lr7IvyVl9lHIQKwB8" {
		t.Fatalf("openid=%v want wechat_identity mp pair, not web snapshot", row["openid"])
	}
}
