package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"taskGitOauth/infrastructure"
)

func TestInvalidateTaskProjectGitLabConnCache(t *testing.T) {
	var gotPath string
	var gotSecret string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.String()
		gotSecret = r.Header.Get("X-Internal-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	app := &App{Cfg: &infrastructure.Config{
		TaskProjectServiceURL: srv.URL,
		DjangoInternalSecret:  "shared-secret",
	}}
	app.invalidateTaskProjectGitLabConnCache("T1")

	if gotMethod != http.MethodPost {
		t.Fatalf("method=%q want POST", gotMethod)
	}
	if gotPath != "/api/internal/taskproject/tenant-gitlab-cache/invalidate?company_id=T1" {
		t.Fatalf("path=%q want invalidate endpoint with company_id", gotPath)
	}
	if gotSecret != "shared-secret" {
		t.Fatalf("X-Internal-Secret=%q want shared-secret", gotSecret)
	}
}
