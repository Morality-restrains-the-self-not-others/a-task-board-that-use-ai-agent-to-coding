package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCreateTaskAgentModelsAbsentOK(t *testing.T) {
	got, err := parseCreateTaskAgentModels(map[string]interface{}{"auto_run": true}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got %#v want nil", got)
	}
}

func TestParseCreateTaskAgentModelsIgnoredWhenNotAutoRun(t *testing.T) {
	got, err := parseCreateTaskAgentModels(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "openai", "model": "gpt-4.1"},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("fork-only must ignore agent_models, got %#v", got)
	}
}

func TestParseCreateTaskAgentModelsValidOneItem(t *testing.T) {
	got, err := parseCreateTaskAgentModels(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "openai", "model": "gpt-4.1-mini"},
		},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["provider"] != "openai" || got[0]["model"] != "gpt-4.1-mini" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseCreateTaskAgentModelsRejectsEmptyOrTwo(t *testing.T) {
	_, err := parseCreateTaskAgentModels(map[string]interface{}{
		"agent_models": []interface{}{},
	}, true)
	if err == nil {
		t.Fatal("expected error for empty list")
	}
	_, err = parseCreateTaskAgentModels(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "openai", "model": "a"},
			map[string]interface{}{"provider": "openai", "model": "b"},
		},
	}, true)
	if err == nil {
		t.Fatal("expected error for two items")
	}
	_, err = parseCreateTaskAgentModels(map[string]interface{}{
		"agent_models": []interface{}{
			map[string]interface{}{"provider": "", "model": "gpt"},
		},
	}, true)
	if err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestCreateTaskAutoRunRejectsInvalidAgentModels(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run bad models",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"agent_models":[{"provider":"openai","model":"a"},{"provider":"openai","model":"b"}],
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTaskAutoRunPassesAgentModelsToTrigger(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)
	var got autoRunTriggerParams
	prev := scheduleTaskAutoRunFn
	scheduleTaskAutoRunFn = func(p autoRunTriggerParams) {
		got = p
	}
	t.Cleanup(func() { scheduleTaskAutoRunFn = prev })

	body := `{
		"title":"Auto run with model",
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
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(got.AgentModels) != 1 {
		t.Fatalf("AgentModels=%#v", got.AgentModels)
	}
	if got.AgentModels[0]["provider"] != "openai" || got.AgentModels[0]["model"] != "gpt-4.1-mini" {
		t.Fatalf("AgentModels[0]=%#v", got.AgentModels[0])
	}
}
