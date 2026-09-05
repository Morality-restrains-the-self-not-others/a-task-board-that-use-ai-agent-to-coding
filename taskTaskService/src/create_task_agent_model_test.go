package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateTaskAutoRunPersistsAgentModelToRowAndJSON 派生副本带 agent_models 创建后，
// agent_models[0].model 必须持久化到任务行并暴露在 create/GET JSON（OPT-20260825-014）。
func TestCreateTaskAutoRunPersistsAgentModelToRowAndJSON(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run with persisted model",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"agent_models":[{"provider":"openai","model":"gpt-4.1-mini"}],
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	// create 响应 JSON 直接暴露 agent_model。
	if got, _ := created["agent_model"].(string); got != "gpt-4.1-mini" {
		t.Fatalf("create response agent_model=%q want gpt-4.1-mini", created["agent_model"])
	}

	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatalf("create response missing id: %v", created)
	}

	// 任务行持久化。
	row, err := loadTask(taskID)
	if err != nil {
		t.Fatalf("loadTask: %v", err)
	}
	if row.AgentModel != "gpt-4.1-mini" {
		t.Fatalf("task row AgentModel=%q want gpt-4.1-mini", row.AgentModel)
	}

	// GET JSON（taskToJSON）同样暴露。
	got := taskToJSON(row, "t1")
	if v, _ := got["agent_model"].(string); v != "gpt-4.1-mini" {
		t.Fatalf("taskToJSON agent_model=%q want gpt-4.1-mini", got["agent_model"])
	}
}

// TestCreateTaskWithoutAgentModelKeepsRowEmpty 无 agent_models 的创建不写 agent_model（nil）。
func TestCreateTaskWithoutAgentModelKeepsRowEmpty(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run without model",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatalf("create response missing id: %v", created)
	}
	row, err := loadTask(taskID)
	if err != nil {
		t.Fatalf("loadTask: %v", err)
	}
	if row.AgentModel != "" {
		t.Fatalf("task row AgentModel=%q want empty", row.AgentModel)
	}
	if v, _ := taskToJSON(row, "t1")["agent_model"]; v != nil {
		t.Fatalf("taskToJSON agent_model=%v want nil", v)
	}
}
