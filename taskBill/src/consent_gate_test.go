package main

import (
	"context"
	"testing"
)

func TestNormalizePaymentTermsKind(t *testing.T) {
	if got := normalizePaymentTermsKind("recharge_points"); got != DocumentKindPaymentTermsCanonical {
		t.Fatalf("alias normalize=%q want %q", got, DocumentKindPaymentTermsCanonical)
	}
	if got := normalizePaymentTermsKind("recharge_cents"); got != DocumentKindPaymentTermsCanonical {
		t.Fatalf("canonical=%q", got)
	}
	if isPaymentTermsKind("service") {
		t.Fatal("service must not be payment terms")
	}
	if !isPaymentTermsKind("recharge_points") {
		t.Fatal("recharge_points alias must count as payment terms")
	}
}

func TestCheckPaymentTermsConsentGate_NoActiveAgreementAllows(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	id, err := checkPaymentTermsConsentGate(context.Background(), "user-1", "")
	if err != nil {
		t.Fatalf("no active agreement should allow: %v", err)
	}
	if id != "" {
		t.Fatalf("resolved consent_id=%q want empty", id)
	}
}

func TestCheckPaymentTermsConsentGate_RequiresConsentWhenActive(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	agreementID := formatID(generateSnowflakeID())
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_license_agreements(id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at)
		VALUES(?,?,?,?,?,1,0,?,?)`,
		agreementID, "支付服务条款", "条款正文", "0.9", DocumentKindPaymentTermsCanonical, now, now,
	); err != nil {
		t.Fatalf("seed agreement: %v", err)
	}

	_, err := checkPaymentTermsConsentGate(context.Background(), "user-pay-1", "")
	if err == nil {
		t.Fatal("expected consent required")
	}
	ce, ok := err.(*consentGateError)
	if !ok || ce.ReasonCode != reasonConsentRequired {
		t.Fatalf("err=%v want reason %s", err, reasonConsentRequired)
	}

	consentID := formatID(generateSnowflakeID())
	if _, err := db.Exec(`
		INSERT INTO billing_user_license_agreement_consents(id,user_id,license_agreement_id,consented_at,context,client_ip,user_agent)
		VALUES(?,?,?,?,?,?,?)`,
		consentID, "user-pay-1", agreementID, now, "order_pay", "", "",
	); err != nil {
		t.Fatalf("seed consent: %v", err)
	}

	got, err := checkPaymentTermsConsentGate(context.Background(), "user-pay-1", "")
	if err != nil {
		t.Fatalf("with consent should allow: %v", err)
	}
	if got != consentID {
		t.Fatalf("consent_id=%q want %q", got, consentID)
	}
}

func TestCheckPaymentTermsConsentGate_ExplicitConsentID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	agreementID := formatID(generateSnowflakeID())
	consentID := formatID(generateSnowflakeID())
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_license_agreements(id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at)
		VALUES(?,?,?,?,?,1,0,?,?)`,
		agreementID, "支付服务条款", "条款正文", "0.9", DocumentKindPaymentTermsCanonical, now, now,
	); err != nil {
		t.Fatalf("seed agreement: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_user_license_agreement_consents(id,user_id,license_agreement_id,consented_at,context,client_ip,user_agent)
		VALUES(?,?,?,?,?,?,?)`,
		consentID, "user-pay-2", agreementID, now, "order_pay", "", "",
	); err != nil {
		t.Fatalf("seed consent: %v", err)
	}

	got, err := checkPaymentTermsConsentGate(context.Background(), "user-pay-2", consentID)
	if err != nil {
		t.Fatalf("explicit consent: %v", err)
	}
	if got != consentID {
		t.Fatalf("got=%q want %q", got, consentID)
	}

	_, err = checkPaymentTermsConsentGate(context.Background(), "user-other", consentID)
	if err == nil {
		t.Fatal("consent belonging to another user must fail")
	}
}

func TestEnrichRechargesWithConsent_AdminGrantNotRequired(t *testing.T) {
	recharges := []map[string]interface{}{
		{"points_source_type": "admin_grant", "amount_points": int64(0), "channel": ""},
		{"points_source_type": "user_recharge_wechat", "amount_points": int64(100), "channel": "wechat"},
	}
	consents := []map[string]interface{}{
		{
			"id":                "c1",
			"agreement_version": "0.9",
			"document_kind":     DocumentKindPaymentTermsCanonical,
			"consented_at":      "2026-08-18T00:00:00Z",
			"content_snapshot":  "条款",
		},
	}
	out := enrichRechargesWithConsent(recharges, consents)
	if out[0]["consent_required"] != false {
		t.Fatalf("admin_grant consent_required=%v want false", out[0]["consent_required"])
	}
	if out[0]["consent_note"] != "系统赠送，无需支付签署" {
		t.Fatalf("admin_grant note=%v", out[0]["consent_note"])
	}
	if out[0]["consent"] != nil {
		t.Fatal("admin_grant should not attach payment consent")
	}
	if out[1]["consent"] == nil {
		t.Fatal("user payment should attach payment-terms consent when available")
	}
	c := out[1]["consent"].(map[string]interface{})
	if c["id"] != "c1" {
		t.Fatalf("attached consent id=%v", c["id"])
	}
}

// OPT-20260825-017 回归：两笔支付挂不同订单 consent_id 时互不覆盖；
// 无订单 consent 的支付行回退最新支付条款（consent_source=fallback）。
func TestEnrichRechargesWithConsent_OrderScopedConsentWins(t *testing.T) {
	recharges := []map[string]interface{}{
		{"points_source_type": "user_recharge_wechat", "amount_points": int64(100), "channel": "wechat", "order_consent_id": "c2"},
		{"points_source_type": "user_recharge_wechat", "amount_points": int64(100), "channel": "wechat", "order_consent_id": "c3"},
		{"points_source_type": "user_recharge_wechat", "amount_points": int64(100), "channel": "wechat", "order_consent_id": ""},
	}
	consents := []map[string]interface{}{
		{"id": "c3", "agreement_version": "1.0", "document_kind": DocumentKindPaymentTermsCanonical, "consented_at": "2026-08-20T00:00:00Z", "content_snapshot": "v1"},
		{"id": "c2", "agreement_version": "0.9", "document_kind": DocumentKindPaymentTermsCanonical, "consented_at": "2026-08-18T00:00:00Z", "content_snapshot": "v0.9"},
	}
	out := enrichRechargesWithConsent(recharges, consents)

	if len(out) != 3 {
		t.Fatalf("len=%d want 3", len(out))
	}
	// 两笔支付各挂各自订单签署的 consent，互不覆盖。
	for i, wantID := range []string{"c2", "c3"} {
		row := out[i]
		if row["consent_source"] != "order" {
			t.Fatalf("row[%d] source=%v want order", i, row["consent_source"])
		}
		c, ok := row["consent"].(map[string]interface{})
		if !ok || c["id"] != wantID {
			t.Fatalf("row[%d] consent id=%v want %s", i, c["id"], wantID)
		}
	}
	// 无订单 consent 的支付行回退最新条款（consents 中最新 c3）。
	fallback := out[2]
	if fallback["consent_source"] != "fallback" {
		t.Fatalf("fallback source=%v want fallback", fallback["consent_source"])
	}
	if c := fallback["consent"].(map[string]interface{}); c["id"] != "c3" {
		t.Fatalf("fallback consent id=%v want c3", c["id"])
	}
}

func TestAttachConsentToOrder(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(9001)
	orderID := generateSnowflakeID()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES (?, ?, ?, 'pending', 55, ?)`,
		orderID, tenantID, "ORD-CONSENT-1", now,
	); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	consentID := formatID(generateSnowflakeID())
	if err := attachConsentToOrder(context.Background(), orderID, consentID); err != nil {
		t.Fatalf("attach: %v", err)
	}
	var got string
	if err := db.QueryRow(`SELECT consent_id FROM billing_resource_order WHERE id = ?`, orderID).Scan(&got); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if got != consentID {
		t.Fatalf("consent_id=%q want %q", got, consentID)
	}
}
