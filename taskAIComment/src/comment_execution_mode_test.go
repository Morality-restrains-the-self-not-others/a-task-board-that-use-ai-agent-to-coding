package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeCommentExecutionMode(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"", executionModeWaitPrevious, false},
		{"wait_previous", executionModeWaitPrevious, false},
		{"independent", executionModeIndependent, false},
		{"bad", "", true},
	}
	for _, tc := range tests {
		got, err := normalizeCommentExecutionMode(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("raw=%q expected error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("raw=%q unexpected error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("raw=%q got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}

func TestAICommentExecutionModeCreateListPatch(t *testing.T) {
	setupTestDB(t)
	startMockTaskService(t)
	startMockAuthService(t)
	startMockCloudService(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	postBody := `{"content":"AI with mode","execution_mode":"independent"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("X-Auth-Tenant-Id", "t1")
	postReq.Header.Set("X-Auth-User-Id", "u1")
	postReq.Header.Set("X-Task-Test-Skip-AI-Comment-Validate", "1")
	postRec := httptest.NewRecorder()
	handleCreateAIComment(postRec, postReq, "t1", "ws1", "task1", "u1")
	if postRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d: %s", postRec.Code, postRec.Body.String())
	}
	var meta map[string]interface{}
	if err := json.NewDecoder(postRec.Body).Decode(&meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	commentID, _ := meta["id"].(string)
	if commentID == "" {
		t.Fatal("expected comment id")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleListAIComments(listRec, listReq, "t1", "ws1", "task1", "u1")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(list))
	}
	if list[0]["execution_mode"] != executionModeIndependent {
		t.Fatalf("list execution_mode=%v", list[0]["execution_mode"])
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/"+commentID+"/", strings.NewReader(`{"execution_mode":"wait_previous"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-Auth-Tenant-Id", "t1")
	patchReq.Header.Set("X-Auth-User-Id", "u1")
	patchRec := httptest.NewRecorder()
	handlePatchAIComment(patchRec, patchReq, "t1", "ws1", "task1", commentID, "u1")
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	var patched map[string]interface{}
	if err := json.NewDecoder(patchRec.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patched["execution_mode"] != executionModeWaitPrevious {
		t.Fatalf("patch execution_mode=%v", patched["execution_mode"])
	}

	loaded, err := loadComment(commentID)
	if err != nil {
		t.Fatalf("load comment: %v", err)
	}
	if loaded.ExecutionMode != executionModeWaitPrevious {
		t.Fatalf("db execution_mode=%q", loaded.ExecutionMode)
	}
}

func TestAICommentExecutionModeCreateDefault(t *testing.T) {
	setupTestDB(t)
	startMockTaskService(t)
	startMockAuthService(t)
	startMockCloudService(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	postBody := `{"content":"default AI"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("X-Auth-Tenant-Id", "t1")
	postReq.Header.Set("X-Auth-User-Id", "u1")
	postReq.Header.Set("X-Task-Test-Skip-AI-Comment-Validate", "1")
	postRec := httptest.NewRecorder()
	handleCreateAIComment(postRec, postReq, "t1", "ws1", "task1", "u1")
	if postRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d: %s", postRec.Code, postRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleListAIComments(listRec, listReq, "t1", "ws1", "task1", "u1")
	var list []map[string]interface{}
	json.NewDecoder(listRec.Body).Decode(&list)
	if len(list) != 1 || list[0]["execution_mode"] != executionModeWaitPrevious {
		t.Fatalf("default execution_mode=%v", list[0]["execution_mode"])
	}
}

func TestAICommentExecutionModePatchInvalid(t *testing.T) {
	setupTestDB(t)
	startMockTaskService(t)
	c := &AIComment{
		ID: "901", Content: "x", TaskID: "task1",
		TenantID: "t1", WorkspaceID: "ws1", CreatedByID: "u1",
	}
	if err := insertComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/901/", strings.NewReader(`{"execution_mode":"nope"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	handlePatchAIComment(patchRec, patchReq, "t1", "ws1", "task1", "901", "u1")
	if patchRec.Code != http.StatusBadRequest {
		t.Fatalf("patch invalid: expected 400, got %d", patchRec.Code)
	}
}
