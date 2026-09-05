package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSaasMachineContainerSkill(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/saas-machine-container.md", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /saas-machine-container.md: status %d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type=%q, want text/plain", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SaaS Machine Container Skill") {
		t.Fatalf("body missing skill title: %s", truncateForTest(body, 200))
	}
	if !strings.Contains(body, "server-container-token") {
		t.Fatalf("body missing server-container-token: %s", truncateForTest(body, 200))
	}
	for _, needle := range []string{
		"/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/",
		"/api/git-oauth/merge-request-status/tenant_id/{tid}/",
		"/api/git-oauth/merge-request-merge/tenant_id/{tid}/",
		"merge_request_merge",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("skill doc missing %q", needle)
		}
	}
}

func TestHandleSaasMachineContainerSkillMethodNotAllowed(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/saas-machine-container.md", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: status %d, want 405", rec.Code)
	}
}

func truncateForTest(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
