package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func staffParseOrderNumberRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, "/api/system-admin/order-number/parse/", nil)
	} else {
		r = httptest.NewRequest(method, "/api/system-admin/order-number/parse/", bytes.NewBufferString(body))
	}
	r.Header.Set("X-Gateway-Auth-Verified", "1")
	r.Header.Set("X-User-Roles", "super_admin")
	return r
}

// OPT-20260819-012：管理端粘贴四段订单号应解析出租户与主键，供前端跳转租户订单页。
func TestHandleSystemAdminParseOrderNumberFourSegment(t *testing.T) {
	req := staffParseOrderNumberRequest(t, http.MethodPost, `{"order_number":"ORD-20260820-1234567890-987654321"}`)
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", rr.Code, rr.Body.String())
	}
	var d struct {
		OrderNumber string `json:"order_number"`
		HasTenant   bool   `json:"has_tenant"`
		TenantID    string `json:"tenant_id"`
		OrderID     string `json:"order_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &d); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rr.Body.String())
	}
	if !d.HasTenant || d.TenantID != "1234567890" || d.OrderID != "987654321" {
		t.Fatalf("four-segment parse wrong: %+v", d)
	}
}

func TestHandleSystemAdminParseOrderNumberThreeSegmentNoTenant(t *testing.T) {
	req := staffParseOrderNumberRequest(t, http.MethodPost, `{"order_number":"ORD-20260820-555"}`)
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", rr.Code, rr.Body.String())
	}
	var d struct {
		HasTenant bool   `json:"has_tenant"`
		OrderID   string `json:"order_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.HasTenant || d.OrderID != "555" {
		t.Fatalf("three-segment parse wrong: %+v", d)
	}
}

func TestHandleSystemAdminParseOrderNumberInvalid(t *testing.T) {
	req := staffParseOrderNumberRequest(t, http.MethodPost, `{"order_number":"ORD-123-abc"}`)
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSystemAdminParseOrderNumberAuthRequired(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/order-number/parse/",
		bytes.NewBufferString(`{"order_number":"ORD-20260820-1-2"}`))
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSystemAdminParseOrderNumberStaffForbidden(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/order-number/parse/",
		bytes.NewBufferString(`{"order_number":"ORD-20260820-1-2"}`))
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSystemAdminParseOrderNumberMethodNotAllowed(t *testing.T) {
	req := staffParseOrderNumberRequest(t, http.MethodGet, "")
	rr := httptest.NewRecorder()
	handleSystemAdminParseOrderNumber(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405 body=%s", rr.Code, rr.Body.String())
	}
}
