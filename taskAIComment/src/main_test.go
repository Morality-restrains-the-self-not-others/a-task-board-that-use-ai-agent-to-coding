package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbload "dbload"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	testDSN, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-ai-comment", repoRoot(), "dataMigrate/taskAIComment",
		func(dsn string) error {
			if err := openDB(dsn); err != nil {
				return err
			}
			defer db.Close()
			return runDataMigrate(repoRoot())
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
}

func startMockTaskService(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/todos/") {
			// Extract task ID from path like /api/tenant/t1/workspace/ws1/todos/task1/
			parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
			taskID := "task1"
			if len(parts) > 0 {
				taskID = parts[len(parts)-1]
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           taskID,
				"tenant_id":    "t1",
				"workspace_id": "ws1",
				"title":        "Test Task",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.TaskServiceURL = srv.URL
}

func startMockAuthService(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/users/u1") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "u1", "username": "testuser", "email": "u1@test.com",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.TaskAuthURL = srv.URL
}

func startMockCloudService(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/cloud-server-config/validate-ai-comment-post/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
			return
		}
		if r.URL.Path == "/api/internal/cloud-server-config/lookup/" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"company_id": "t1", "workspace_id": "ws1", "task_id": "task1",
				"server_url": "http://127.0.0.1:1/",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.TaskCloudServiceURL = srv.URL
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCreateAndListAIComments(t *testing.T) {
	setupTestDB(t)
	startMockTaskService(t)
	startMockAuthService(t)
	startMockCloudService(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	postBody := `{"content":"请 AI 总结任务"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/task-detail/task1/ai-comments/", strings.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("X-Auth-Tenant-Id", "t1")
	postReq.Header.Set("X-Auth-User-Id", "u1")
	postRec := httptest.NewRecorder()
	handleCreateAIComment(postRec, postReq, "t1", "ws1", "task1", "u1")
	if postRec.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d: %s", postRec.Code, postRec.Body.String())
	}
	var meta map[string]interface{}
	if err := json.NewDecoder(postRec.Body).Decode(&meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta["kind"] != "ai_comment_created" {
		t.Fatalf("kind = %v, want ai_comment_created", meta["kind"])
	}
	if meta["todo"] != "task1" {
		t.Fatalf("todo = %v, want task1", meta["todo"])
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
	if list[0]["content"] != "请 AI 总结任务" {
		t.Fatalf("content = %v", list[0]["content"])
	}
	createdBy, _ := list[0]["created_by"].(map[string]interface{})
	if createdBy == nil || createdBy["username"] != "testuser" {
		t.Fatalf("created_by = %v", list[0]["created_by"])
	}
}

func TestPatchAssistantResponse(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	c := &AIComment{
		ID: "900", Content: "hello", TaskID: "task1",
		TenantID: "t1", WorkspaceID: "ws1", CreatedByID: "u1",
	}
	if err := insertComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	body := `{"assistant_response":"AI reply text"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/900/assistant-response/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "900")
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	loaded, err := loadComment("900")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "AI reply text" {
		t.Fatalf("assistant_response = %v", loaded.AssistantResponse)
	}
}

func TestImportAIComments(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	payload := `{"comments":[{"id":"1001","content":"imported","task_id":"task1","tenant_id":"t1","workspace_id":"ws1","created_by_id":"u1","assistant_response":"done"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	loaded, err := loadComment("1001")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Content != "imported" {
		t.Fatalf("content = %q", loaded.Content)
	}
}

func TestPatchAssistantResponseForbidden(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/1/assistant-response/", strings.NewReader(`{"assistant_response":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGenerateSnowflakeID(t *testing.T) {
	a := GenerateID()
	b := GenerateID()
	if a == b {
		t.Fatalf("expected distinct snowflake ids, got %d", a)
	}
	if a <= 0 || b <= 0 {
		t.Fatalf("snowflake ids must be positive: %d %d", a, b)
	}
}
