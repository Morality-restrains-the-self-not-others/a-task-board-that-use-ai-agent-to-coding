package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

func TestMarkProfitSharingStatusPersistsFailTraceID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	ctx := tracelog.ContextWithTraceID(t.Context(), "ps-fail-trace-abc123")

	markProfitSharingStatus(ctx, psID, psStatusFailed, `create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST"}`)

	var reason, tid, status string
	if err := db.QueryRow(`
		SELECT status, COALESCE(fail_reason, ''), COALESCE(fail_trace_id, '')
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&status, &reason, &tid); err != nil {
		t.Fatalf("load: %v", err)
	}
	if status != psStatusFailed {
		t.Fatalf("status=%s want failed", status)
	}
	if reason == "" {
		t.Fatal("fail_reason empty")
	}
	if tid != "ps-fail-trace-abc123" {
		t.Fatalf("fail_trace_id=%q want ps-fail-trace-abc123", tid)
	}

	markProfitSharingStatus(ctx, psID, psStatusProcessing, "")
	if err := db.QueryRow(`
		SELECT status, COALESCE(fail_reason, ''), COALESCE(fail_trace_id, '')
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&status, &reason, &tid); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if status != psStatusProcessing {
		t.Fatalf("status=%s want processing", status)
	}
	if reason != "" || tid != "" {
		t.Fatalf("processing should clear fail_reason and fail_trace_id, got reason=%q tid=%q", reason, tid)
	}
}

func TestHandleSystemAdminListProfitSharingReturnsFailTraceID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusFailed, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`
		UPDATE billing_profit_sharing
		SET fail_reason = ?, fail_trace_id = ?
		WHERE id = ?`,
		`create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"appid与openid不匹配"}`,
		"list-ps-fail-tid-01", psID); err != nil {
		t.Fatalf("seed fail_trace_id: %v", err)
	}

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/?status=failed")
	rec := httptest.NewRecorder()
	handleSystemAdminListProfitSharing(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	items, _ := body["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	row, _ := items[0].(map[string]interface{})
	if row["fail_trace_id"] != "list-ps-fail-tid-01" {
		t.Fatalf("fail_trace_id=%v want list-ps-fail-tid-01 row=%v", row["fail_trace_id"], row)
	}
	reason, _ := row["fail_reason"].(string)
	if reason == "" {
		t.Fatal("fail_reason empty")
	}
}

func TestHandleSystemAdminShareProfitSharingPersistsFailTraceID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(_ context.Context, _, _, _ string, _ []profitSharingReceiver) (string, error) {
		return "", fmt.Errorf(`create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST"}`)
	}

	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-fail-tid", "admin-1", nil)
	req = req.WithContext(tracelog.ContextWithTraceID(req.Context(), "share-ps-fail-tid-9"))
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("want share failure, got 200 body=%s", rec.Body.String())
	}

	var tid string
	if err := db.QueryRow(`SELECT COALESCE(fail_trace_id, '') FROM billing_profit_sharing WHERE id = ?`, psID).
		Scan(&tid); err != nil {
		t.Fatalf("load: %v", err)
	}
	if tid != "share-ps-fail-tid-9" {
		t.Fatalf("fail_trace_id=%q want share-ps-fail-tid-9", tid)
	}
}

func TestHandleSystemAdminGetOrderProfitSharingFailTraceID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WXPSFT", "420000psft")
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	if _, err := db.Exec(`
		UPDATE billing_profit_sharing SET status = ?, fail_reason = ?, fail_trace_id = ? WHERE id = ?`,
		psStatusFailed, "qualification_revoked", "order-ps-fail-tid", psID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/orders/"+formatID(orderID)+"/")
	rec := httptest.NewRecorder()
	handleSystemAdminGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	raw, _ := body["profit_sharing"].([]interface{})
	if len(raw) != 1 {
		t.Fatalf("profit_sharing=%v", body["profit_sharing"])
	}
	row, _ := raw[0].(map[string]interface{})
	if row["fail_trace_id"] != "order-ps-fail-tid" {
		t.Fatalf("fail_trace_id=%v", row["fail_trace_id"])
	}
}
