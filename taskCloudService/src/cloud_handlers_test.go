package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedCloudAuth(t *testing.T, companyID, authID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_platform_authorizations(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active) VALUES(?,?,?,?,?,?,?,1)`,
		authID, "aliyun", "access_key", "AKTEST1234", "SKTEST5678", "test", companyID,
	)
	if err != nil {
		t.Fatalf("seed auth: %v", err)
	}
}

func TestCloudAuthListReturnsArray(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/cloud/cloud-platform-authorizations/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, nil)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"authorizations"`) {
		t.Fatalf("expected plain array, got wrapped payload: %s", body)
	}
	if !strings.Contains(body, `"id":"auth1"`) {
		t.Fatalf("expected auth1 in list: %s", body)
	}
	if strings.Contains(body, "AKTEST1234") || strings.Contains(body, "SKTEST5678") {
		t.Fatalf("expected masked credentials, got full values: %s", body)
	}
	if !strings.Contains(body, "AKTE") || !strings.Contains(body, "1234") {
		t.Fatalf("expected access key prefix/suffix visible: %s", body)
	}
	if !strings.Contains(body, "SKTE") || !strings.Contains(body, "5678") {
		t.Fatalf("expected secret key prefix/suffix visible: %s", body)
	}
}

func TestMaskCredentialKeepsPrefixAndSuffix(t *testing.T) {
	got := maskCredential("AKTEST1234567890")
	want := "AKTE********7890"
	if got != want {
		t.Fatalf("maskCredential()=%q want=%q", got, want)
	}
	if maskCredential("12345678") != "********" {
		t.Fatalf("short value should be fully masked")
	}
}

func TestCloudAuthUpdateSkipsMaskedCredentials(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")

	payload := `{"secret_id":"AKTE****1234","secret_key":"SKTE****5678","remark":"unchanged"}`
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/cloud/cloud-platform-authorizations/auth1/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{"auth1"})

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var sid, sk, remark string
	if err := db.QueryRow("SELECT secret_id, secret_key, remark FROM cloud_platform_authorizations WHERE id=?", "auth1").Scan(&sid, &sk, &remark); err != nil {
		t.Fatalf("query: %v", err)
	}
	if sid != "AKTEST1234" || sk != "SKTEST5678" {
		t.Fatalf("masked update must not overwrite secrets: sid=%q sk=%q", sid, sk)
	}
	if remark != "unchanged" {
		t.Fatalf("remark should update: %q", remark)
	}
}

func TestCloudAuthDelete(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth-del")

	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/t1/cloud/cloud-platform-authorizations/auth-del/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{"auth-del"})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM cloud_platform_authorizations WHERE id=?", "auth-del").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected row deleted, count=%d", count)
	}
}

func TestCloudAuthDeleteMissingReturns404(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/tenant/t1/cloud/cloud-platform-authorizations/missing/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{"missing"})

	if rec.Code != 404 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCloudAuthVerifyCredentialsMock(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/cloud-platform-authorizations/auth1/verify-credentials/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{"auth1", "verify-credentials"})

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("expected success: %s", rec.Body.String())
	}
}

func TestGlobalCloudRegionsMock(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet, "/api/cloud/regions/?platform_type=aliyun", nil)
	req.Header.Set("Referer", "http://127.0.0.1:4000/tenant/850256677331562496/settings/cloud-platform/")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleCloudRegions(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"id":"cn-hangzhou"`) {
		t.Fatalf("expected region array: %s", body)
	}
	if strings.Contains(body, `"regions"`) {
		t.Fatalf("expected plain array not wrapped payload: %s", body)
	}
}
