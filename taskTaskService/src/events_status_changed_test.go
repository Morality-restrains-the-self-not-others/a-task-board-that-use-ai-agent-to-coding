package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPublishTaskStatusChangedOnColumnChange(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	var mu sync.Mutex
	var calls []map[string]interface{}
	prev := publishTaskStatusChangedFn
	publishTaskStatusChangedFn = func(ctx context.Context, tenantID, workspaceID, taskID string, prevColumnID, columnID string, prevCompleted, completed bool, progressColumnName string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, map[string]interface{}{
			"tenant_id":            tenantID,
			"workspace_id":         workspaceID,
			"task_id":              taskID,
			"prev_column":          prevColumnID,
			"column":               columnID,
			"prev_completed":       prevCompleted,
			"completed":            completed,
			"progress_column_name": progressColumnName,
		})
		return nil
	}
	t.Cleanup(func() { publishTaskStatusChangedFn = prev })

	createBody := `{"title":"StatusEvt","workspace_id":"ws1","progress_column_id":"col-todo"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("missing task id")
	}

	patch := `{"progress_column_id":"col-done","title":"StatusEvt"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(patch))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	req2.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req2.Header.Set("X-Task-Test-Progress-Column-Name", "已完成")
	req2.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-done")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(calls))
	}
	if calls[0]["column"] != "col-done" {
		t.Fatalf("column=%v", calls[0]["column"])
	}
	if calls[0]["prev_column"] != "col-todo" {
		t.Fatalf("prev_column=%v", calls[0]["prev_column"])
	}
	if calls[0]["progress_column_name"] != "已完成" {
		t.Fatalf("progress_column_name=%v", calls[0]["progress_column_name"])
	}
}

func TestNoPublishTaskStatusChangedWhenUnchanged(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	var calls int
	prev := publishTaskStatusChangedFn
	publishTaskStatusChangedFn = func(ctx context.Context, tenantID, workspaceID, taskID string, prevColumnID, columnID string, prevCompleted, completed bool, progressColumnName string) error {
		calls++
		return nil
	}
	t.Cleanup(func() { publishTaskStatusChangedFn = prev })

	createBody := `{"title":"NoChange","workspace_id":"ws1","feature_params_source":"company","progress_column_id":"col-a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	patch := `{"title":"NoChange Renamed","progress_column_id":"col-a"}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(patch))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	req2.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req2.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-a")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	if calls != 0 {
		t.Fatalf("expected no publish, got %d", calls)
	}
}

func TestPublishTaskStatusChangedOnCompletedFlag(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	var calls int
	prev := publishTaskStatusChangedFn
	publishTaskStatusChangedFn = func(ctx context.Context, tenantID, workspaceID, taskID string, prevColumnID, columnID string, prevCompleted, completed bool, progressColumnName string) error {
		calls++
		if prevCompleted || !completed {
			t.Errorf("expected completed false->true, got %v->%v", prevCompleted, completed)
		}
		return nil
	}
	t.Cleanup(func() { publishTaskStatusChangedFn = prev })

	createBody := `{"title":"DoneFlag","workspace_id":"ws1","feature_params_source":"company"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	patch := `{"completed":true}`
	req2 := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(patch))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	req2.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	if calls != 1 {
		t.Fatalf("expected 1 publish, got %d", calls)
	}
}

func TestPublishTaskStatusChangedOnSwitch(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	var mu sync.Mutex
	var calls []map[string]interface{}
	prev := publishTaskStatusChangedFn
	publishTaskStatusChangedFn = func(ctx context.Context, tenantID, workspaceID, taskID string, prevColumnID, columnID string, prevCompleted, completed bool, progressColumnName string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, map[string]interface{}{
			"prev_completed": prevCompleted,
			"completed":      completed,
			"task_id":        taskID,
		})
		return nil
	}
	t.Cleanup(func() { publishTaskStatusChangedFn = prev })

	createBody := `{"title":"SwitchEvt","workspace_id":"ws1","feature_params_source":"company"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/switch", strings.NewReader(`{"completed":true}`))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("switch: %d %s", rec2.Code, rec2.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("expected 1 publish on switch, got %d", len(calls))
	}
	if calls[0]["prev_completed"] != false || calls[0]["completed"] != true {
		t.Fatalf("calls[0]=%v", calls[0])
	}

	// No-op when completed unchanged
	req3 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/switch", strings.NewReader(`{"completed":true}`))
	req3.Header.Set("X-Auth-Tenant-Id", "t1")
	req3.Header.Set("X-Auth-User-Id", "u1")
	rec3 := httptest.NewRecorder()
	handleTaskRoutes(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("switch noop: %d %s", rec3.Code, rec3.Body.String())
	}
	if len(calls) != 1 {
		t.Fatalf("expected still 1 publish after noop switch, got %d", len(calls))
	}
}

func TestPublishDomainEventSkipsWithoutKafka(t *testing.T) {
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })
	if err := publishDomainEvent(context.Background(), "TASK_STATUS_CHANGED", map[string]interface{}{"task_id": "x"}, "x"); err != nil {
		t.Fatalf("expected no-op nil, got %v", err)
	}
}
