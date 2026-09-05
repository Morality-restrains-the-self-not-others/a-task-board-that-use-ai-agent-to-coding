package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskGitOauth/infrastructure"
)

func testApp(t *testing.T) *App {
	t.Helper()
	db := setupMySQLTestDB(t)

	// Create tables (MySQL DDL)
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS git_oauth_appusercredential (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  provider varchar(64) NOT NULL,
  task2app_user_id varchar(36) NOT NULL,
  refresh_token_cipher text NOT NULL,
  remote_user_id varchar(32) NOT NULL,
  remote_login varchar(255) NOT NULL DEFAULT '',
  scope varchar(512) NOT NULL DEFAULT '',
  bind_status varchar(32) NOT NULL DEFAULT 'active',
  bind_error varchar(512) NOT NULL DEFAULT '',
  created_at datetime NOT NULL,
  updated_at datetime NOT NULL,
  UNIQUE(provider, task2app_user_id, remote_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS git_oauth_appaccesstokenuseaudit (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  site varchar(255) NOT NULL,
  task2app_user_id varchar(36) NOT NULL,
  company_id bigint NULL,
  workspace_id bigint NULL,
  action varchar(64) NOT NULL,
  access_token_fingerprint varchar(64) NOT NULL DEFAULT '',
  detail text NOT NULL,
  created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS git_oauth_taskcredentialaudit (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  provider varchar(64) NOT NULL,
  task2app_task_id varchar(64) NOT NULL,
  task2app_workspace_id varchar(64) NOT NULL DEFAULT '',
  task2app_company_id bigint NULL,
  task2app_user_id varchar(36) NULL,
  action varchar(64) NOT NULL,
  detail text NOT NULL,
  created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS git_oauth_grant_ticket (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  task2app_user_id VARCHAR(36) NOT NULL,
  gitsite VARCHAR(255) NOT NULL,
  remote_user_id VARCHAR(64) NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  consumed_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &infrastructure.Config{
		SecretKey:           infrastructure.DefaultSecretKey,
		Port:                8002,
		Host:                "127.0.0.1",
		DatabasePath:        "",
		PublicBaseURL:       "http://localhost:8002",
		BridgeJWTSecret:     "test-secret",
		BridgeJWTIssuer:     "task2app",
		BridgeJWTAudienceGH: "gitOauth-github-oauth",
		BridgeJWTAudienceGL: "gitOauth-gitlab-oauth",
		Task2appInternalAPI: "http://127.0.0.1:8001",
		Providers:           map[string][]infrastructure.ProviderConfig{},
		Debug:               true,
		RequireBridgeSecret: true,
		TenantAuthBypass:    true,
	}
	// Also create the tenant connections table
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS git_oauth_tenant_gitlab_oauth_connections (
  id VARCHAR(64) PRIMARY KEY,
  company_id VARCHAR(255) UNIQUE NOT NULL,
  base_url TEXT NOT NULL,
  client_id TEXT NOT NULL,
  client_secret_enc TEXT NOT NULL,
  remark TEXT NOT NULL,
  redirect_uri TEXT NOT NULL,
  scope VARCHAR(512) NOT NULL DEFAULT 'read_repository write_repository api read_user',
  active INTEGER NOT NULL DEFAULT 1,
  intranet INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatal(err)
	}
	return NewApp(cfg, &infrastructure.DB{DB: db})
}

func TestHealthHandler(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["service"] != "gitOauth" {
		t.Fatalf("service=%v", body["service"])
	}
	if body["ok"] != true {
		t.Fatalf("ok=%v", body["ok"])
	}
}

func TestAccessForUserNotFound(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-access-for-user/",
		strings.NewReader(`{"user_id": 999999}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["detail"] != "not_found" {
		t.Fatalf("detail=%v", body["detail"])
	}
}

func TestProviderKeyAliasAccessForUser(t *testing.T) {
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {{
			Provider: "github", ServiceProvider: "github-official",
			ProviderKey: "github:github-official",
			ClientID:    "cid", ClientSecret: "sec",
		}},
	}
	f := infrastructure.NewFernet(infrastructure.DefaultSecretKey)
	cipher, err := f.Encrypt("ghu_test_pat_token")
	if err != nil {
		t.Fatal(err)
	}
	// legacy bare provider row must be found via compound key
	_, err = app.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider:           "github",
		Task2appUserID:     "42",
		RefreshTokenCipher: cipher,
		RemoteUserID:       "99",
		RemoteLogin:        "octocat",
		Scope:              "repo",
		BindStatus:         "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-access-for-user/",
		strings.NewReader(`{"user_id":42,"provider_key":"github:github-official"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["access_token"] != "ghu_test_pat_token" {
		t.Fatalf("access_token=%v", body["access_token"])
	}
	if body["github_user_id"] != "99" {
		t.Fatalf("github_user_id=%v", body["github_user_id"])
	}
}

func TestGithubStoredProviderKey(t *testing.T) {
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {{Provider: "github", ServiceProvider: "github-official", ProviderKey: "github:github-official"}},
	}
	if got := app.githubStoredProviderKey(); got != "github:github-official" {
		t.Fatalf("got %q", got)
	}
}

// TestGithubStoredProviderKeysMultiConfig — 多 github provider 配置时状态检查必须覆盖全部
// 存储键（OPT-20260809-009）：单键接口取 rows[0] 权威键，多键接口展开所有行并去重。
func TestGithubStoredProviderKeysMultiConfig(t *testing.T) {
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {
			{Provider: "github", ServiceProvider: "github-official", ProviderKey: "github:github-official"},
			{Provider: "github", ServiceProvider: "daydaymoney", ProviderKey: "github:daydaymoney"},
		},
	}
	got := app.githubStoredProviderKeys()
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %v", got)
	}
	if got[0] != "github:github-official" || got[1] != "github:daydaymoney" {
		t.Fatalf("got %v", got)
	}

	// 行缺 ProviderKey 时退化为 composite github:<sp>，且去重
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {
			{Provider: "github", ServiceProvider: "github-official"},
			{Provider: "github", ServiceProvider: "github-official"},
			{Provider: "github", ServiceProvider: "daydaymoney", ProviderKey: "github:daydaymoney"},
		},
	}
	got = app.githubStoredProviderKeys()
	want := []string{"github:github-official", "github:daydaymoney"}
	if len(got) != len(want) {
		t.Fatalf("dedup failed: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSummaryProviderKeyAlias(t *testing.T) {
	app := testApp(t)
	f := infrastructure.NewFernet(infrastructure.DefaultSecretKey)
	cipher, _ := f.Encrypt("ghu_x")
	_, err := app.DB.UpsertCredential(&infrastructure.CredentialRow{
		Provider: "github", Task2appUserID: "7", RefreshTokenCipher: cipher,
		RemoteUserID: "1", RemoteLogin: "u", BindStatus: "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-credential-summary/",
		strings.NewReader(`{"user_id":7,"provider_key":"github:github-official"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitOauth-Bridge-Secret", "test-secret")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["connected"] != true {
		t.Fatalf("connected=%v body=%v", body["connected"], body)
	}
}

func TestBridgeSecretRequired(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-oauth/github-access-for-user/",
		strings.NewReader(`{"user_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401 body %s", rr.Code, rr.Body.String())
	}
}

func TestSwaggerHasComponents(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/swagger/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	comps, ok := body["components"].(map[string]any)
	if !ok || comps["schemas"] == nil {
		t.Fatalf("missing components.schemas keys=%v", keysOf(body))
	}
	paths, _ := body["paths"].(map[string]any)
	af, _ := paths["/api/internal/git-oauth/github-access-for-user/"].(map[string]any)
	post, _ := af["post"].(map[string]any)
	if post["requestBody"] == nil {
		t.Fatal("access-for-user missing requestBody")
	}
	if post["responses"] == nil {
		t.Fatal("access-for-user missing responses")
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
