package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedOAuthToken(t *testing.T, companyID, tokenID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO cloud_oauth_tokens(id,authorization_id,access_token,refresh_token,company_id,platform_type,authorization_type,scope)
		 VALUES(?,?,?,?,?,?,?,?)`,
		tokenID, tokenID, "access-secret", "refresh-secret", companyID, "aliyun", "oauth", "ecs",
	)
	if err != nil {
		t.Fatalf("seed oauth: %v", err)
	}
}

func TestOAuthTokenList(t *testing.T) {
	setupCloudTestDB(t)
	seedOAuthToken(t, "t1", "oat1")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/cloud/oauth-tokens/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleOAuthTokens(rec, req, nil)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"id":"oat1"`) {
		t.Fatalf("expected oat1: %s", body)
	}
	if strings.Contains(body, "access-secret") {
		t.Fatalf("must not expose access_token: %s", body)
	}
}

func TestOAuthTokenCreate(t *testing.T) {
	setupCloudTestDB(t)

	body := `{"platform_type":"aliyun","authorization_type":"oauth","access_token":"tok","scope":"ecs"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/oauth-tokens/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleOAuthTokens(rec, req, nil)

	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"platform_type":"aliyun"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestOAuthTokenCreateHonorsIsActiveTrue(t *testing.T) {
	setupCloudTestDB(t)

	body := `{"platform_type":"aliyun","authorization_type":"oauth","access_token":"tok","scope":"ecs","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/oauth-tokens/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user1")
	rec := httptest.NewRecorder()
	handleOAuthTokens(rec, req, nil)
	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("missing id: %s", rec.Body.String())
	}
	if !authIsActive("t1", id, "oauth") {
		t.Fatalf("oauth create with is_active:true must persist as active, id=%s", id)
	}
}

func TestInternalOAuthTokenLookup(t *testing.T) {
	setupCloudTestDB(t)
	seedOAuthToken(t, "t1", "oat1")

	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud/oauth-token/lookup/?id=oat1&company_id=t1", nil)
	req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	rec := httptest.NewRecorder()
	handleInternalOAuthTokenLookup(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"platform_type":"aliyun"`) {
		t.Fatalf("unexpected: %s", rec.Body.String())
	}
}

func TestResolveToggleAuthTargetOAuth(t *testing.T) {
	setupCloudTestDB(t)
	seedOAuthToken(t, "t1", "oat1")

	pt, method, err := resolveToggleAuthTarget("t1", "oat1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if pt != "aliyun" || method != "oauth" {
		t.Fatalf("pt=%s method=%s", pt, method)
	}
}
