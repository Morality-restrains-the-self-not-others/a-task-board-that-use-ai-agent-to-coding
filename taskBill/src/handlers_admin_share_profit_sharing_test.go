package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func adminShareReq(t *testing.T, id int64, reason, idemKey, actor string, extraHeaders map[string]string) *http.Request {
	t.Helper()
	path := "/api/system-admin/profit-sharing/" + formatID(id) + "/share/"
	req := staffAdminJSONReq(t, http.MethodPost, path, map[string]interface{}{"reason": reason})
	req.Header.Set("X-User-Id", actor)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	return req
}

func TestHandleSystemAdminShareProfitSharingRejectsShortReason(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-09-01T00:00:00Z")
	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-ps-1", nil
	}
	req := adminShareReq(t, psID, "短", "ik-short", "admin-1", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calls != 0 {
		t.Fatalf("CreateOrder calls=%d want 0", calls)
	}
}

func TestHandleSystemAdminShareProfitSharingForbiddenForNonStaff(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-09-01T00:00:00Z")
	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-forbidden", "user-1", nil)
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSystemAdminShareProfitSharingSuccessAndIdempotentReplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-20T00:00:00Z")
	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-ps-admin-1", nil
	}

	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-share-1", "admin-9", map[string]string{
		"X-Impersonator-Id":          "staff-imp",
		"X-Impersonation-Session-Id": "impsess-1",
	})
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	if body["state"] != "shared" {
		t.Fatalf("body=%v", body)
	}
	if calls != 1 {
		t.Fatalf("CreateOrder calls=%d want 1", calls)
	}

	var outcome, actor, reason, impersonator string
	if err := db.QueryRow(`
		SELECT outcome, actor_user_id, reason, impersonator_user_id
		FROM billing_profit_sharing_admin_action WHERE idempotency_key = ?`, "ik-share-1").
		Scan(&outcome, &actor, &reason, &impersonator); err != nil {
		t.Fatalf("audit: %v", err)
	}
	if outcome != "succeeded" || actor != "admin-9" || reason != "客服复核后补发起分账" || impersonator != "staff-imp" {
		t.Fatalf("audit outcome=%s actor=%s reason=%s impersonator=%s", outcome, actor, reason, impersonator)
	}

	req2 := adminShareReq(t, psID, "客服复核后补发起分账", "ik-share-1", "admin-9", nil)
	rec2 := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	body2 := decodeJSONMap(t, rec2)
	if body2["idempotent"] != true {
		t.Fatalf("replay body=%v", body2)
	}
	if calls != 1 {
		t.Fatalf("replay CreateOrder calls=%d want 1", calls)
	}
}

func TestHandleSystemAdminShareProfitSharingMuxRoute(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		return "wx-ps-mux", nil
	}
	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-mux-1", "admin-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mux status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleSystemAdminShareProfitSharingBypassesReferrerFreeze(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-09-10T00:00:00Z")
	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-ps-freeze-bypass", nil
	}
	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-freeze-bypass", "admin-1", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calls != 1 {
		t.Fatalf("CreateOrder calls=%d want 1 (admin bypasses settle_after freeze)", calls)
	}
}

func TestHandleSystemAdminShareProfitSharingRejectsEmptyOpenid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubWechatPayIdentity(t, wechatPayIdentity{}, nil)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = '' WHERE id = ?`, psID); err != nil {
		t.Fatalf("clear openid: %v", err)
	}
	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-should-not-run", nil
	}
	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-empty-openid", "admin-1", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calls != 0 {
		t.Fatalf("CreateOrder calls=%d want 0", calls)
	}
	body := decodeJSONMap(t, rec)
	msg, _ := body["error"].(string)
	if !strings.Contains(msg, "未绑定微信收款账号") {
		t.Fatalf("error=%q", msg)
	}
	if strings.Contains(msg, "PARAM_ERROR") || strings.Contains(msg, "HTTP 400") {
		t.Fatalf("leaked wechat dump: %s", msg)
	}
	var failReason, status string
	if err := db.QueryRow(`SELECT status, COALESCE(fail_reason, '') FROM billing_profit_sharing WHERE id = ?`, psID).
		Scan(&status, &failReason); err != nil {
		t.Fatalf("load: %v", err)
	}
	if status != psStatusFailed {
		t.Fatalf("status=%s want failed", status)
	}
	if failReason != errReferrerOpenidMissing.Error() {
		t.Fatalf("fail_reason=%q want referrer_openid_missing", failReason)
	}
}

func TestHandleSystemAdminShareProfitSharingRequiresIdempotencyKey(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-09-01T00:00:00Z")
	req := adminShareReq(t, psID, "客服复核后补发起分账", "", "admin-1", nil)
	req.Header.Del("Idempotency-Key")
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
