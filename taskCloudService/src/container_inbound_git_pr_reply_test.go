package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitPrProviderOf(t *testing.T) {
	if got := gitPrProviderOf("https://gitlab.example/a/b/-/merge_requests/2"); got != "gitlab" {
		t.Fatalf("provider=%q", got)
	}
	if got := gitPrProviderOf("https://github.com/acme/demo/pull/3"); got != "github" {
		t.Fatalf("provider=%q", got)
	}
}

func TestHandleGitPrReplyPostsTaskCommentAndIsIdempotent(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-gpr", "task-gpr", "cmt-exec", "i-gpr", "203.0.113.90")

	var posts []string
	var gotUser, gotTenant, gotPath string
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/created-by") && r.Method == http.MethodGet {
			if r.Header.Get("X-Internal-Secret") != "sec-gpr" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"comment_id":"cmt-exec","user_id":"u-author"}`)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		gotUser = r.Header.Get("X-Auth-User-Id")
		gotTenant = r.Header.Get("X-Auth-Tenant-Id")
		raw, _ := io.ReadAll(r.Body)
		posts = append(posts, string(raw))
		w.Header().Set("Content-Type", "application/json")
		if len(posts) == 1 {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"cmt_pr1","parent_comment_id":"cmt-exec"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"cmt_pr1","parent_comment_id":"cmt-exec"}`)
	}))
	defer taskSrv.Close()

	prevTask := cfg.TaskServiceURL
	prevSec := cfg.InternalSecret
	prevCred := cfg.CredentialServiceURL
	cfg.TaskServiceURL = taskSrv.URL
	cfg.InternalSecret = "sec-gpr"
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"ws1","task_id":"task-gpr","expires_at":"2099-01-01"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.InternalSecret = prevSec
		cfg.CredentialServiceURL = prevCred
	})

	htmlURL := "https://gitlab.example/a/b/-/merge_requests/4"
	body, _ := json.Marshal(map[string]any{
		"access_token":      "tok",
		"comment_id":        "cmt-exec",
		"html_url":          htmlURL,
		"parent_comment_id": "cmt-exec",
		"git_pr":            map[string]string{"html_url": htmlURL, "provider": "gitlab"},
	})
	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/git-pr-reply", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleContainerInboundToken(rec, req, "t1", "ws1", "task-gpr", "git-pr-reply")
		return rec
	}

	first := post()
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	var out1 map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &out1); err != nil {
		t.Fatal(err)
	}
	if out1["id"] != "cmt_pr1" || out1["skipped"] != false {
		t.Fatalf("first=%v", out1)
	}

	second := post()
	if second.Code != http.StatusOK {
		t.Fatalf("second status=%d body=%s", second.Code, second.Body.String())
	}
	var out2 map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &out2); err != nil {
		t.Fatal(err)
	}
	if out2["skipped"] != true || out2["id"] != "cmt_pr1" {
		t.Fatalf("second=%v", out2)
	}

	if len(posts) != 2 {
		t.Fatalf("posts=%d", len(posts))
	}
	if gotUser != "u-author" || gotTenant != "t1" {
		t.Fatalf("auth user=%q tenant=%q", gotUser, gotTenant)
	}
	if !strings.Contains(gotPath, "/api/tenant_id/t1/workspaceId/ws1/tasks/task-gpr/comments/cmt-exec/") {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.Contains(posts[0], `"execution_mode":"independent"`) || !strings.Contains(posts[0], htmlURL) {
		t.Fatalf("body=%s", posts[0])
	}
}

func TestHandleGitPrReplyRequiresHTMLURL(t *testing.T) {
	setupCloudTestDB(t)
	seedInboundCSC(t, "cfg-gpr2", "task-gpr2", "cmt-exec2", "i-gpr2", "203.0.113.91")
	prevCred := cfg.CredentialServiceURL
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"ws1","task_id":"task-gpr2","expires_at":"2099-01-01"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	body, _ := json.Marshal(map[string]any{
		"access_token": "tok",
		"comment_id":   "cmt-exec2",
	})
	req := httptest.NewRequest(http.MethodPost, "/git-pr-reply", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleContainerInboundToken(rec, req, "t1", "ws1", "task-gpr2", "git-pr-reply")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
