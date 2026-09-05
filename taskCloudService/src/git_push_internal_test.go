package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLayerGitPushPrepare_ReuseGithubAuthByRepo(t *testing.T) {
	cfg.InternalSecret = ""
	cfg.GitOauthBaseURL = "http://127.0.0.1:1" // must not be called

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task1","layer_id":"L1","user_id":"42","target_branch":"feat/x","github_auth_by_repo":{"acme/repo":"ghu_tok"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("ok=%v", out["ok"])
	}
	if out["use_oauth_access_push"] != true {
		t.Fatalf("use_oauth=%v", out["use_oauth_access_push"])
	}
	pb, _ := out["push_body"].(map[string]any)
	auth, _ := pb["github_auth_by_repo"].(map[string]any)
	if auth["acme/repo"] != "ghu_tok" {
		t.Fatalf("push_body=%v", pb)
	}
	if pb["target_branch"] != "feat/x" {
		t.Fatalf("target_branch=%v", pb["target_branch"])
	}
}

func TestLayerGitPushPrepare_SentinelUserIDRejected(t *testing.T) {
	cfg.InternalSecret = ""
	cfg.GitOauthBaseURL = "http://127.0.0.1:1"

	body := `{"tenant_id":"1","task_id":"task1","layer_id":"L1","user_id":"internal_gateway","identity_id":"gi_1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s want 401 not fake identity 404", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["detail"] != "内部推送缺少真实用户身份，无法查找 Git 身份" {
		t.Fatalf("detail=%v", out["detail"])
	}
}

func TestLayerGitPushPrepare_IdentityIDNotFound(t *testing.T) {
	cfg.InternalSecret = ""
	store := newSaasHTTPStore()
	_ = startSaasInternalMock(t, store)
	prevFn := fetchUserCompanyGitIdentityFn
	fetchUserCompanyGitIdentityFn = func(_ context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
		return storeLookupGitIdentity(store, identityID, userID, companyID)
	}
	t.Cleanup(func() { fetchUserCompanyGitIdentityFn = prevFn })

	body := `{"tenant_id":"1","task_id":"task1","layer_id":"L1","user_id":"42","identity_id":"99"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["detail"] != "Git 身份不存在或不属于当前租户" {
		t.Fatalf("detail=%v", out["detail"])
	}
}

func TestLayerGitPushPrepare_IdentityIDSuccess(t *testing.T) {
	cfg.InternalSecret = ""
	store := newSaasHTTPStore()
	store.putGitIdentity("9", "42", "1", "dev", "dev@example.com")
	_ = startSaasInternalMock(t, store)
	prevFn := fetchUserCompanyGitIdentityFn
	fetchUserCompanyGitIdentityFn = func(_ context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
		return storeLookupGitIdentity(store, identityID, userID, companyID)
	}
	t.Cleanup(func() { fetchUserCompanyGitIdentityFn = prevFn })
	prevTask := cfg.TaskServiceURL
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "42", "github.com", nil).URL

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/github/oauth/access-for-user/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghu_identity"})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"1","task_id":"task1","layer_id":"L1","user_id":"42","identity_id":"9","repo_url":"https://github.com/acme/demo.git"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true {
		t.Fatalf("ok=%v", out["ok"])
	}
	pb, _ := out["push_body"].(map[string]any)
	auth, _ := pb["github_auth_by_repo"].(map[string]any)
	if auth["acme/demo"] != "ghu_identity" {
		t.Fatalf("auth=%v", auth)
	}
}

func TestLayerGitPushPrepare_PreferContainerRemoteWithoutBareOptInReturns409(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	cfg.TaskServiceURL = ""
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })
	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","prefer_container_remote":true,"target_branch":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s want 409", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != false {
		t.Fatalf("ok=%v want false", out["ok"])
	}
	detail, _ := out["detail"].(string)
	if detail == "" || strings.Contains(detail, "terminal prompts") {
		t.Fatalf("detail=%q", detail)
	}
}

func TestLayerGitPushPrepare_PreferContainerRemoteAllowBare(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	cfg.TaskServiceURL = ""
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })
	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","prefer_container_remote":true,"allow_bare_git_push":true,"target_branch":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["use_oauth_access_push"] != false {
		t.Fatalf("use_oauth=%v", out["use_oauth_access_push"])
	}
}

func TestLayerGitPushPrepare_GitlabOauthWhenLegacySummary404(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	prevProj := cfg.ProjectServiceURL
	prevOauth := cfg.GitOauthBaseURL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.ProjectServiceURL = prevProj
		cfg.GitOauthBaseURL = prevOauth
	})
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "877397583960502272", "gitlab-tencent-sh-1.daydaymoney.com", nil).URL
	cfg.ProjectServiceURL = ""

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "user-credential/summary-for-user") {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/oauth/access-for-user/") {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "glpat_from_access"})
			return
		}
		http.NotFound(w, r)
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"877397583960502272","prefer_container_remote":true,"identity_id":"gi_1","repo_url":"https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad","target_branch":"feature/x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200 (legacy summary 404 must not skip access-for-user)", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["use_oauth_access_push"] != true {
		t.Fatalf("use_oauth=%v body=%v", out["use_oauth_access_push"], out)
	}
	pb, _ := out["push_body"].(map[string]any)
	oauthAuth, _ := pb["oauth_auth_by_repo"].(map[string]any)
	if len(oauthAuth) == 0 {
		t.Fatalf("oauth_auth_by_repo empty: %v", pb)
	}
}

func TestLayerGitPushPrepare_PreferContainerRemoteWithGitlabOauth(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	prevProj := cfg.ProjectServiceURL
	prevOauth := cfg.GitOauthBaseURL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.ProjectServiceURL = prevProj
		cfg.GitOauthBaseURL = prevOauth
	})

	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mimic live taskTaskService: projects[] with stored_repo_address, no project_ids.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "task1",
			"projects": []any{
				map[string]any{
					"project_id":          "proj_1",
					"stored_repo_address": "https://gitlab.daydaymoney.com/ljy/somanyad-emailD.git",
					"project_repo_url":    "https://gitlab.daydaymoney.com/ljy/somanyad-emailD.git",
				},
				map[string]any{
					"project_id":          "proj_1",
					"stored_repo_address": "https://gitlab.daydaymoney.com/ljy/somanyad.git",
					"project_repo_url":    "https://gitlab.daydaymoney.com/ljy/somanyad-emailD.git",
				},
			},
			"comments": []any{
				map[string]any{
					"created_by": map[string]any{"id": "42"},
					"repo_identities": []any{
						map[string]any{"oauth_gitsite": "gitlab.daydaymoney.com"},
					},
				},
			},
		})
	}))
	defer taskSrv.Close()
	cfg.TaskServiceURL = taskSrv.URL
	cfg.ProjectServiceURL = ""

	oauthHits := 0
	seenKeys := map[string]bool{}
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)
		pk, _ := reqBody["provider_key"].(string)
		switch {
		case strings.Contains(r.URL.Path, "credential-summary"):
			// Only daydaymoney-gitlab is connected (matches production daydaymoney binding).
			connected := pk == "gitlab:daydaymoney-gitlab"
			_ = json.NewEncoder(w).Encode(map[string]any{"connected": connected})
		case strings.HasSuffix(r.URL.Path, "/oauth/access-for-user/"):
			if pk != "gitlab:daydaymoney-gitlab" {
				http.NotFound(w, r)
				return
			}
			oauthHits++
			seenKeys[pk] = true
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "glpat_multi"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task1","layer_id":"L1","user_id":"42","prefer_container_remote":true,"identity_id":"","repo_url":"","target_branch":"feature/x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["use_oauth_access_push"] != true {
		t.Fatalf("use_oauth=%v body=%v", out["use_oauth_access_push"], out)
	}
	pb, _ := out["push_body"].(map[string]any)
	oauthAuth, _ := pb["oauth_auth_by_repo"].(map[string]any)
	if len(oauthAuth) < 2 {
		t.Fatalf("oauth_auth_by_repo=%v", oauthAuth)
	}
	entry, _ := oauthAuth["https://gitlab.daydaymoney.com/ljy/somanyad"].(map[string]any)
	if entry["access_token"] != "glpat_multi" || entry["provider"] != "gitlab" {
		t.Fatalf("somanyad entry=%v", entry)
	}
	if entry["provider_key"] != "gitlab:daydaymoney-gitlab" {
		t.Fatalf("provider_key=%v", entry["provider_key"])
	}
	if oauthHits != 1 {
		t.Fatalf("expected single gitlab token refresh, hits=%d keys=%v", oauthHits, seenKeys)
	}
}

func TestRelatedProjectsFromTaskProjectsField(t *testing.T) {
	raw := []any{
		map[string]any{
			"project_id":          "p1",
			"stored_repo_address": "https://gitlab.daydaymoney.com/a/b.git",
		},
		map[string]any{
			"project_id":          "p1",
			"stored_repo_address": "https://gitlab.daydaymoney.com/a/c.git",
		},
	}
	got := relatedProjectsFromTaskProjectsField(raw)
	if len(got) != 1 || len(got[0].Repos) != 2 {
		t.Fatalf("got=%+v", got)
	}
}

func TestLayerGitPushPrepare_GitOauthUnavailable(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "42", "github.com", nil).URL
	cfg.GitOauthBaseURL = "http://127.0.0.1:1"
	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","repo_url":"https://github.com/acme/demo.git"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	detail, _ := out["detail"].(string)
	if !strings.Contains(detail, "gitOauth") {
		t.Fatalf("expected explicit gitOauth error, got %q", detail)
	}
}

func TestLayerGitPushPrepare_GitOauthSuccess(t *testing.T) {
	cfg.InternalSecret = ""
	prevTask := cfg.TaskServiceURL
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "42", "github.com", nil).URL
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/github/oauth/access-for-user/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghu_from_oauth"})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","repo_url":"https://github.com/acme/demo.git"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	pb, _ := out["push_body"].(map[string]any)
	auth, _ := pb["github_auth_by_repo"].(map[string]any)
	if auth["acme/demo"] != "ghu_from_oauth" {
		t.Fatalf("auth=%v", auth)
	}
}

func TestLayerGitPushComplete_OK(t *testing.T) {
	cfg.InternalSecret = ""
	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","container_upstream":{"ok":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushComplete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != true {
		t.Fatalf("body=%v", out)
	}
}

func TestGithubRepoSlugFromURL(t *testing.T) {
	if got := githubRepoSlugFromURL("https://github.com/Acme/Demo.git"); got != "acme/demo" {
		t.Fatalf("got=%q", got)
	}
	if got := githubRepoSlugFromURL("git@github.com:Acme/Demo.git"); got != "acme/demo" {
		t.Fatalf("ssh got=%q", got)
	}
}

func TestFetchGitOauthAccessForUser_SendsBridgeSecret(t *testing.T) {
	prevURL := cfg.GitOauthBaseURL
	prevSecret := cfg.GitOauthBridgeSecret
	t.Cleanup(func() {
		cfg.GitOauthBaseURL = prevURL
		cfg.GitOauthBridgeSecret = prevSecret
	})

	gotHeader := ""
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-GitOauth-Bridge-Secret")
		if r.URL.Path != "/api/internal/github/oauth/access-for-user/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ghu_bridge_ok"})
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL
	cfg.GitOauthBridgeSecret = "test-bridge-secret"

	tok, errDetail := fetchGitOauthAccessForUser("42", "github", "github:github-official-daydaymoney")
	if errDetail != "" {
		t.Fatalf("errDetail=%q", errDetail)
	}
	if tok != "ghu_bridge_ok" {
		t.Fatalf("tok=%q", tok)
	}
	if gotHeader != "test-bridge-secret" {
		t.Fatalf("bridge header=%q", gotHeader)
	}
}

func TestFetchGitOauthAccessForUser_404PropagatesErrDetail(t *testing.T) {
	prevURL := cfg.GitOauthBaseURL
	prevSecret := cfg.GitOauthBridgeSecret
	t.Cleanup(func() {
		cfg.GitOauthBaseURL = prevURL
		cfg.GitOauthBridgeSecret = prevSecret
	})
	cfg.GitOauthBridgeSecret = "test-secret"

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"detail":"not_found","errDetail":"未找到该用户的 Git OAuth 授权，请重新完成授权"}`)
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	tok, errDetail := fetchGitOauthAccessForUser("42", "github", "github:github-official-daydaymoney")
	if tok != "" {
		t.Fatalf("tok=%q want empty on 404", tok)
	}
	if !strings.Contains(errDetail, "重新完成授权") {
		t.Fatalf("errDetail=%q want taskGitOauth errDetail propagated", errDetail)
	}
}

func TestFetchGitOauthAccessForUser_404WithoutBodyFallsBackToGuidance(t *testing.T) {
	prevURL := cfg.GitOauthBaseURL
	prevSecret := cfg.GitOauthBridgeSecret
	t.Cleanup(func() {
		cfg.GitOauthBaseURL = prevURL
		cfg.GitOauthBridgeSecret = prevSecret
	})
	cfg.GitOauthBridgeSecret = "test-secret"

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, ``)
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	tok, errDetail := fetchGitOauthAccessForUser("42", "github", "github:github-official-daydaymoney")
	if tok != "" {
		t.Fatalf("tok=%q", tok)
	}
	if errDetail == "" || !strings.Contains(errDetail, "Git OAuth 授权") {
		t.Fatalf("errDetail=%q want fallback guidance", errDetail)
	}
}

func TestLayerGitPushPrepare_UnauthorizedBridgeMapsGuidance(t *testing.T) {
	cfg.InternalSecret = ""
	prevURL := cfg.GitOauthBaseURL
	prevSecret := cfg.GitOauthBridgeSecret
	prevTask := cfg.TaskServiceURL
	t.Cleanup(func() {
		cfg.GitOauthBaseURL = prevURL
		cfg.GitOauthBridgeSecret = prevSecret
		cfg.TaskServiceURL = prevTask
	})
	cfg.TaskServiceURL = startTaskCommentGrantServer(t, "42", "github.com", nil).URL
	cfg.GitOauthBridgeSecret = "wrong-secret"

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":"unauthorized"}`)
	}))
	defer oauth.Close()
	cfg.GitOauthBaseURL = oauth.URL

	body := `{"tenant_id":"t1","task_id":"task1","layer_id":"L1","user_id":"42","prefer_container_remote":true,"repo_url":"https://github.com/acme/demo.git"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-push/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitPushPrepare(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s want 502", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	detail, _ := out["detail"].(string)
	if !strings.Contains(detail, "bridge secret") {
		t.Fatalf("detail=%q want bridge secret guidance", detail)
	}
}
