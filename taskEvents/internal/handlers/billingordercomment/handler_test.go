package billingordercomment

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"taskEvents/domain"
)

func mustCmd(t *testing.T, data map[string]interface{}) domain.DomainCommand {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return domain.DomainCommand{
		EventType: "BILLING_ORDER_COMMENT_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "BILLING_ORDER_COMMENT_CREATED", Data: raw},
	}
}

func validData() map[string]interface{} {
	return map[string]interface{}{
		"comment_id":     "oc_1",
		"order_id":       "ord_100",
		"tenant_id":      "t1",
		"author_side":    "system_admin",
		"author_user_id": "admin-1",
		"created_at":     "2026-08-13T00:00:00Z",
	}
}

// Regression（OPT-20260811-084）：有效事件应通知对侧（SSE 发布被调用）并返回成功。
func TestDispatchValidPublishesToOrderSSE(t *testing.T) {
	h := &Handler{}
	var published string
	var called bool
	orig := publishOrderCommentSSEFn
	publishOrderCommentSSEFn = func(_ *Handler, _ context.Context, data map[string]interface{}, commentID, orderID, tenantID, authorSide string) error {
		called = true
		published = orderID
		_ = data
		return nil
	}
	defer func() { publishOrderCommentSSEFn = orig }()

	out, err := h.Dispatch(context.Background(), mustCmd(t, validData()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v want success", out)
	}
	if !called || published != "ord_100" {
		t.Fatalf("SSE publish not called with expected order; called=%v order=%s", called, published)
	}
}

// 回归：author_side=tenant 时同样通知（对侧为超管），发布照常。
func TestDispatchTenantSideStillPublishes(t *testing.T) {
	h := &Handler{}
	var called bool
	orig := publishOrderCommentSSEFn
	publishOrderCommentSSEFn = func(_ *Handler, _ context.Context, _ map[string]interface{}, _, _, _, _ string) error {
		called = true
		return nil
	}
	defer func() { publishOrderCommentSSEFn = orig }()

	data := validData()
	data["author_side"] = "tenant"
	out, err := h.Dispatch(context.Background(), mustCmd(t, data))
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("dispatch out=%v err=%v", out, err)
	}
	if !called {
		t.Fatal("SSE publish should be called for tenant-side comment")
	}
}

func TestDispatchUnsupportedEvent(t *testing.T) {
	h := &Handler{}
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{EventType: "OTHER_EVENT"})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want permanent", out)
	}
}

func TestDispatchMissingFields(t *testing.T) {
	h := &Handler{}
	data := validData()
	delete(data, "order_id")
	out, err := h.Dispatch(context.Background(), mustCmd(t, data))
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want permanent", out)
	}
}

func TestDispatchInvalidAuthorSide(t *testing.T) {
	h := &Handler{}
	data := validData()
	data["author_side"] = "bogus"
	out, err := h.Dispatch(context.Background(), mustCmd(t, data))
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want permanent", out)
	}
}

func TestDispatchSSEPublishErrorIsRetryable(t *testing.T) {
	h := &Handler{}
	orig := publishOrderCommentSSEFn
	publishOrderCommentSSEFn = func(_ *Handler, _ context.Context, _ map[string]interface{}, _, _, _, _ string) error {
		return context.DeadlineExceeded
	}
	defer func() { publishOrderCommentSSEFn = orig }()

	out, err := h.Dispatch(context.Background(), mustCmd(t, validData()))
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchRetryable {
		t.Fatalf("outcome %v want retryable", out)
	}
}

// 纯函数：payload 含订单级通道字段与 author_side，供「对侧」过滤。
func TestBuildOrderCommentSSEPayload(t *testing.T) {
	data := validData()
	body := buildOrderCommentSSEPayload(data, "oc_1", "ord_100", "t1", "system_admin")
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["task_id"] != "billing:order:ord_100" {
		t.Fatalf("task_id=%v", m["task_id"])
	}
	sd, ok := m["status_data"].(map[string]interface{})
	if !ok {
		t.Fatalf("status_data missing: %v", m)
	}
	if sd["event_name"] != "billing_order_comment_created" {
		t.Fatalf("event_name=%v", sd["event_name"])
	}
	if !strings.Contains(sd["message"].(string), "新评论") {
		t.Fatalf("message=%v", sd["message"])
	}
}
