package main

import (
	"authz"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedCommentOrder(t *testing.T, tenantID int64) int64 {
	t.Helper()
	oid := generateSnowflakeID()
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES (?, ?, ?, 'pending', 100, ?)`,
		oid, tenantID, "ORD-CMT-"+formatID(oid), utcNow(),
	)
	if err != nil {
		t.Fatalf("seed order: %v", err)
	}
	return oid
}

func TestValidateOrderCommentContent(t *testing.T) {
	if err := validateOrderCommentContent("  "); err != ErrOrderCommentEmpty {
		t.Fatalf("empty: %v", err)
	}
	long := strings.Repeat("字", maxOrderCommentRunes+1)
	if err := validateOrderCommentContent(long); err != ErrOrderCommentTooLong {
		t.Fatalf("too long: %v", err)
	}
	if err := validateOrderCommentContent("ok"); err != nil {
		t.Fatalf("ok: %v", err)
	}
}

func TestCreateAndListOrderComments(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9100000001)
	oid := seedCommentOrder(t, tenantID)

	c1, err := createOrderComment(context.Background(), tenantID, oid, "u-tenant", AuthorSideTenant, "需要发票")
	if err != nil {
		t.Fatalf("create tenant comment: %v", err)
	}
	c2, err := createOrderComment(context.Background(), tenantID, oid, "u-admin", AuthorSideSystemAdmin, "已处理")
	if err != nil {
		t.Fatalf("create admin comment: %v", err)
	}
	if c1.AuthorSide != AuthorSideTenant || c2.AuthorSide != AuthorSideSystemAdmin {
		t.Fatalf("author sides: %s %s", c1.AuthorSide, c2.AuthorSide)
	}

	list, err := listOrderComments(context.Background(), tenantID, oid, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 comments, got %d", len(list))
	}
	if list[0].Content != "需要发票" || list[1].Content != "已处理" {
		t.Fatalf("order/content mismatch: %+v", list)
	}

	_, err = createOrderComment(context.Background(), tenantID+1, oid, "u", AuthorSideTenant, "x")
	if err != ErrOrderCommentOrder {
		t.Fatalf("cross-tenant want ErrOrderCommentOrder, got %v", err)
	}
}

func TestHandleTenantOrderCommentsAuthAndCreate(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9100000002)
	oid := seedCommentOrder(t, tenantID)
	tid := formatID(tenantID)
	path := "/api/tenant/" + tid + "/billing/orders/" + formatID(oid) + "/comments/"

	// view-only cannot POST
	body := bytes.NewBufferString(`{"content":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authz.HeaderUserID, "u1")
	req.Header.Set(authz.HeaderTenantPerms, tid+":billing:view")
	w := httptest.NewRecorder()
	handleTenantOrderComments(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("view-only POST want 403, got %d body=%s", w.Code, w.Body.String())
	}

	// manage can POST
	body = bytes.NewBufferString(`{"content":"hello manage"}`)
	req = httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authz.HeaderUserID, "u1")
	req.Header.Set(authz.HeaderTenantPerms, tid+":billing:view,billing:manage")
	w = httptest.NewRecorder()
	handleTenantOrderComments(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("manage POST want 201, got %d body=%s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created["author_side"] != AuthorSideTenant {
		t.Fatalf("author_side=%v", created["author_side"])
	}

	// view can GET
	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(authz.HeaderUserID, "u1")
	req.Header.Set(authz.HeaderTenantPerms, tid+":billing:view")
	w = httptest.NewRecorder()
	handleTenantOrderComments(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET want 200, got %d", w.Code)
	}
	var listed map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if int(listed["total"].(float64)) != 1 {
		t.Fatalf("total=%v", listed["total"])
	}

	// empty content
	body = bytes.NewBufferString(`{"content":"  "}`)
	req = httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authz.HeaderUserID, "u1")
	req.Header.Set(authz.HeaderTenantPerms, tid+":billing:manage")
	w = httptest.NewRecorder()
	handleTenantOrderComments(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty content want 400, got %d", w.Code)
	}
}

func TestHandleSystemAdminOrderComments(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9100000003)
	oid := seedCommentOrder(t, tenantID)
	path := "/api/system_admin/orders/" + formatID(oid) + "/comments/"

	// non-staff
	body := bytes.NewBufferString(`{"tenant_id":"` + formatID(tenantID) + `","content":"admin note"}`)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authz.HeaderUserID, "staff1")
	req.Header.Set(authz.HeaderGatewayVerify, "1")
	w := httptest.NewRecorder()
	handleSystemAdminOrderComments(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-staff want 403, got %d", w.Code)
	}

	// staff
	body = bytes.NewBufferString(`{"tenant_id":"` + formatID(tenantID) + `","content":"admin note"}`)
	req = httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authz.HeaderUserID, "staff1")
	req.Header.Set(authz.HeaderGatewayVerify, "1")
	req.Header.Set(authz.HeaderUserRoles, "employee")
	w = httptest.NewRecorder()
	handleSystemAdminOrderComments(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("staff POST want 201, got %d body=%s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created["author_side"] != AuthorSideSystemAdmin {
		t.Fatalf("author_side=%v", created["author_side"])
	}

	req = httptest.NewRequest(http.MethodGet, path+"?tenant_id="+formatID(tenantID), nil)
	req.Header.Set(authz.HeaderUserID, "staff1")
	req.Header.Set(authz.HeaderGatewayVerify, "1")
	req.Header.Set(authz.HeaderUserRoles, "employee")
	w = httptest.NewRecorder()
	handleSystemAdminOrderComments(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("staff GET want 200, got %d", w.Code)
	}
}

func TestBillingOrderCommentEventTopicMapped(t *testing.T) {
	topic, ok := billingEventTopics["BILLING_ORDER_COMMENT_CREATED"]
	if !ok || topic != "billing-order-comment-created" {
		t.Fatalf("topic map: ok=%v topic=%q", ok, topic)
	}
}
