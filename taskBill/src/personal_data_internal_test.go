package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func personalDataReq(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-TaskBill-Internal-Secret", "test-secret")
	return req
}

func TestInternalUserPersonalDataRequiresSecret(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/users/123/personal-data/", nil)
	w := httptest.NewRecorder()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	handleInternalUserPersonalData(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", w.Code)
	}
}

func TestInternalUserPersonalDataBadPath(t *testing.T) {
	req := personalDataReq(http.MethodGet, "/api/internal/taskbill/users/123/other/")
	w := httptest.NewRecorder()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	handleInternalUserPersonalData(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func TestInternalUserPersonalDataCollectsSections(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	userID := fmt.Sprintf("%d", generateSnowflakeID())
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()

	// 订单（user_id 归属本人）
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, user_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at
		) VALUES (?, ?, ?, ?, 'paid', 1000, 'wechat', '', ?)`,
		orderID, tenantID, userID, fmt.Sprintf("ORD-PD-%d", orderID), utcNow())
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}
	// 他人订单不应出现
	otherID := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, user_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at
		) VALUES (?, ?, 999999999, ?, 'pending', 100, 'wechat', '', ?)`,
		otherID, tenantID, fmt.Sprintf("ORD-OTHER-%d", otherID), utcNow())
	if err != nil {
		t.Fatalf("insert other order: %v", err)
	}

	// 发票（关联本人订单）
	_, err = db.Exec(`
		INSERT INTO billing_invoice (
			id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose,
			status, amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number,
			buyer_snapshot, fail_reason, created_at, updated_at
		) VALUES (?, ?, ?, 0, NULL, 'normal', 'order', 'issued', 1000, 'FP-PD-001', '', '', '{}', '', ?, ?)`,
		generateSnowflakeID(), tenantID, orderID, utcNow(), utcNow())
	if err != nil {
		t.Fatalf("insert invoice: %v", err)
	}

	// 退款申请（本人提交）
	_, err = db.Exec(`
		INSERT INTO billing_refund_application (
			tenant_id, account_id, applicant_user_id, frozen_points, status,
			reason, reviewer_user_id, review_note, payment_refund_refs, order_id, created_at, updated_at
		) VALUES (?, 1, ?, 500, 'pending', 'test', '', '', '[]', ?, ?, ?)`,
		tenantID, userID, orderID, utcNow(), utcNow())
	if err != nil {
		t.Fatalf("insert refund: %v", err)
	}

	// 协议签署记录
	_, err = db.Exec(`INSERT INTO billing_license_agreements (id, title, content, version, document_kind, is_active, is_material_change) VALUES ('la-1', '服务协议', 'x', 'v1', 'service', 1, 0)`)
	if err != nil {
		t.Fatalf("insert license: %v", err)
	}
	_, err = db.Exec(`INSERT INTO billing_user_license_agreement_consents (id, user_id, license_agreement_id, consented_at, context) VALUES (?, ?, 'la-1', ?, 'signup')`,
		fmt.Sprintf("ulac-%d", generateSnowflakeID()), userID, utcNow())
	if err != nil {
		t.Fatalf("insert license consent: %v", err)
	}

	w := httptest.NewRecorder()
	handleInternalUserPersonalData(w, personalDataReq(http.MethodGet, "/api/internal/taskbill/users/"+userID+"/personal-data/"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var wrapper struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data := wrapper.Data

	for _, key := range []string{"orders", "invoices", "refund_applications", "pending_payments",
		"license_agreement_consents", "privacy_policy_consents", "transactions", "referral_edge"} {
		arr, ok := data[key].([]interface{})
		if !ok {
			t.Fatalf("missing section %q in %v", key, data)
		}
		if arr == nil {
			t.Fatalf("section %q is nil", key)
		}
	}

	orders, _ := data["orders"].([]interface{})
	if len(orders) != 1 {
		t.Fatalf("orders=%d want 1 (other user's order must not leak)", len(orders))
	}
	inv, _ := data["invoices"].([]interface{})
	if len(inv) != 1 {
		t.Fatalf("invoices=%d want 1", len(inv))
	}
	refunds, _ := data["refund_applications"].([]interface{})
	if len(refunds) != 1 {
		t.Fatalf("refunds=%d want 1", len(refunds))
	}
	licConsents, _ := data["license_agreement_consents"].([]interface{})
	if len(licConsents) != 1 {
		t.Fatalf("license consents=%d want 1", len(licConsents))
	}
	// 凭证不导出：支付凭据列不在结果中
	raw := w.Body.String()
	for _, forbidden := range []string{"payment_ref", "buyer_snapshot", "pay_openid", "pay_unionid", "verification_secret", "client_token"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("export leaks credential field %q", forbidden)
		}
	}
}

func TestInternalUserPersonalDataEmptyUser(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	w := httptest.NewRecorder()
	handleInternalUserPersonalData(w, personalDataReq(http.MethodGet, "/api/internal/taskbill/users/987654321/personal-data/"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", w.Code)
	}
	var wrapper struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"orders", "invoices", "refund_applications", "pending_payments",
		"license_agreement_consents", "privacy_policy_consents", "transactions", "referral_edge"} {
		arr, _ := wrapper.Data[key].([]interface{})
		if len(arr) != 0 {
			t.Fatalf("section %q for empty user: %v", key, arr)
		}
	}
}
