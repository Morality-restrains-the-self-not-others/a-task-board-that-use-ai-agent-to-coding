package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupTestCfg sets config for auth-only tests (no DB needed).
func setupTestCfg(t *testing.T) {
	t.Helper()
	cfg.TaskAICommentInternalSecret = "test-secret"
}

// ── handleInternalRoutes dispatch tests (no DB needed) ──

func TestInternalRoutesDispatchImport(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	payload := `{"comments":[{"id":"imp-d-1","content":"test","task_id":"t1","tenant_id":"t1","workspace_id":"w1","created_by_id":"u1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalRoutesDispatchImportSecretRejected(t *testing.T) {
	setupTestCfg(t)

	payload := `{"comments":[{"id":"x","content":"t","task_id":"t1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInternalRoutesDispatchNotFound(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/unknown-path", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalRoutesDispatchTasksAICommentsSecretRejected(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/t1/ai-comments", nil)
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing secret, got %d", rec.Code)
	}
}

// ── handleInternalTaskCommentLists (validation before DB) ──

func TestInternalTaskCommentListsMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/tasks/t1/ai-comments", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalTaskCommentLists(rec, req, "tasks/t1/ai-comments")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestInternalTaskCommentListsForbiddenNoSecret(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/t1/ai-comments", nil)
	rec := httptest.NewRecorder()
	handleInternalTaskCommentLists(rec, req, "tasks/t1/ai-comments")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInternalTaskCommentListsInvalidPath(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/t1", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalTaskCommentLists(rec, req, "tasks/t1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for short path, got %d", rec.Code)
	}
}

func TestInternalTaskCommentListsUnknownKind(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/t1/unknown-kind", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalTaskCommentLists(rec, req, "tasks/t1/unknown-kind")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown kind, got %d", rec.Code)
	}
}

// ── handleInternalPatchContainerAgentComment (validation before DB) ──

func TestInternalPatchContainerAgentCommentMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/c1", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestInternalPatchContainerAgentCommentForbiddenNoSecret(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/c1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInternalPatchContainerAgentCommentEmptyID(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d", rec.Code)
	}
}

func TestInternalPatchContainerAgentCommentInvalidJSON(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/c1", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rec.Code)
	}
}

func TestInternalPatchContainerAgentCommentUnsupportedRunStatus(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"invalid_status"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/c1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported run_status, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalPatchContainerAgentCommentEmptyRunStatus(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/c1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty run_status, got %d", rec.Code)
	}
}

func TestInternalPatchContainerAgentCommentNoFields(t *testing.T) {
	setupTestCfg(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/c1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "c1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when no fields, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ── handleInternalPatchContainerAgentStatusByParent (validation before DB) ──

func TestInternalPatchStatusByParentMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/by-parent/p1/status", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "p1")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestInternalPatchStatusByParentForbiddenNoSecret(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/p1/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "p1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInternalPatchStatusByParentEmptyParentID(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/x/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "  ")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty parent_comment_id, got %d", rec.Code)
	}
}

func TestInternalPatchStatusByParentMissingRunStatus(t *testing.T) {
	setupTestCfg(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/p1/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "p1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing run_status, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalPatchStatusByParentUnsupportedStatus(t *testing.T) {
	setupTestCfg(t)

	body := `{"run_status":"completed"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/p1/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "p1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported status, got %d", rec.Code)
	}
}

func TestInternalPatchStatusByParentInvalidJSON(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/p1/status", strings.NewReader("bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentStatusByParent(rec, req, "p1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rec.Code)
	}
}

// ── handlePatchAssistantResponse (validation before DB) ──

func TestPatchAssistantResponseMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/1/assistant-response", strings.NewReader(`{"assistant_response":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "1")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST, got %d", rec.Code)
	}
}

func TestPatchAssistantResponseEmptyID(t *testing.T) {
	setupTestCfg(t)

	body := `{"assistant_response":"text"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment//assistant-response", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty id, got %d", rec.Code)
	}
}

func TestPatchAssistantResponseMissingText(t *testing.T) {
	setupTestCfg(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/1/assistant-response", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing assistant_response, got %d", rec.Code)
	}
}

func TestPatchAssistantResponseForbiddenNoSecret(t *testing.T) {
	setupTestCfg(t)

	body := `{"assistant_response":"text"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/1/assistant-response", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestPatchAssistantResponseInvalidJSON(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/1/assistant-response", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handlePatchAssistantResponse(rec, req, "1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rec.Code)
	}
}

// ── handleImportAIComments (validation before DB) ──

func TestImportAICommentsMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/import", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestImportAICommentsSecretRejected(t *testing.T) {
	setupTestCfg(t)

	payload := `{"comments":[{"id":"1","content":"t","task_id":"t1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestImportAICommentsInvalidJSON(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rec.Code)
	}
}

func TestImportAICommentsEmptyComments(t *testing.T) {
	setupTestCfg(t)

	payload := `{"comments":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty comments, got %d", rec.Code)
	}
}

// ── Tests requiring DB (compile-checked, run with MySQL) ──

func TestInternalRoutesDispatchTasksAICommentsWithData(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	c := &AIComment{
		ID: "c-disp", Content: "hello", TaskID: "task-disp",
		TenantID: "t1", WorkspaceID: "w1", CreatedByID: "u1",
	}
	if err := insertComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/tasks/task-disp/ai-comments", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Fatal("expected non-empty list")
	}
}

func TestInternalRoutesDispatchPatchStatusByParentWithData(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	cac := &ContainerAgentComment{
		ID: "cac-parent", TenantID: "t1", WorkspaceID: "w1", TaskID: "task-ps",
		ParentCommentID: "parent-1", InstalledImageID: "img-1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(cac); err != nil {
		t.Fatalf("insert: %v", err)
	}

	body := `{"run_status":"running"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/by-parent/parent-1/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loaded, _ := loadContainerAgentComment("cac-parent")
	if loaded.RunStatus != runStatusRunning {
		t.Fatalf("run_status=%q want running", loaded.RunStatus)
	}
}

func TestInternalPatchContainerAgentCommentRunStatusSuccess(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	cac := &ContainerAgentComment{
		ID: "cac-rs", TenantID: "t1", WorkspaceID: "w1", TaskID: "task-rs",
		ParentCommentID: "p1", InstalledImageID: "img-1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(cac); err != nil {
		t.Fatalf("insert: %v", err)
	}

	for _, status := range []string{runStatusStarting, runStatusRunning, runStatusStreaming} {
		body := `{"run_status":"` + status + `"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/cac-rs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
		rec := httptest.NewRecorder()
		handleInternalPatchContainerAgentComment(rec, req, "cac-rs")
		if rec.Code != http.StatusOK {
			t.Fatalf("patch to %s: expected 200, got %d body=%s", status, rec.Code, rec.Body.String())
		}

		loaded, _ := loadContainerAgentComment("cac-rs")
		if loaded.RunStatus != status {
			t.Fatalf("run_status=%q want %s", loaded.RunStatus, status)
		}
	}
}

func TestInternalPatchContainerAgentCommentBadContextPackType(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	cac := &ContainerAgentComment{
		ID: "cac-bcp", TenantID: "t1", WorkspaceID: "w1", TaskID: "task-bcp",
		ParentCommentID: "p1", InstalledImageID: "img-1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(cac); err != nil {
		t.Fatalf("insert: %v", err)
	}

	body := `{"context_pack":"not an object"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/cac-bcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(rec, req, "cac-bcp")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-object context_pack, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestImportAICommentsRowsAlias(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	payload := `{"rows":[{"id":"row-1","content":"via rows","task_id":"t1","tenant_id":"t1","workspace_id":"w1","created_by_id":"u1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for rows alias, got %d body=%s", rec.Code, rec.Body.String())
	}

	loaded, _ := loadComment("row-1")
	if loaded.Content != "via rows" {
		t.Fatalf("content=%q", loaded.Content)
	}
}

func TestImportAICommentsWithAssistantResponse(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	payload := `{"comments":[{"id":"ar-1","content":"has ar","task_id":"t1","tenant_id":"t1","workspace_id":"w1","created_by_id":"u1","assistant_response":"AI said hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loaded, _ := loadComment("ar-1")
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "AI said hi" {
		t.Fatalf("assistant_response=%v", loaded.AssistantResponse)
	}
}

func TestImportAICommentsFallbackTaskField(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	payload := `{"comments":[{"id":"fb-1","content":"fallback","task":"task-fb","tenant_id":"t1","workspace_id":"w1","created_by":"u1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loaded, _ := loadComment("fb-1")
	if loaded.TaskID != "task-fb" {
		t.Fatalf("task_id=%q want task-fb", loaded.TaskID)
	}
}

func TestImportAICommentsWithTimestamps(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)

	payload := `{"comments":[{"id":"ts-1","content":"timestamped","task_id":"t1","tenant_id":"t1","workspace_id":"w1","created_by_id":"u1","created_at":"2025-01-15T10:30:00Z","updated_at":"2025-01-15T11:00:00Z"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/import", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleImportAIComments(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	loaded, _ := loadComment("ts-1")
	expected := "2025-01-15T10:30:00Z"
	if loaded.CreatedAt.UTC().Format("2006-01-02T15:04:05Z") != expected {
		t.Fatalf("created_at=%v want %s", loaded.CreatedAt.UTC(), expected)
	}
}

func TestGetContainerAgentCommentSuccess(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)
	startMockTaskService(t)

	cac := &ContainerAgentComment{
		ID: "cac-get", TenantID: "t1", WorkspaceID: "w1", TaskID: "task-get",
		ParentCommentID: "p-get", InstalledImageID: "img-get",
		RunStatus: runStatusCompleted,
		Content: sql.NullString{String: "hello from agent", Valid: true},
		AssistantResponse: sql.NullString{String: "result", Valid: true},
	}
	if err := insertContainerAgentComment(cac); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/w1/task/task-get/container-agent-comments/cac-get", nil)
	req.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	rec := httptest.NewRecorder()
	handleGetContainerAgentComment(rec, req, "t1", "w1", "task-get", "cac-get")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var result map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	if result["id"] != "cac-get" {
		t.Fatalf("id=%v want cac-get", result["id"])
	}
	if result["run_status"] != runStatusCompleted {
		t.Fatalf("run_status=%v", result["run_status"])
	}
}

func TestListContainerAgentCommentsWithData(t *testing.T) {
	setupTestDB(t)
	setupTestCfg(t)
	startMockTaskService(t)

	for i := 0; i < 3; i++ {
		cac := &ContainerAgentComment{
			ID: "lc-" + string(rune('a'+i)), TenantID: "t1", WorkspaceID: "w1",
			TaskID: "task-lcd", ParentCommentID: "p1", InstalledImageID: "img-1",
			RunStatus: runStatusCompleted,
		}
		_ = insertContainerAgentComment(cac)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/w1/task/task-lcd/container-agent-comments/", nil)
	req.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	rec := httptest.NewRecorder()
	handleListContainerAgentComments(rec, req, "t1", "w1", "task-lcd")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(list))
	}
}

func TestInternalActiveContainerAgentByTaskMissingTaskID(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/active-by-task", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalActiveContainerAgentByTask(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing task_id, got %d", rec.Code)
	}
}

func TestInternalActiveContainerAgentByTaskForbidden(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/active-by-task?task_id=t1", nil)
	rec := httptest.NewRecorder()
	handleInternalActiveContainerAgentByTask(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestInternalActiveContainerAgentByTaskMethodNotAllowed(t *testing.T) {
	setupTestCfg(t)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/container-agent-comments/active-by-task", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalActiveContainerAgentByTask(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
