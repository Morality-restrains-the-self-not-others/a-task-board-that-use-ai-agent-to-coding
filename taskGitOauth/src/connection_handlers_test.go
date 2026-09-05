package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskGitOauth/infrastructure"
)

// providerTestApp adds the same provider configs the dev YAML declares,
// so service_provider / repo_url resolution works in tests.
func providerTestApp(t *testing.T) *App {
	t.Helper()
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {
			{
				Provider:        "github",
				ServiceProvider: "github-official-daydaymoney",
				ProviderKey:     "github:github-official-daydaymoney",
				Website:         "http://github.com",
			},
		},
		"gitlab": {
			{
				Provider:        "gitlab",
				ServiceProvider: "gitlab-local",
				ProviderKey:     "gitlab:gitlab-local",
				Website:         "http://localhost:8012",
			},
		},
	}
	return app
}

func TestHandleUserAppConnectionGetConventionPath(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?service_provider=github-official-daydaymoney", nil)
	req.Header.Set("X-User-Id", "873093522473906176")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != false {
		t.Fatalf("connected=%v", body["connected"])
	}
	if body["connections"] == nil {
		t.Fatalf("connections missing: %v", body)
	}
}

func TestHandleUserAppConnectionGetProviderQuery(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?provider=gitlab&service_provider=gitlab-local", nil)
	req.Header.Set("X-User-Id", "827923618451263488")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != false {
		t.Fatalf("connected=%v", body["connected"])
	}
}

func TestHandleUserAppConnectionGetRepoURLResolution(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo", nil)
	req.Header.Set("X-User-Id", "873093522473906176")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleUserAppConnectionMissingUserID(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?service_provider=github-official-daydaymoney", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "detail") {
		t.Fatalf("expected JSON detail, got %s", rec.Body.String())
	}
}

// TestHandleUserAppConnectionGetRepoURLOnlyFindsConfiguredKey — 回归缺陷：
// 任务详情页状态检查只传 repo_url（无 provider/service_provider），而凭据按
// 配置的 service_provider 键存储（github:github-official-daydaymoney）。
// 此前检查键固定为 github:default → 查不到 → connected=false → 已授权仓库
// 误显 "OAuth 绑定" 按钮。
func TestHandleUserAppConnectionGetRepoURLOnlyFindsConfiguredKey(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "873093522473906176"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-1', 'octocat', 'repo read:user',
        'active', '', NOW(), NOW())`,
		"github:github-official-daydaymoney", userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgithub.com%2Forg%2Frepo", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != true {
		t.Fatalf("repo_url-only check must find credential under configured provider key; connected=%v body=%s",
			body["connected"], rec.Body.String())
	}
}

// TestHandleUserAppConnectionGetRepoURLOnlyFindsConfiguredGitlabKey — 同上的
// GitLab 变体：凭据存于 gitlab:gitlab-local，检查仅传自托管仓库 repo_url。
func TestHandleUserAppConnectionGetRepoURLOnlyFindsConfiguredGitlabKey(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "827923618451263488"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-gl-1', 'root', 'read_repository api read_user',
        'active', '', NOW(), NOW())`,
		"gitlab:gitlab-local", userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != true {
		t.Fatalf("repo_url-only check must find gitlab credential under configured provider key; connected=%v body=%s",
			body["connected"], rec.Body.String())
	}
}

// TestHandleUserAppConnectionRepoURLDoesNotTreatOtherGitLabInstanceAsConnected
// 回归：用户只绑了 gitlab:tencent-sh-1 时，查询另一 GitLab host 不得 connected=true
// （此前 gitlab 分支展开全部配置键，创建任务 OAuth 门禁误放行）。
func TestHandleUserAppConnectionRepoURLDoesNotTreatOtherGitLabInstanceAsConnected(t *testing.T) {
	app := providerTestApp(t)
	app.Cfg.Providers["gitlab"] = append(app.Cfg.Providers["gitlab"], infrastructure.ProviderConfig{
		Provider:        "gitlab",
		ServiceProvider: "tencent-sh-1",
		ProviderKey:     "gitlab:tencent-sh-1",
		Website:         "https://gitlab-tencent-sh-1.daydaymoney.com",
	})
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "877397583960502272"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-gl-sh1', 'root', 'read_repository api read_user',
        'active', '', NOW(), NOW())`,
		"gitlab:tencent-sh-1", userID)
	if err != nil {
		t.Fatal(err)
	}

	reqOther := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?repo_url=http%3A%2F%2Flocalhost%3A8012%2Fgroup%2Frepo", nil)
	reqOther.Header.Set("X-User-Id", userID)
	recOther := httptest.NewRecorder()
	mux.ServeHTTP(recOther, reqOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("other host: expected 200, got %d body=%s", recOther.Code, recOther.Body.String())
	}
	var other map[string]any
	if err := json.Unmarshal(recOther.Body.Bytes(), &other); err != nil {
		t.Fatal(err)
	}
	if other["connected"] != false {
		t.Fatalf("binding gitlab:tencent-sh-1 must not mark localhost:8012 connected; body=%s", recOther.Body.String())
	}

	reqSame := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?repo_url=https%3A%2F%2Fgitlab-tencent-sh-1.daydaymoney.com%2Fgroup%2Frepo", nil)
	reqSame.Header.Set("X-User-Id", userID)
	recSame := httptest.NewRecorder()
	mux.ServeHTTP(recSame, reqSame)
	if recSame.Code != http.StatusOK {
		t.Fatalf("same host: expected 200, got %d body=%s", recSame.Code, recSame.Body.String())
	}
	var same map[string]any
	if err := json.Unmarshal(recSame.Body.Bytes(), &same); err != nil {
		t.Fatal(err)
	}
	if same["connected"] != true {
		t.Fatalf("tencent-sh-1 repo_url must find tencent-sh-1 credential; body=%s", recSame.Body.String())
	}
}

// TestHandleUserAppConnectionGetNonNumericUserID — 回归 2026-08-07：
// bootstrap-admin（taskAuth 确定性非数字 user ID）查询连接状态必须 200
// （此前 parsePositiveInt 拒绝 → 401，前端连接面板对 super admin 不可用）。
func TestHandleUserAppConnectionGetNonNumericUserID(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "bootstrap-admin"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-1', 'octocat', 'repo read:user',
        'active', '', NOW(), NOW())`,
		"github:github-official-daydaymoney", userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?service_provider=github-official-daydaymoney", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bootstrap-admin: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != true {
		t.Fatalf("connected=%v body=%s", body["connected"], rec.Body.String())
	}
}

func TestHandleUserAppConnectionGetConnected(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "873093522473906176"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-1', 'octocat', 'repo read:user',
        'active', '', NOW(), NOW())`,
		"github:github-official-daydaymoney", userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/git-oauth/user-app-connection/?service_provider=github-official-daydaymoney", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["connected"] != true {
		t.Fatalf("connected=%v body=%s", body["connected"], rec.Body.String())
	}
	connections, ok := body["connections"].([]any)
	if !ok || len(connections) != 1 {
		t.Fatalf("connections=%v", body["connections"])
	}
	first, _ := connections[0].(map[string]any)
	if first["github_login"] != "octocat" {
		t.Fatalf("github_login=%v", first["github_login"])
	}
}

func TestHandleUserAppConnectionDeleteConventionPath(t *testing.T) {
	app := providerTestApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	const userID = "873093522473906176"
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_appusercredential
  (provider, task2app_user_id, refresh_token_cipher, remote_user_id, remote_login,
   scope, bind_status, bind_error, created_at, updated_at)
VALUES (?, ?, 'cipher', 'remote-1', 'octocat', 'repo read:user',
        'active', '', NOW(), NOW())`,
		"github:github-official-daydaymoney", userID)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete,
		"/api/git-oauth/user-app-connection/?service_provider=github-official-daydaymoney", nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["deleted_count"] != float64(1) {
		t.Fatalf("deleted_count=%v", body["deleted_count"])
	}
}
