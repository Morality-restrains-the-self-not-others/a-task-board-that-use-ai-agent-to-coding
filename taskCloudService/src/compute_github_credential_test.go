package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- github-credential-status / github-credential-approve Go 重新实现测试 ---
// OPT-052: Django saas-backend retired 2026-07-30；本测试验证 taskCloudService
// 本地实现（gitOauth summary-for-user + cloud_task_repo_github_bindings 表）。

// githubCredentialTestEnv spins up mock gitOauth (credential summary) and mock
// task service (task detail with projects[].stored_repo_address), and points
// cfg at them. Returns a cleanup func.
type githubCredentialTestEnv struct {
	gitOauthCalls int
	taskCalls     int
	cleanup       func()
}

func newGithubCredentialTestEnv(t *testing.T, summaryBody string, taskProjects []map[string]any) *githubCredentialTestEnv {
	t.Helper()
	env := &githubCredentialTestEnv{}

	gitOauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.gitOauthCalls++
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/github-credential-summary/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, summaryBody)
	}))

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env.taskCalls++
		if !strings.HasPrefix(r.URL.Path, "/api/tasks/") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": "task1", "projects": %s}`, mustJSONArray(t, taskProjects))
	}))

	oldGitOauth := cfg.GitOauthBaseURL
	oldTaskSrv := cfg.TaskServiceURL
	oldSecret := cfg.GitOauthBridgeSecret
	cfg.GitOauthBaseURL = gitOauth.URL
	cfg.TaskServiceURL = taskSrv.URL
	cfg.GitOauthBridgeSecret = "test-secret"

	env.cleanup = func() {
		cfg.GitOauthBaseURL = oldGitOauth
		cfg.TaskServiceURL = oldTaskSrv
		cfg.GitOauthBridgeSecret = oldSecret
		gitOauth.Close()
		taskSrv.Close()
	}
	t.Cleanup(env.cleanup)
	return env
}

// TestGithubCredentialApproveTaskServiceDown — 回归 OPT-20260809-014：
// taskTaskService 返回 5xx 时 approve 必须 503 + detail（而非误报业务 400「仓库不在任务关联项目」），
// status 端点保持空列表降级 200。
func TestGithubCredentialApproveTaskServiceDown(t *testing.T) {
	setupCloudTestDB(t)
	gitOauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, testGithubSummaryConnected)
	}))
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "task service unavailable", http.StatusBadGateway)
	}))

	oldGitOauth := cfg.GitOauthBaseURL
	oldTaskSrv := cfg.TaskServiceURL
	oldSecret := cfg.GitOauthBridgeSecret
	cfg.GitOauthBaseURL = gitOauth.URL
	cfg.TaskServiceURL = taskSrv.URL
	cfg.GitOauthBridgeSecret = "test-secret"
	t.Cleanup(func() {
		cfg.GitOauthBaseURL = oldGitOauth
		cfg.TaskServiceURL = oldTaskSrv
		cfg.GitOauthBridgeSecret = oldSecret
		gitOauth.Close()
		taskSrv.Close()
	})

	// approve：收集失败 → 503，而非 400 业务拒绝
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialApproveRequest(
		`{"repo_url":"https://github.com/Acme/RepoA.git","repo_slug":"acme/repoa","github_user_id":"1321779"}`),
		"cloud/compute/github-credential-approve/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("approve expected 503 on task service down, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "收集失败") {
		t.Fatalf("approve 503 body must explain collection failure, got %s", rec.Body.String())
	}

	// status：保持空列表降级 200（展示层可接受）
	rec2 := httptest.NewRecorder()
	handleCloudTaskRoutes(rec2, githubCredentialStatusRequest("task1"), "cloud/compute/github-credential-status/")
	if rec2.Code != http.StatusOK {
		t.Fatalf("status expected 200 degradation on task service down, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func mustJSONArray(t *testing.T, items []map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}

func githubCredentialStatusRequest(taskID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/task/task1/cloud/compute/github-credential-status/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	if taskID != "" {
		req.Header.Set("X-Task-Id", taskID)
	}
	req.Header.Set("X-Auth-User-Id", "user1")
	return req
}

func githubCredentialApproveRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/cloud/compute/github-credential-approve/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	req.Header.Set("X-Auth-User-Id", "user1")
	return req
}

const testGithubSummaryConnected = `{
  "connected": true,
  "github_user_id": "1321779",
  "github_login": "ruandao",
  "scope": "repo",
  "bind_status": "active",
  "bind_error": null,
  "connections": [
    {"connected": true, "github_user_id": "1321779", "github_login": "ruandao",
     "scope": "repo", "bind_status": "active", "bind_error": null, "updated_at": null}
  ]
}`

const testGithubSummaryDisconnected = `{
  "connected": false,
  "github_user_id": null,
  "github_login": null,
  "scope": null,
  "bind_status": null,
  "bind_error": null,
  "connections": []
}`

func testRepoProject(repoURL string) map[string]any {
	return map[string]any{
		"project_id":          "proj1",
		"project_name":        "proj1",
		"stored_repo_address": repoURL,
	}
}

// TestGithubCredentialStatusReturnsBindings: 已连接 GitHub 账号 + 已绑定仓库 →
// 200，github_connections / repo_bindings / all_repo_bound 完整。
func TestGithubCredentialStatusReturnsBindings(t *testing.T) {
	setupCloudTestDB(t)
	newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
		testRepoProject("https://github.com/Acme/RepoB.git"),
	})

	if err := upsertTaskRepoGithubBinding("task1", "https://github.com/Acme/RepoA.git", "acme/repoa", "1321779"); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialStatusRequest("task1"), "cloud/compute/github-credential-status/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		ApprovedForTask    bool   `json:"approved_for_task"`
		GithubAppConnected bool   `json:"github_app_connected"`
		GithubLogin        string `json:"github_login"`
		GithubConnections  []struct {
			Connected    bool   `json:"connected"`
			GithubUserID string `json:"github_user_id"`
			GithubLogin  string `json:"github_login"`
		} `json:"github_connections"`
		RepoBindings []struct {
			RepoURL              string `json:"repo_url"`
			RepoSlug             string `json:"repo_slug"`
			SelectedGithubUserID string `json:"selected_github_user_id"`
			SelectedGithubLogin  string `json:"selected_github_login"`
		} `json:"repo_bindings"`
		AllRepoBound bool `json:"all_repo_bound"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
	}
	if !body.GithubAppConnected {
		t.Fatalf("github_app_connected must be true: %s", rec.Body.String())
	}
	if body.GithubLogin != "ruandao" {
		t.Fatalf("github_login=ruandao, got %q", body.GithubLogin)
	}
	if len(body.GithubConnections) != 1 || body.GithubConnections[0].GithubUserID != "1321779" {
		t.Fatalf("connections mismatch: %+v", body.GithubConnections)
	}
	if len(body.RepoBindings) != 2 {
		t.Fatalf("repo_bindings must cover 2 repos, got %+v", body.RepoBindings)
	}
	var boundRepoA, unboundRepoB bool
	for _, rb := range body.RepoBindings {
		switch rb.RepoSlug {
		case "acme/repoa":
			boundRepoA = rb.SelectedGithubUserID == "1321779" && rb.SelectedGithubLogin == "ruandao"
		case "acme/repob":
			unboundRepoB = rb.SelectedGithubUserID == ""
		}
	}
	if !boundRepoA || !unboundRepoB {
		t.Fatalf("binding resolution mismatch: %+v", body.RepoBindings)
	}
	if body.AllRepoBound {
		t.Fatalf("all_repo_bound must be false when RepoB unbound: %s", rec.Body.String())
	}
	if body.ApprovedForTask {
		t.Fatalf("approved_for_task must be false when a repo is unbound")
	}
}

// TestGithubCredentialStatusAllBound: 全部仓库绑定 → all_repo_bound=true, approved_for_task=true。
func TestGithubCredentialStatusAllBound(t *testing.T) {
	setupCloudTestDB(t)
	env := newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})
	if err := upsertTaskRepoGithubBinding("task1", "https://github.com/Acme/RepoA.git", "acme/repoa", "1321779"); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialStatusRequest("task1"), "cloud/compute/github-credential-status/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		AllRepoBound    bool `json:"all_repo_bound"`
		ApprovedForTask bool `json:"approved_for_task"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.AllRepoBound || !body.ApprovedForTask {
		t.Fatalf("expected all_repo_bound=true approved_for_task=true, got %+v", body)
	}
	_ = env
}

// TestGithubCredentialStatusEmptyState: 无连接无仓库 → 200 空数组，不报错。
func TestGithubCredentialStatusEmptyState(t *testing.T) {
	setupCloudTestDB(t)
	newGithubCredentialTestEnv(t, testGithubSummaryDisconnected, nil)

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialStatusRequest("task1"), "cloud/compute/github-credential-status/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		GithubConnections []any `json:"github_connections"`
		RepoBindings      []any `json:"repo_bindings"`
		AllRepoBound      bool  `json:"all_repo_bound"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.GithubConnections == nil || body.RepoBindings == nil {
		t.Fatalf("arrays must be non-nil: %s", rec.Body.String())
	}
	if body.AllRepoBound {
		t.Fatalf("all_repo_bound must be false")
	}
}

// TestGithubCredentialStatusUsesGatewayXUserId — 回归：APISIX 只注入 X-User-Id
// （+ verified + internal secret），不直接带 X-Auth-User-Id。若缺少
// gatewayUserMiddleware，getAuthUser 为空 → github_connections=[] → 任务详情下拉空。
func TestGithubCredentialStatusUsesGatewayXUserId(t *testing.T) {
	setupCloudTestDB(t)
	cfg.GatewayInternalSecret = "test-gw-secret"
	newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})

	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/github-credential-status/tenant_id/t1/workspace_id/ws1/task_id/task1/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
	req.Header.Set("X-User-Id", "user1")
	// 故意不设置 X-Auth-User-Id：模拟真实网关转发。

	handler := gatewayUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCloudTaskRoutes(w, r, "cloud/compute/github-credential-status/")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		GithubAppConnected bool `json:"github_app_connected"`
		GithubConnections  []struct {
			Connected    bool   `json:"connected"`
			GithubUserID string `json:"github_user_id"`
			GithubLogin  string `json:"github_login"`
		} `json:"github_connections"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
	}
	if !body.GithubAppConnected {
		t.Fatalf("expected github_app_connected=true, body=%s", rec.Body.String())
	}
	if len(body.GithubConnections) != 1 || body.GithubConnections[0].GithubUserID != "1321779" {
		t.Fatalf("expected one connected github account 1321779, body=%s", rec.Body.String())
	}
}

// TestGithubCredentialStatusRejectsNonGET: 非 GET → 405。
func TestGithubCredentialStatusRejectsNonGET(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/cloud/compute/github-credential-status/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "cloud/compute/github-credential-status/")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGithubCredentialStatusMissingContext: 缺任务上下文 → 400。
func TestGithubCredentialStatusMissingContext(t *testing.T) {
	setupCloudTestDB(t)
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialStatusRequest(""), "cloud/compute/github-credential-status/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGithubCredentialApprovePersistsBinding: 合法仓库 + 已连接账号 →
// 200 approved=true，绑定落库，随后 status 反映绑定。
func TestGithubCredentialApprovePersistsBinding(t *testing.T) {
	setupCloudTestDB(t)
	env := newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialApproveRequest(
		`{"repo_url":"https://github.com/Acme/RepoA.git","repo_slug":"acme/repoa","github_user_id":"1321779"}`),
		"cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Approved     bool   `json:"approved"`
		RepoSlug     string `json:"repo_slug"`
		GithubUserID string `json:"github_user_id"`
		GithubLogin  string `json:"github_login"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Approved || body.RepoSlug != "acme/repoa" || body.GithubUserID != "1321779" || body.GithubLogin != "ruandao" {
		t.Fatalf("approve response mismatch: %+v", body)
	}

	rows, err := db.Query(`SELECT repo_url, github_user_id FROM cloud_task_repo_github_bindings WHERE task_id='task1'`)
	if err != nil {
		t.Fatalf("query bindings: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		var repoURL, gid string
		if err := rows.Scan(&repoURL, &gid); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if repoURL != "https://github.com/Acme/RepoA.git" || gid != "1321779" {
			t.Fatalf("binding row mismatch: %s %s", repoURL, gid)
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 binding row, got %d", count)
	}
	_ = env
}

// TestGithubCredentialApproveRejectsRepoNotInTask: 仓库不在任务关联项目 → 400。
func TestGithubCredentialApproveRejectsRepoNotInTask(t *testing.T) {
	setupCloudTestDB(t)
	newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialApproveRequest(
		`{"repo_url":"https://github.com/Other/Evil.git","repo_slug":"other/evil","github_user_id":"1321779"}`),
		"cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGithubCredentialApproveRejectsDisconnectedGithubUser: github_user_id 非当前用户已连接账号 → 400。
func TestGithubCredentialApproveRejectsDisconnectedGithubUser(t *testing.T) {
	setupCloudTestDB(t)
	newGithubCredentialTestEnv(t, testGithubSummaryConnected, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialApproveRequest(
		`{"repo_url":"https://github.com/Acme/RepoA.git","repo_slug":"acme/repoa","github_user_id":"999999"}`),
		"cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// TestGithubCredentialApproveUpdatesExistingBinding: 换绑到另一已连接账号 → upsert 更新。
func TestGithubCredentialApproveUpdatesExistingBinding(t *testing.T) {
	setupCloudTestDB(t)
	newGithubCredentialTestEnv(t, `{
	  "connected": true,
	  "github_user_id": "999",
	  "github_login": "second",
	  "connections": [
	    {"connected": true, "github_user_id": "1321779", "github_login": "ruandao", "scope": "repo", "bind_status": "active"},
	    {"connected": true, "github_user_id": "999", "github_login": "second", "scope": "repo", "bind_status": "active"}
	  ]
	}`, []map[string]any{
		testRepoProject("https://github.com/Acme/RepoA.git"),
	})
	if err := upsertTaskRepoGithubBinding("task1", "https://github.com/Acme/RepoA.git", "acme/repoa", "1321779"); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, githubCredentialApproveRequest(
		`{"repo_url":"https://github.com/Acme/RepoA.git","repo_slug":"acme/repoa","github_user_id":"999"}`),
		"cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var rowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_task_repo_github_bindings WHERE task_id='task1'`).Scan(&rowCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("expected 1 binding row after upsert, got %d", rowCount)
	}
	var gid string
	if err := db.QueryRow(`SELECT github_user_id FROM cloud_task_repo_github_bindings WHERE task_id='task1' AND repo_url='https://github.com/Acme/RepoA.git'`).Scan(&gid); err != nil {
		t.Fatalf("read gid: %v", err)
	}
	if gid != "999" {
		t.Fatalf("expected updated github_user_id=999, got %q", gid)
	}
}

// TestGithubCredentialApproveRejectsNonPOST: 非 POST → 405（替代旧 410 断言）。
func TestGithubCredentialApproveRejectsNonPOST(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/task/task1/cloud/compute/github-credential-approve/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "not yet ported") {
		t.Fatalf("GET must not hit unported 501: %s", rec.Body.String())
	}
}

// TestGithubCredentialApproveMissingContext: 缺任务上下文 → 400。
func TestGithubCredentialApproveMissingContext(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/cloud/compute/github-credential-approve/",
		strings.NewReader(`{"repo_url":"https://github.com/Acme/RepoA.git","repo_slug":"acme/repoa","github_user_id":"1321779"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "cloud/compute/github-credential-approve/")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
