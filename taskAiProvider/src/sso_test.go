package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

func TestSSOExchangeStaff(t *testing.T) {
	app := testApp(t)
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":      app.Cfg.SSOJwtIssuer,
		"aud":      app.Cfg.SSOAudience,
		"sub":      float64(900001),
		"typ":      "staff_bridge",
		"username": "go_mig_staff",
		"name":     "Go Mig Staff",
		"iat":      now,
		"exp":      now + 600,
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]any{"bridge": bridge})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["role"] != "staff" {
		t.Fatalf("role=%v", resp["role"])
	}
	access, _ := resp["access"].(string)
	if access == "" {
		t.Fatal("empty access")
	}
}

// 主站 sso_bridge_token.py 签发 sub=str(subject_id)；换票须接受字符串 sub（Snowflake 安全）。
func TestSSOExchangeStaffStringSub(t *testing.T) {
	app := testApp(t)
	now := time.Now().Unix()
	claims := map[string]any{
		"iss":      app.Cfg.SSOJwtIssuer,
		"aud":      app.Cfg.SSOAudience,
		"sub":      "850256677331562496",
		"typ":      "staff_bridge",
		"username": "go_str_sub_staff",
		"name":     "String Sub Staff",
		"iat":      now,
		"exp":      now + 600,
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]any{"bridge": bridge})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["role"] != "staff" {
		t.Fatalf("role=%v", resp["role"])
	}
	if resp["access"] == "" || resp["access"] == nil {
		t.Fatal("empty access")
	}
}

func TestSSOExchangeVendorStringSub(t *testing.T) {
	app := testApp(t)
	vendorID := infrastructure.NextID()
	saasUserID := infrastructure.NextID()
	email := "vendor_str_sub_" + infrastructure.IDStr(vendorID) + "@example.com"
	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at, saas_user_id)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		vendorID, email, "x", "VendorCo", "Contact", 1, nowStr, nowStr, saasUserID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})

	now := time.Now().Unix()
	claims := map[string]any{
		"iss":   app.Cfg.SSOJwtIssuer,
		"aud":   app.Cfg.SSOAudience,
		"sub":   infrastructure.IDStr(saasUserID),
		"typ":   "vendor_bridge",
		"email": email,
		"iat":   now,
		"exp":   now + 600,
	}
	bridge, err := infrastructure.SignHS256(app.Cfg.SSOJwtSecret, claims)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	body, _ := json.Marshal(map[string]any{"bridge": bridge})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["role"] != "vendor" {
		t.Fatalf("role=%v detail=%v", resp["role"], resp["detail"])
	}
	if resp["access"] == "" || resp["access"] == nil {
		t.Fatal("empty access")
	}
}

func TestSSOOnlyLogin(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/auth/login/", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Fatalf("status %d", rr.Code)
	}
}
