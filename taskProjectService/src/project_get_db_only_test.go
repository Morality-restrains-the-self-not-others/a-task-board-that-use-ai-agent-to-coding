package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetProjectDoesNotCallGitLabProbe(t *testing.T) {
	setupTestDB(t)

	var gitoauthHits atomic.Int32
	gitoauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gitoauthHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer gitoauthSrv.Close()
	oldGitoauth := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = gitoauthSrv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldGitoauth })

	body := `{"name":"probe-split","git_repos":["https://github.com/o/private.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("missing project id: %#v", created)
	}

	gitoauthHits.Store(0)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "1001")
	reqGet.Header.Set("X-Resource-Id", pid)
	reqGet.Header.Set("X-Trace-Id", "test-trace-get-project-db-only")
	recGet := httptest.NewRecorder()
	start := time.Now()
	handleGetProject(recGet, reqGet)
	elapsed := time.Since(start)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("GET project blocked on GitLab probe: elapsed=%s", elapsed)
	}
	if n := gitoauthHits.Load(); n != 0 {
		t.Fatalf("GET project must not call git-oauth/GitLab, hits=%d", n)
	}
	var detail map[string]interface{}
	_ = json.NewDecoder(recGet.Body).Decode(&detail)
	if got := detail["name"]; got != "probe-split" {
		t.Fatalf("name=%v", got)
	}
	rawStatus, _ := detail["git_repos_status"].([]interface{})
	if len(rawStatus) != 1 {
		t.Fatalf("expected placeholder git_repos_status, got %#v", detail["git_repos_status"])
	}
	st, _ := rawStatus[0].(map[string]interface{})
	if st["token_status"] != "not_applicable" {
		t.Fatalf("GET project must not live-probe; token_status=%#v", st["token_status"])
	}
}
