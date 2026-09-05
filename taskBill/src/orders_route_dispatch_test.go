package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mock-complete 已下线：POST 应落入通用订单详情路由，因方法不是 GET → 405。
func TestRouteDispatchMockCompleteRemoved(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/billing/orders/42/mock-complete/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST mock-complete 已移除，应 405，实际 %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "mock 支付成功") {
		t.Fatalf("mock-complete 入账路径仍可达: %s", rec.Body.String())
	}
}

// 反向回归：GET 普通订单详情仍走 handleGetOrder（不破坏既有路由）
func TestRouteDispatchGetOrderStillWorks(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()

	// 不存在的订单 → 404（而非 405），证明命中了 handleGetOrder
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/billing/orders/999999/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatalf("GET 订单详情被 405 拦截，路由顺序被破坏")
	}
	// 404 或 401（鉴权）都说明进入了订单处理路径而非 405
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("GET 订单详情预期 404/401/403 之一（进入 handleGetOrder），实际 %d body=%s", rec.Code, rec.Body.String())
	}
}
