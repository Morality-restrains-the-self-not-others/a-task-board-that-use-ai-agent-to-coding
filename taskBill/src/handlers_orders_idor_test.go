package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// OPT-20260815-001: handleGetOrder 须校验 URL 租户与订单归属（IDOR 防护）。
// 同租户 200；跨租户 404（不探测订单存在性）。

func seedOrderForIDORTest(t *testing.T) int64 {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	res, err := db.Exec(
		`INSERT INTO billing_resource_order(tenant_id, order_number, status, total_yuan_cents, created_at)
		 VALUES (?, ?, 'pending', 1000, ?)`,
		1001, fmt.Sprintf("ORD-IDOR-%d", time.Now().UnixNano()),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func TestHandleGetOrderSameTenantOK(t *testing.T) {
	orderID := seedOrderForIDORTest(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tenant/1001/billing/orders/%d/", orderID), nil)
	rec := httptest.NewRecorder()
	handleGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("same-tenant GET status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
}

func TestHandleGetOrderCrossTenant404(t *testing.T) {
	orderID := seedOrderForIDORTest(t)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tenant/9999/billing/orders/%d/", orderID), nil)
	rec := httptest.NewRecorder()
	handleGetOrder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant GET status=%d body=%s, want 404", rec.Code, rec.Body.String())
	}
}

func TestHandlePayOrderCrossTenant404(t *testing.T) {
	orderID := seedOrderForIDORTest(t)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tenant/9999/billing/orders/%d/pay/", orderID), nil)
	rec := httptest.NewRecorder()
	handlePayOrder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant pay status=%d body=%s, want 404", rec.Code, rec.Body.String())
	}
}

func TestHandleOrderCancelCrossTenant404(t *testing.T) {
	orderID := seedOrderForIDORTest(t)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tenant/9999/billing/orders/%d/cancel/", orderID), nil)
	rec := httptest.NewRecorder()
	handleOrderCancel(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant cancel status=%d body=%s, want 404", rec.Code, rec.Body.String())
	}
}
