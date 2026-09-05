package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"taskGitOauth/infrastructure"
	"testing"
)

func TestHandleRefreshMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-refresh/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleRefreshMissingToken(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-refresh/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing refresh_token, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleRefreshBridgeSecretRequired(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"refresh_token":"test-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-refresh/", bytes.NewBufferString(body))
	// No bridge secret header
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 without bridge secret, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ── handleTokenUseReport tests ──

func TestHandleTokenUseReportMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-token-use-report/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTokenUseReportInvalidJSON(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString("not json"))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid json, got %d", rec.Code)
	}
}

func TestHandleTokenUseReportMissingUserID(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","audit":{"action":"clone"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing user_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTokenUseReportMissingAuditAction(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":123,"audit":{"detail":"no action"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing audit.action, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTokenUseReportSuccess(t *testing.T) {
	app := testApp(t)
	seedAuditSiteProviders(app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":42,"audit":{"action":"clone","detail":{"repo":"test/repo"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got %v", out)
	}

	// Same for gitlab path
	body2 := `{"provider_key":"gitlab","user_id":42,"audit":{"action":"push"}}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/gitlab-token-use-report/", bytes.NewBufferString(body2))
	req2.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for gitlab path, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestHandleTokenUseReportBridgeSecretRejected(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":42,"audit":{"action":"clone"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-token-use-report/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 without bridge secret, got %d", rec.Code)
	}
}

// ── handleCredentialDelete tests ──

func TestHandleCredentialDeleteMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-credential-delete/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleCredentialDeleteMissingUserID(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-credential-delete/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing user_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleCredentialDeleteSuccess(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":99}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-credential-delete/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got %v", out)
	}
}

// ── handleCredentialUserIDs tests ──

func TestHandleCredentialUserIDsMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-credential-user-ids/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleCredentialUserIDsEmpty(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-credential-user-ids/?provider_key=github", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	ids, _ := out["user_ids"].([]any)
	if ids == nil {
		t.Fatalf("expected user_ids array, got %v", out)
	}
}

func TestHandleCredentialUserIDsGitLab(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/gitlab-credential-user-ids/?provider_key=gitlab", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ── handleTaskAuditReport tests ──

func TestHandleTaskAuditReportMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-audit-report/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskAuditReportMissingTaskID(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-audit-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing task_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskAuditReportMissingAction(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"task_id":"task-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-audit-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing action, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskAuditReportSuccess(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","task_id":"task-1","workspace_id":"ws-1","company_id":10,"user_id":42,"action":"task-clone","detail":{"repo":"owner/name"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-audit-report/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got %v", out)
	}

	// GitLab path
	body2 := `{"provider_key":"gitlab","task_id":"task-2","action":"gitlab-clone"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/gitlab-audit-report/", bytes.NewBufferString(body2))
	req2.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for gitlab path, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestHandleTaskAuditReportBridgeSecretRejected(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"task_id":"task-1","action":"clone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-audit-report/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 without bridge secret, got %d", rec.Code)
	}
}

// ── handleAccessForUser edge cases ──

func TestHandleAccessForUserMissingUserID(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-access-for-user/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing user_id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleAccessForUserMethodNotAllowed(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/git-oauth/github-access-for-user/", nil)
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 405 {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAccessForUserNotFound(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"provider_key":"github","user_id":999999}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-access-for-user/", bytes.NewBufferString(body))
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404 for non-existent user, got %d body=%s", rec.Code, rec.Body.String())
	}
	// OPT-20260821-037: 404 必须带可读 errDetail，供调用方/前端展示而非「未能换取」。
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["errDetail"] == "" || out["detail"] == "" {
		t.Fatalf("404 body=%s want detail+errDetail", rec.Body.String())
	}
}

// 多区域 GitLab 不得把默认实例 token 借给另一 host（trace 2a45093b：tencent-sh-1 仓 401）。
func TestHandleAccessForUserDoesNotReuseOtherGitLabInstanceToken(t *testing.T) {
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"gitlab": {
			{
				Provider: "gitlab", ServiceProvider: "daydaymoney-gitlab",
				ProviderKey: "gitlab:daydaymoney-gitlab",
				Website:     "https://gitlab.daydaymoney.com",
				ClientID:    "cid-default", RedirectURI: "https://daydaymoney.com/redirect/gitsite/gitlab.daydaymoney.com/oauth/callback/",
			},
			{
				Provider: "gitlab", ServiceProvider: "tencent-sh-1",
				ProviderKey: "gitlab:tencent-sh-1",
				Website:     "https://gitlab-tencent-sh-1.daydaymoney.com",
				ClientID:    "cid-sh1", RedirectURI: "https://daydaymoney.com/redirect/gitsite/gitlab-tencent-sh-1.daydaymoney.com/oauth/callback/",
			},
		},
	}
	_, err := app.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider:           "gitlab:daydaymoney-gitlab",
		Task2appUserID:     "877397583960502272",
		RefreshTokenCipher: "cipher-default-instance",
		RemoteUserID:       "2",
		RemoteLogin:        "example-user",
		Scope:              "api",
		BindStatus:         "active",
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/gitlab/oauth/access-for-user/",
		strings.NewReader(`{"user_id":877397583960502272,"provider_key":"gitlab:tencent-sh-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (must not reuse default GitLab token), got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRefreshAccessTokenUnknownTenantKeyStillErrors(t *testing.T) {
	app := testApp(t)
	_, err := app.refreshAccessToken("gitlab:tenant-missing-company", "rt")
	if err == nil {
		t.Fatal("expected missing provider config error")
	}
	if !strings.Contains(err.Error(), "未找到 provider_key=gitlab:tenant-missing-company") {
		t.Fatalf("err=%v", err)
	}
}
