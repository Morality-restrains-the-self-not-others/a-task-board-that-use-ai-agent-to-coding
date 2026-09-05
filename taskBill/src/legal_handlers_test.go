package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegalTablesCreatedByMigration(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tables := []string{"billing_license_agreements", "billing_user_license_agreement_consents", "billing_privacy_policies", "billing_user_privacy_policy_consents"}
	for _, tbl := range tables {
		var count int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", tbl,
		).Scan(&count); err != nil {
			t.Fatalf("check table %s: %v", tbl, err)
		}
		if count != 1 {
			t.Fatalf("table %s not found", tbl)
		}
	}
}

func TestLicenseAgreementCRUD(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mountRoutes(mux)

	// Create
	body := `{"title":"服务协议","content":"请遵守以下条款","version":"1.0","document_kind":"service","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/system_admin/license-agreement/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/system_admin/license-agreement/", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr2.Code)
	}

	// Dash alias: FE 统一使用连字符形态，须路由到列表而非 detail（否则 404 服务协议不存在）
	reqDash := httptest.NewRequest(http.MethodGet, "/api/system-admin/license-agreement/?kind=service", nil)
	rrDash := httptest.NewRecorder()
	mux.ServeHTTP(rrDash, reqDash)
	if rrDash.Code != http.StatusOK {
		t.Fatalf("dash alias list status=%d body=%s", rrDash.Code, rrDash.Body.String())
	}
}

func TestPrivacyPolicyCRUD(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mountRoutes(mux)

	// Create
	body := `{"title":"隐私条款","content":"隐私保护条款内容","version":"1.0","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/system_admin/privacy-policy/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/system_admin/privacy-policy/", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr2.Code)
	}
}

func TestPublicCurrentEndpoints(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	// Seed an active agreement and policy
	now := utcNow()
	db.Exec(`INSERT INTO billing_license_agreements(id,title,content,version,document_kind,is_active,created_at,updated_at) VALUES(?,?,?,?,?,1,?,?)`,
		"la-001", "服务协议", "内容", "v1", "service", now, now)
	db.Exec(`INSERT INTO billing_privacy_policies(id,title,content,version,is_active,created_at,updated_at) VALUES(?,?,?,?,1,?,?)`,
		"pp-001", "隐私条款", "内容", "v1", now, now)

	mux := http.NewServeMux()
	mountRoutes(mux)

	// Public license agreement
	req := httptest.NewRequest(http.MethodGet, "/api/license-agreement/public/current/?kind=service", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("public license status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Public privacy policy
	req2 := httptest.NewRequest(http.MethodGet, "/api/privacy-policy/public/current/", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("public privacy status=%d body=%s", rr2.Code, rr.Body.String())
	}

	// Not found when no active
	db.Exec(`UPDATE billing_license_agreements SET is_active=0`)
	req3 := httptest.NewRequest(http.MethodGet, "/api/license-agreement/public/current/?kind=service", nil)
	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNotFound {
		t.Fatalf("expect 404 for no active, got %d", rr3.Code)
	}
}
