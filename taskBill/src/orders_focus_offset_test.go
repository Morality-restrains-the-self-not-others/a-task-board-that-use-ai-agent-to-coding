package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 退款审批页「关联订单」深链：列表 API 带 order_id 时须把 offset 对齐到含该订单的页。

func seedOrdersForFocusOffset(t *testing.T, tenantID int64, n int) []int64 {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	ids := make([]int64, 0, n)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		// 越晚创建越新；列表 ORDER BY created_at DESC
		created := base.Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
		res, err := db.Exec(
			`INSERT INTO billing_resource_order(tenant_id, order_number, status, total_yuan_cents, created_at)
			 VALUES (?, ?, 'paid', 1000, ?)`,
			tenantID, fmt.Sprintf("ORD-FOCUS-%d-%d", time.Now().UnixNano(), i), created,
		)
		if err != nil {
			t.Fatalf("seed order %d: %v", i, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestTenantOrderFocusOffsetAlignsPage(t *testing.T) {
	const tenantID int64 = 88001
	ids := seedOrdersForFocusOffset(t, tenantID, 20)
	// ids[0] 最早 → 列表末尾；ids[19] 最新 → 列表第 1 条
	oldest := ids[0]
	off, ok, err := tenantOrderFocusOffset(tenantID, oldest, "", 15)
	if err != nil {
		t.Fatalf("tenantOrderFocusOffset: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for oldest order")
	}
	// 19 条更新 → rank=19 → page2 offset=15
	if off != 15 {
		t.Fatalf("oldest offset=%d, want 15", off)
	}

	newest := ids[19]
	off2, ok2, err := tenantOrderFocusOffset(tenantID, newest, "", 15)
	if err != nil {
		t.Fatalf("tenantOrderFocusOffset newest: %v", err)
	}
	if !ok2 || off2 != 0 {
		t.Fatalf("newest offset=%d ok=%v, want 0/true", off2, ok2)
	}
}

func TestAdminOrderFocusOffsetAlignsPage(t *testing.T) {
	// 跨租户列表：order_id 对齐不应受租户归属限制。
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	ids := make([]int64, 0, 20)
	base := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		tenantID := int64(88010 + i%2)
		created := base.Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
		res, err := db.Exec(
			`INSERT INTO billing_resource_order(tenant_id, order_number, status, total_yuan_cents, created_at)
			 VALUES (?, ?, 'paid', 1000, ?)`,
			tenantID, fmt.Sprintf("ORD-ADMIN-FOCUS-%d-%d", time.Now().UnixNano(), i), created,
		)
		if err != nil {
			t.Fatalf("seed order %d: %v", i, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	// ids[0] 最早 → 全局列表末尾；ids[19] 最新 → 全局第 1 条
	off, ok, err := adminOrderFocusOffset(ids[0], "", 15)
	if err != nil {
		t.Fatalf("adminOrderFocusOffset: %v", err)
	}
	if !ok || off != 15 {
		t.Fatalf("oldest offset=%d ok=%v, want 15/true", off, ok)
	}
	off2, ok2, err := adminOrderFocusOffset(ids[19], "", 15)
	if err != nil {
		t.Fatalf("adminOrderFocusOffset newest: %v", err)
	}
	if !ok2 || off2 != 0 {
		t.Fatalf("newest offset=%d ok=%v, want 0/true", off2, ok2)
	}
}

func TestDoAdminListOrdersFocusOrderID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	ids := make([]int64, 0, 20)
	base := time.Date(2026, 8, 2, 14, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		tenantID := int64(88020 + i%3)
		created := base.Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
		res, err := db.Exec(
			`INSERT INTO billing_resource_order(tenant_id, order_number, status, total_yuan_cents, created_at)
			 VALUES (?, ?, 'paid', 1000, ?)`,
			tenantID, fmt.Sprintf("ORD-ADMIN-LIST-%d-%d", time.Now().UnixNano(), i), created,
		)
		if err != nil {
			t.Fatalf("seed order %d: %v", i, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	oldest := ids[0]

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/system-admin/orders/?limit=15&order_id=%d", oldest),
		nil,
	)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["focus_order_id"] != formatID(oldest) {
		t.Fatalf("focus_order_id=%v want %s", body["focus_order_id"], formatID(oldest))
	}
	if int(body["offset"].(float64)) != 15 {
		t.Fatalf("offset=%v want 15", body["offset"])
	}
	orders, _ := body["orders"].([]interface{})
	found := false
	for _, raw := range orders {
		m := raw.(map[string]interface{})
		if m["id"] == formatID(oldest) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("focused order not in returned page")
	}
}

func TestDoAdminListOrdersFocusOrderIDMissingReturnsNoFocus(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/system-admin/orders/?limit=15&order_id=999999999999999999",
		nil,
	)
	rec := httptest.NewRecorder()
	doAdminListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, has := body["focus_order_id"]; has {
		t.Fatalf("unexpected focus_order_id for missing order: %v", body["focus_order_id"])
	}
	if int(body["offset"].(float64)) != 0 {
		t.Fatalf("offset=%v want 0", body["offset"])
	}
}

func TestHandleListOrdersFocusOrderID(t *testing.T) {
	const tenantID int64 = 88002
	ids := seedOrdersForFocusOffset(t, tenantID, 20)
	oldest := ids[0]

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/billing/orders/?limit=15&order_id=%d", tenantID, oldest),
		nil,
	)
	rec := httptest.NewRecorder()
	handleListOrders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["focus_order_id"] != formatID(oldest) {
		t.Fatalf("focus_order_id=%v want %s", body["focus_order_id"], formatID(oldest))
	}
	if int(body["offset"].(float64)) != 15 {
		t.Fatalf("offset=%v want 15", body["offset"])
	}
	orders, _ := body["orders"].([]interface{})
	found := false
	for _, raw := range orders {
		m := raw.(map[string]interface{})
		if m["id"] == formatID(oldest) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("focused order not in returned page")
	}
}
