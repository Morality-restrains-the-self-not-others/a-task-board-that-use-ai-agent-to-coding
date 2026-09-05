package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tracelog"
)

func TestLayerGitRepoIdentitiesPrepare_Success(t *testing.T) {
	cfg.InternalSecret = ""
	store := newSaasHTTPStore()
	store.putGitIdentity("859748037572919296", "42", "850256677331562496", "Ljy", "ljy@example.com")
	_ = startSaasInternalMock(t, store)
	prevFn := fetchUserCompanyGitIdentityFn
	fetchUserCompanyGitIdentityFn = func(_ context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
		return storeLookupGitIdentity(store, identityID, userID, companyID)
	}
	t.Cleanup(func() { fetchUserCompanyGitIdentityFn = prevFn })

	body := `{
		"tenant_id":"850256677331562496",
		"user_id":"42",
		"repos":[
			{"repo_url":"https://gitlab.daydaymoney.com/example-user/somanyad-emailD.git","identity_id":"859748037572919296"},
			{"repo_url":"https://gitlab.daydaymoney.com/example-user/somanyad.git","identity_id":"859748037572919296"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-repo-identities/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitRepoIdentitiesPrepare(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		OK    bool `json:"ok"`
		Repos []struct {
			RepoMatchKey string `json:"repo_match_key"`
			UserName     string `json:"user_name"`
			UserEmail    string `json:"user_email"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || len(out.Repos) != 2 {
		t.Fatalf("out=%+v", out)
	}
	if out.Repos[0].RepoMatchKey != "gitlab.daydaymoney.com/example-user/somanyad-emaild" {
		t.Fatalf("key0=%q", out.Repos[0].RepoMatchKey)
	}
	if out.Repos[0].UserName != "Ljy" || out.Repos[0].UserEmail != "ljy@example.com" {
		t.Fatalf("identity fields=%+v", out.Repos[0])
	}
	if out.Repos[1].RepoMatchKey != "gitlab.daydaymoney.com/example-user/somanyad" {
		t.Fatalf("key1=%q", out.Repos[1].RepoMatchKey)
	}
}

func TestLayerGitRepoIdentitiesPrepare_IdentityNotFound(t *testing.T) {
	cfg.InternalSecret = ""
	store := newSaasHTTPStore()
	_ = startSaasInternalMock(t, store)
	prevFn := fetchUserCompanyGitIdentityFn
	fetchUserCompanyGitIdentityFn = func(_ context.Context, identityID, userID, companyID string) (*userCompanyGitIdentityRow, error) {
		return storeLookupGitIdentity(store, identityID, userID, companyID)
	}
	t.Cleanup(func() { fetchUserCompanyGitIdentityFn = prevFn })

	body := `{"tenant_id":"1","user_id":"42","repos":[{"repo_url":"https://github.com/a/b.git","identity_id":"99"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-repo-identities/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitRepoIdentitiesPrepare(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLayerGitRepoIdentitiesPrepare_EmptyRepos(t *testing.T) {
	cfg.InternalSecret = ""
	body := `{"tenant_id":"1","user_id":"42","repos":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/layer-git-repo-identities/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalLayerGitRepoIdentitiesPrepare(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestNoCrossDatabaseGitIdentitySQL 回归：OPT-20260820-016 已将 git 身份
// 查找改为 taskTaskService 内部 HTTP，禁止本仓残留对 task_task.*git_identit*
// 的跨库 SQL（原 MySQL 1146 Table doesn't exist 根因）。任何源码文件出现
// 该 FQN 即失败。
func TestNoCrossDatabaseGitIdentitySQL(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, fq := range []string{"task_task.task_git_identities", "task_task.git_identities"} {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			src, err := os.ReadFile(e.Name())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(src), fq) {
				t.Errorf("%s: cross-database reference %s; use taskTaskService internal API (OPT-20260820-016)", e.Name(), fq)
			}
		}
	}
}

// TestRealFetchUserCompanyGitIdentity_HTTP 用 httptest 模拟 taskTaskService
// lookup 端点，验证真实生产路径（HTTP）能命中行并正确透传归属校验结果。
func TestRealFetchUserCompanyGitIdentity_HTTP(t *testing.T) {
	prevURL := cfg.TaskServiceURL
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevURL
		cfg.InternalSecret = prevSecret
	})

	var gotBody map[string]string
	var gotSecret, gotUser, gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSecret = r.Header.Get("X-Internal-Secret")
		gotUser = r.Header.Get("X-Auth-User-Id")
		gotTrace = r.Header.Get("X-Trace-Id")
		if r.URL.Path != "/api/internal/git-identities/lookup/" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		if gotBody["identity_id"] == "gid-1146" && gotBody["user_id"] == "user-42" && gotBody["company_id"] == "tenant-9" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"found": true, "git_user_name": "Ljy", "git_user_email": "ljy@example.com",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"found": false})
	}))
	t.Cleanup(srv.Close)
	cfg.TaskServiceURL = srv.URL

	// OPT-20260821-015: 出站 lookup 透传入站 X-Trace-Id。
	row, err := realFetchUserCompanyGitIdentity(tracelog.ContextWithTraceID(context.Background(), "trace-git-1146"), "gid-1146", "user-42", "tenant-9")
	if err != nil {
		t.Fatalf("realFetch: %v", err)
	}
	if row == nil {
		t.Fatal("expected identity row")
	}
	if row.GitUserName != "Ljy" || row.GitUserEmail != "ljy@example.com" {
		t.Fatalf("row=%+v", row)
	}
	if gotSecret != "test-secret" {
		t.Fatalf("internal secret header=%q", gotSecret)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal", gotUser)
	}
	if gotTrace != "trace-git-1146" {
		t.Fatalf("X-Trace-Id=%q want trace-git-1146 (出站透传)", gotTrace)
	}
	if gotBody["identity_id"] != "gid-1146" || gotBody["user_id"] != "user-42" || gotBody["company_id"] != "tenant-9" {
		t.Fatalf("request body=%+v", gotBody)
	}

	// 归属不符（user_id 不匹配）→ found=false → nil
	miss, err := realFetchUserCompanyGitIdentity(context.Background(), "gid-1146", "other-user", "tenant-9")
	if err != nil {
		t.Fatalf("ownership miss: %v", err)
	}
	if miss != nil {
		t.Fatal("expected nil when user_id does not match")
	}

	// 空参数直接 nil，不发请求
	empty, err := realFetchUserCompanyGitIdentity(context.Background(), "", "user-42", "tenant-9")
	if err != nil {
		t.Fatalf("empty identity: %v", err)
	}
	if empty != nil {
		t.Fatal("expected nil for empty identity_id")
	}
}

// TestRealFetchUserCompanyGitIdentity_SendsInternalUserWhenSecretEmpty 回归：
// conf shared.internalSecret 为空时 Cloud 不会发 X-Internal-Secret，taskTaskService
// lookup 要求 isInternalCall（X-Auth-User-Id=internal）或 secret。缺 header 时
// 返回 403 {"error":"internal only"}，页面表现为「提交成功但推送失败」。
func TestRealFetchUserCompanyGitIdentity_SendsInternalUserWhenSecretEmpty(t *testing.T) {
	prevURL := cfg.TaskServiceURL
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevURL
		cfg.InternalSecret = prevSecret
	})

	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Auth-User-Id")
		if gotUser != "internal" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"internal only","message":"internal only","status":"error"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"found": true, "git_user_name": "Ljy", "git_user_email": "ljy@example.com",
		})
	}))
	t.Cleanup(srv.Close)
	cfg.TaskServiceURL = srv.URL

	row, err := realFetchUserCompanyGitIdentity(context.Background(), "gid-403", "user-42", "tenant-9")
	if err != nil {
		t.Fatalf("realFetch: %v", err)
	}
	if row == nil {
		t.Fatal("expected identity row when X-Auth-User-Id=internal")
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal (empty InternalSecret still must mark internal call)", gotUser)
	}
}
