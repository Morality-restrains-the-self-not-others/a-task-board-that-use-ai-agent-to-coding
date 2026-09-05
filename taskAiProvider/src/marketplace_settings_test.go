package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSOMarketplaceSettingsDefaultEnabled(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/ai-provider/marketplace-settings/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["vendor_application_review_enabled"] != true {
		t.Fatalf("expected review enabled by default, got %v", body)
	}
}

func TestSSOAdminMarketplaceSettingsPatchByPlatformStaff(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/ai-provider/admin-marketplace-settings/",
		strings.NewReader(`{"vendor_application_review_enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["vendor_application_review_enabled"] != false {
		t.Fatalf("expected false after patch, got %v", body)
	}
	if app.DB.IsVendorApplicationReviewEnabled() {
		t.Fatal("DB still reports review enabled")
	}
}

func TestSSOAdminMarketplaceSettingsForbiddenWithoutRole(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/ai-provider/admin-marketplace-settings/",
		strings.NewReader(`{"vendor_application_review_enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without staff/platform role, got %d", rec.Code)
	}
}

func TestSSOVendorStatusIncludesReviewFlag(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	if err := app.DB.SetVendorApplicationReviewEnabled(false); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	rec := vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "90001", "X-User-Email": "a@example.com"})
	body := decodeVendorStatus(t, rec)
	if body["vendor_application_review_enabled"] != false {
		t.Fatalf("expected review flag false in vendor-status, got %v", body)
	}
}

func TestSSOExchangeAutoCreateWhenReviewDisabled(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	if err := app.DB.SetVendorApplicationReviewEnabled(false); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "71001", "vendor_bridge",
		map[string]any{"email": "auto-provision@example.com"}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/",
		strings.NewReader(`{"bridge":"`+bridge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 auto-provision when review off, got %d body=%s", rec.Code, rec.Body.String())
	}
	var active int
	if err := app.DB.SQL.QueryRow(`SELECT is_active FROM ai_provider_vendor WHERE saas_user_id=71001`).Scan(&active); err != nil || active != 1 {
		t.Fatalf("expected active vendor auto-created, active=%d err=%v", active, err)
	}
}

func TestSSOExchangeActivatePendingWhenReviewDisabled(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(71002), "pending-activate@example.com", false)
	if err := app.DB.SetVendorApplicationReviewEnabled(false); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "71002", "vendor_bridge",
		map[string]any{"email": "pending-activate@example.com"}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/",
		strings.NewReader(`{"bridge":"`+bridge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 activate pending when review off, got %d body=%s", rec.Code, rec.Body.String())
	}
}
