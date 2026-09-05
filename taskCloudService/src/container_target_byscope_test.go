package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTokenByScopeResult_PropagatesErrorCode(t *testing.T) {
	prevURL := cfg.CredentialServiceURL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevURL })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v1/token/by-scope") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"token exists but access_token has expired","error_code":"TOKEN_EXPIRED"}`))
	}))
	t.Cleanup(srv.Close)
	cfg.CredentialServiceURL = srv.URL

	got := fetchTokenByScopeResult("t1", "ws1", "task1", "")
	if got.Token != "" {
		t.Fatalf("expected empty token, got %q", got.Token)
	}
	if got.HTTPStatus != http.StatusNotFound {
		t.Fatalf("HTTPStatus=%d", got.HTTPStatus)
	}
	if got.ErrorCode != "TOKEN_EXPIRED" {
		t.Fatalf("ErrorCode=%q", got.ErrorCode)
	}
	if !strings.Contains(got.Detail, "expired") {
		t.Fatalf("Detail=%q", got.Detail)
	}
}

func TestFetchTokenByScopeResult_OK(t *testing.T) {
	prevURL := cfg.CredentialServiceURL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevURL })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-live","error_code":""}`))
	}))
	t.Cleanup(srv.Close)
	cfg.CredentialServiceURL = srv.URL

	got := fetchTokenByScopeResult("t1", "ws1", "task1", "")
	if got.Token != "tok-live" {
		t.Fatalf("Token=%q", got.Token)
	}
	if got.HTTPStatus != http.StatusOK {
		t.Fatalf("HTTPStatus=%d", got.HTTPStatus)
	}
	if got.ErrorCode != "" {
		t.Fatalf("ErrorCode=%q want empty", got.ErrorCode)
	}
}

func TestFetchTokenByScopeResult_IncludesCommentID(t *testing.T) {
	prevURL := cfg.CredentialServiceURL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevURL })

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-cmt"}`))
	}))
	t.Cleanup(srv.Close)
	cfg.CredentialServiceURL = srv.URL

	got := fetchTokenByScopeResult("t1", "ws1", "task1", "cmt_9")
	if got.Token != "tok-cmt" {
		t.Fatalf("Token=%q", got.Token)
	}
	if !strings.Contains(gotQuery, "comment_id=cmt_9") {
		t.Fatalf("query missing comment_id: %s", gotQuery)
	}
}

func TestInternalContainerTarget_MissingTokenIncludesErrorCode(t *testing.T) {
	setupCloudTestDB(t)
	// server_url 有 base 但无 token → 走 by-scope
	seedCloudConfig(t, "t1", "ws1", "task-miss-tok", "http://203.0.113.50:8765/")

	prevURL := cfg.CredentialServiceURL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevURL })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"no token found for this scope","error_code":"TOKEN_NOT_FOUND"}`))
	}))
	t.Cleanup(srv.Close)
	cfg.CredentialServiceURL = srv.URL

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/container-target/?tenant_id=t1&workspace_id=ws1&task_id=task-miss-tok", nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["detail"] != "缺少容器 access_token" {
		t.Fatalf("detail=%v", body["detail"])
	}
	if body["error_code"] != "TOKEN_NOT_FOUND" {
		t.Fatalf("error_code=%v body=%s", body["error_code"], rec.Body.String())
	}
	if body["credential_status"] != float64(404) && body["credential_status"] != 404 {
		t.Fatalf("credential_status=%v", body["credential_status"])
	}
}
