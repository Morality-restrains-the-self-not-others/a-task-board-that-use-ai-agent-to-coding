package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestInternalCommentCreatedBy verifies the comment author lookup API used by
// task-credential-service HTTP fallback (OPT-20260820-021).
func TestInternalCommentCreatedBy(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	createBody := `{"title":"Author","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,created_at) VALUES(?,?,?,?,?,NOW())`,
		"cmt_author", taskID, "u77", "hello", "",
	); err != nil {
		t.Fatalf("insert comment: %v", err)
	}

	// Found: returns the comment author user id.
	reqFound := httptest.NewRequest(http.MethodGet, "/api/internal/comments/cmt_author/created-by", nil)
	recFound := httptest.NewRecorder()
	handleInternalCommentCreatedBy(recFound, reqFound, "cmt_author")
	if recFound.Code != http.StatusOK {
		t.Fatalf("created-by found: %d %s", recFound.Code, recFound.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(recFound.Body).Decode(&body)
	if body["comment_id"] != "cmt_author" || body["user_id"] != "u77" {
		t.Fatalf("created-by body=%v want comment_id=cmt_author user_id=u77", body)
	}

	// Missing comment → 404.
	reqMiss := httptest.NewRequest(http.MethodGet, "/api/internal/comments/cmt_missing/created-by", nil)
	recMiss := httptest.NewRecorder()
	handleInternalCommentCreatedBy(recMiss, reqMiss, "cmt_missing")
	if recMiss.Code != http.StatusNotFound {
		t.Fatalf("missing comment: expected 404, got %d", recMiss.Code)
	}

	// Empty comment id → 400.
	reqEmpty := httptest.NewRequest(http.MethodGet, "/api/internal/comments//created-by", nil)
	recEmpty := httptest.NewRecorder()
	handleInternalCommentCreatedBy(recEmpty, reqEmpty, "")
	if recEmpty.Code != http.StatusBadRequest {
		t.Fatalf("empty comment id: expected 400, got %d", recEmpty.Code)
	}
}
