package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInternalListAICommentsByTask(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	c := &AIComment{
		ID: "ai-1", Content: "hello", TaskID: "task1",
		TenantID: "t1", WorkspaceID: "ws1", CreatedByID: "u1",
		AssistantResponse: sql.NullString{String: "assistant text here", Valid: true},
	}
	if err := insertComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/task1/ai-comments?full=1", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0]["content"] != "hello" {
		t.Fatalf("unexpected: %v", out)
	}
}

func TestListAICommentsPaginationPreview(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	long := strings.Repeat("a", 3000)
	c := &AIComment{
		ID: "ai-page-1", Content: "q", TaskID: "task-page",
		TenantID: "t1", WorkspaceID: "ws1", CreatedByID: "u1",
		AssistantResponse: sql.NullString{String: long, Valid: true},
	}
	if err := insertComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/task-page/ai-comments?limit=10&preview=1", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	results, _ := page["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("results=%v", page)
	}
	item, _ := results[0].(map[string]interface{})
	if item["assistant_response"] != nil {
		t.Fatalf("expected preview mode to omit full assistant_response")
	}
	if item["assistant_preview"] == nil {
		t.Fatalf("missing assistant_preview")
	}
}
