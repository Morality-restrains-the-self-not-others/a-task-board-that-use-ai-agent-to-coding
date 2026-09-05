package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// OPT-20260808-016: 订单支付 SSE 化 — markOrderPaid 落地后经 taskSSE
// /internal/publish 推送 order_paid 事件（billing:user:{userId} hub key 推导）。

func TestPublishOrderPaidSSEPayload(t *testing.T) {
	var got map[string]interface{}
	var gotSecret string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Task-Sse-Secret")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldURL, oldSecret := cfg.TaskSseURL, cfg.TaskSseSecret
	cfg.TaskSseURL, cfg.TaskSseSecret = srv.URL, "sse-test-secret"
	defer func() { cfg.TaskSseURL, cfg.TaskSseSecret = oldURL, oldSecret }()

	order := &ResourceOrder{ID: 42, TenantID: 7, OrderNumber: "ORD-20260808-001", Status: OrderStatusPaid, UserID: "uid-abc-123"}
	publishOrderPaidSSE(context.Background(), order)

	if got == nil {
		t.Fatal("no publish request received")
	}
	if gotSecret != "sse-test-secret" {
		t.Errorf("secret header = %q, want sse-test-secret", gotSecret)
	}
	// 无 task_id → taskSSE normalizeInboundMessage 从 user_id 推导 billing:user:{uid}
	if got["user_id"] != "uid-abc-123" {
		t.Errorf("user_id = %v, want uid-abc-123", got["user_id"])
	}
	sd, ok := got["status_data"].(map[string]interface{})
	if !ok {
		t.Fatalf("status_data missing: %v", got)
	}
	if sd["event_name"] != "order_paid" || sd["status"] != "completed" {
		t.Errorf("event_name/status = %v/%v, want order_paid/completed", sd["event_name"], sd["status"])
	}
	if sd["order_id"] != float64(42) || sd["order_number"] != "ORD-20260808-001" {
		t.Errorf("order_id/order_number = %v/%v", sd["order_id"], sd["order_number"])
	}
	if sd["tenant_id"] != float64(7) {
		t.Errorf("tenant_id = %v, want 7", sd["tenant_id"])
	}
}

func TestPublishOrderPaidSSEWithoutUserID(t *testing.T) {
	// user_id 为空（未 link 微信/网关未注入）：跳过发布，不 panic
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldURL := cfg.TaskSseURL
	cfg.TaskSseURL = srv.URL
	defer func() { cfg.TaskSseURL = oldURL }()

	publishOrderPaidSSE(context.Background(), &ResourceOrder{ID: 1, TenantID: 2, OrderNumber: "ORD-X", Status: OrderStatusPaid})
	if hit {
		t.Error("publish should be skipped when user_id empty")
	}
}

func TestPublishOrderPaidSSEDisabled(t *testing.T) {
	// TaskSseURL 为空 → 直接返回，不 panic
	oldURL := cfg.TaskSseURL
	cfg.TaskSseURL = ""
	defer func() { cfg.TaskSseURL = oldURL }()
	publishOrderPaidSSE(context.Background(), &ResourceOrder{ID: 1, TenantID: 2, OrderNumber: "ORD-Y", Status: OrderStatusPaid, UserID: "uid-1"})
}

func TestPublishOrderPaidSSEUpstreamError(t *testing.T) {
	// 上游 500：仅记录日志，不 panic
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldURL := cfg.TaskSseURL
	cfg.TaskSseURL = srv.URL
	defer func() { cfg.TaskSseURL = oldURL }()

	publishOrderPaidSSE(context.Background(), &ResourceOrder{ID: 1, TenantID: 2, OrderNumber: "ORD-Z", Status: OrderStatusPaid, UserID: "uid-2"})
}

func TestMarkOrderPaidPublishesSSEOnce(t *testing.T) {
	// markOrderPaid 成功路径发布一次 order_paid SSE；幂等路径不重复发布
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	orig := publishOrderPaidSSEFn
	defer func() { publishOrderPaidSSEFn = orig }()

	const tenantID = 9000000020
	const userID = "9000000021"
	const orderNumber = "ORD-SSE-001"
	orderID := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 1000, ?, ?)`,
		generateSnowflakeID(), tenantID, utcNow(), utcNow()); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at, user_id)
		VALUES (?, ?, ?, 'pending', 100, ?, ?)`,
		orderID, tenantID, orderNumber, utcNow(), userID,
	)
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}

	events := 0
	published := make(chan struct{}, 2)
	// markOrderPaid 内 order 是支付前 loadOrder 的快照（Status=pending），
	// 发布以 ID+UserID 为匹配即可（支付落地即发布）。
	publishOrderPaidSSEFn = func(_ context.Context, order *ResourceOrder) {
		if order.ID == orderID && order.UserID == userID {
			events++
			published <- struct{}{}
		}
	}

	if err := markOrderPaid(context.Background(), orderID, "wechat", "mock_ref", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}
	select {
	case <-published:
	case <-time.After(2 * time.Second):
		t.Fatal("order_paid SSE not published after markOrderPaid")
	}
	if events != 1 {
		t.Errorf("published %d SSE events, want 1", events)
	}

	// 幂等重放（订单已 paid）：不再发布
	if err := markOrderPaid(context.Background(), orderID, "wechat", "mock_ref", tenantID); err != nil {
		t.Fatalf("markOrderPaid idempotent: %v", err)
	}
	select {
	case <-published:
		t.Errorf("idempotent path published an extra SSE event (total %d, want 1)", events)
	case <-time.After(500 * time.Millisecond):
		// 预期：无新发布
	}
	if events != 1 {
		t.Errorf("idempotent path published %d SSE events total, want 1", events)
	}
}
