package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestContainerAgentCommentCreateStreamComplete(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"
	cfg.TaskSseURL = ""

	createBody := `{
		"tenant_id":"t1","workspace_id":"ws1","task_id":"task1",
		"parent_comment_id":"human-1","installed_image_id":"img-1",
		"content":"已 @ 镜像启动"
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/container-agent-comments/", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	createRec := httptest.NewRecorder()
	handleInternalCreateContainerAgentComment(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("missing id: %v", created)
	}
	if created["run_status"] != runStatusPending {
		t.Fatalf("run_status = %v, want pending", created["run_status"])
	}

	streamBody := `{"chunk":"hello "}`
	streamReq := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/"+id+"/stream",
		strings.NewReader(streamBody))
	streamReq.Header.Set("Content-Type", "application/json")
	streamReq.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	streamRec := httptest.NewRecorder()
	handleContainerAgentStream(streamRec, streamReq, "t1", "ws1", "task1", id)
	if streamRec.Code != http.StatusOK {
		t.Fatalf("stream: expected 200, got %d: %s", streamRec.Code, streamRec.Body.String())
	}

	streamBody2 := `{"chunk":"world"}`
	streamReq2 := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/"+id+"/stream",
		strings.NewReader(streamBody2))
	streamReq2.Header.Set("Content-Type", "application/json")
	streamReq2.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	streamRec2 := httptest.NewRecorder()
	handleContainerAgentStream(streamRec2, streamReq2, "t1", "ws1", "task1", id)
	if streamRec2.Code != http.StatusOK {
		t.Fatalf("stream2: expected 200, got %d: %s", streamRec2.Code, streamRec2.Body.String())
	}

	completeReq := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/"+id+"/complete",
		strings.NewReader(`{}`))
	completeReq.Header.Set("Content-Type", "application/json")
	completeReq.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	completeRec := httptest.NewRecorder()
	handleContainerAgentComplete(completeRec, completeReq, "t1", "ws1", "task1", id)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d: %s", completeRec.Code, completeRec.Body.String())
	}

	loaded, err := loadContainerAgentComment(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.RunStatus != runStatusCompleted {
		t.Fatalf("run_status = %q, want completed", loaded.RunStatus)
	}
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "hello world" {
		t.Fatalf("assistant_response = %v", loaded.AssistantResponse)
	}
}

func TestContainerAgentCommentFail(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	createBody := `{
		"tenant_id":"t1","workspace_id":"ws1","task_id":"task1",
		"parent_comment_id":"human-2","installed_image_id":"img-2"
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/container-agent-comments/", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	createRec := httptest.NewRecorder()
	handleInternalCreateContainerAgentComment(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(createRec.Body).Decode(&created)
	id, _ := created["id"].(string)

	failReq := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/"+id+"/fail",
		strings.NewReader(`{"error":"container crashed"}`))
	failReq.Header.Set("Content-Type", "application/json")
	failReq.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	failRec := httptest.NewRecorder()
	handleContainerAgentFail(failRec, failReq, "t1", "ws1", "task1", id)
	if failRec.Code != http.StatusOK {
		t.Fatalf("fail: expected 200, got %d: %s", failRec.Code, failRec.Body.String())
	}

	loaded, err := loadContainerAgentComment(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.RunStatus != runStatusFailed {
		t.Fatalf("run_status = %q, want failed", loaded.RunStatus)
	}
	if !loaded.AssistantResponse.Valid || loaded.AssistantResponse.String != "container crashed" {
		t.Fatalf("assistant_response = %v", loaded.AssistantResponse)
	}
}

func TestValidateOneActiveRun(t *testing.T) {
	if err := ValidateOneActiveRun(0); err != nil {
		t.Fatalf("expected nil for 0 active, got %v", err)
	}
	if err := ValidateOneActiveRun(1); err == nil {
		t.Fatalf("expected error for 1 active")
	}
}

func TestContainerAgentCommentDuplicateActiveRunRejected(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	body := `{
		"tenant_id":"t1","workspace_id":"ws1","task_id":"task1",
		"parent_comment_id":"human-dup","installed_image_id":"img-1"
	}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/container-agent-comments/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
		rec := httptest.NewRecorder()
		handleInternalCreateContainerAgentComment(rec, req)
		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("first create: expected 201, got %d", rec.Code)
		}
		if i == 1 && rec.Code != http.StatusConflict {
			t.Fatalf("second create: expected 409, got %d: %s", rec.Code, rec.Body.String())
		}
	}
}

func TestContainerAgentStreamForbiddenWithoutAuth(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	c := &ContainerAgentComment{
		ID: "777", TenantID: "t1", WorkspaceID: "ws1", TaskID: "task1",
		ParentCommentID: "p1", InstalledImageID: "img-1", RunStatus: runStatusPending,
	}
	if err := insertContainerAgentComment(c); err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/777/stream",
		strings.NewReader(`{"chunk":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleContainerAgentStream(rec, req, "t1", "ws1", "task1", "777")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestPublicCreateContainerAgentCommentAcceptsContainerAccessTokenBypass(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	body := `{
		"parent_comment_id":"human-token","installed_image_id":"img-token",
		"content":"edit_run deliver"
	}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/task/task1/container-agent-comments/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Task-Test-Skip-Container-Agent-Auth", "1")
	rec := httptest.NewRecorder()
	handlePublicCreateContainerAgentComment(rec, req, "t1", "ws1", "task1")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created["id"] == nil || created["id"] == "" {
		t.Fatalf("missing id: %v", created)
	}
	if created["parent_comment_id"] != "human-token" {
		t.Fatalf("parent=%v", created["parent_comment_id"])
	}
}

func TestContainerAgentContextPackCreateAndActiveByTask(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"

	createBody := `{
		"tenant_id":"t1","workspace_id":"ws1","task_id":"task-pack",
		"parent_comment_id":"human-pack","installed_image_id":"img-pack",
		"content":"@img go",
		"context_pack":{
			"at_mention_run":{
				"parent_comment_id":"human-pack",
				"trigger_comment":{"id":"human-pack","content":"go","created_by":"u1"},
				"installed_image":{"id":"img-pack","name":"Pack"}
			},
			"comment_thread":[{"kind":"human","id":"human-pack","content":"go"}]
		}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/internal/task-ai-comment/container-agent-comments/", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	createRec := httptest.NewRecorder()
	handleInternalCreateContainerAgentComment(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	pack, _ := created["context_pack"].(map[string]interface{})
	if pack == nil {
		t.Fatalf("expected context_pack in create response, got %#v", created)
	}
	at, _ := pack["at_mention_run"].(map[string]interface{})
	if at == nil || at["run_id"] != id || at["agent_comment_id"] != id {
		t.Fatalf("at_mention_run run ids not set: %#v", at)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/active-by-task?task_id=task-pack", nil)
	getReq.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	getRec := httptest.NewRecorder()
	handleInternalActiveContainerAgentByTask(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("active-by-task: expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var active map[string]interface{}
	_ = json.NewDecoder(getRec.Body).Decode(&active)
	if active["id"] != id {
		t.Fatalf("active id=%v want %s", active["id"], id)
	}
	activePack, _ := active["context_pack"].(map[string]interface{})
	if activePack == nil {
		t.Fatal("expected context_pack on active-by-task")
	}

	patchBody := `{"run_status":"starting"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/internal/task-ai-comment/container-agent-comments/"+id, strings.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	patchRec := httptest.NewRecorder()
	handleInternalPatchContainerAgentComment(patchRec, patchReq, id)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	loaded, err := loadContainerAgentComment(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.RunStatus != runStatusStarting {
		t.Fatalf("run_status=%q want starting", loaded.RunStatus)
	}
}

func TestContainerAgentActiveByTaskNotFound(t *testing.T) {
	setupTestDB(t)
	cfg.TaskAICommentInternalSecret = "test-secret"
	req := httptest.NewRequest(http.MethodGet, "/api/internal/task-ai-comment/container-agent-comments/active-by-task?task_id=none", nil)
	req.Header.Set("X-TaskAIComment-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalActiveContainerAgentByTask(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPublishContainerAgentStreamSSE(t *testing.T) {
	var mu sync.Mutex
	var captured map[string]interface{}
	sseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/publish" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		mu.Lock()
		defer mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer sseSrv.Close()
	cfg.TaskSseURL = sseSrv.URL
	cfg.TaskSseSecret = "sse-secret"

	publishContainerAgentStreamSSE(nil, "task1", "agent-1", "chunk", "hi", "trace-1")

	mu.Lock()
	defer mu.Unlock()
	if captured == nil {
		t.Fatalf("no SSE payload captured")
	}
	statusData, _ := captured["status_data"].(map[string]interface{})
	if statusData == nil || statusData["status"] != "container_agent_stream" {
		t.Fatalf("status_data = %v", statusData)
	}
	if statusData["phase"] != "chunk" || statusData["message"] != "hi" {
		t.Fatalf("phase/message = %v / %v", statusData["phase"], statusData["message"])
	}
}
