package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260830-004: GET 任务详情 403 须用读语义，不得复用「修改」文案。
func TestGetTaskForbiddenMessageIsReadNotWrite(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") && !strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"user_id": "u-real-member"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	createBody := `{"title":"Forbidden read copy","workspace_id":"ws1"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCreate.Header.Set("X-Auth-User-Id", "u-real-member")
	recCreate := httptest.NewRecorder()
	handleTaskRoutes(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(recCreate.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	reqDenied := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/", nil)
	reqDenied.Header.Set("X-Auth-Tenant-Id", "t1")
	reqDenied.Header.Set("X-Auth-User-Id", "u-stranger")
	reqDenied.Header.Set("X-Task-Id", taskID)
	recDenied := httptest.NewRecorder()
	handleGetTaskByID(recDenied, reqDenied)
	if recDenied.Code != http.StatusForbidden {
		t.Fatalf("stranger GET: expected 403, got %d: %s", recDenied.Code, recDenied.Body.String())
	}
	body := recDenied.Body.String()
	if strings.Contains(body, "修改") {
		t.Fatalf("GET 403 body must not contain 修改: %s", body)
	}
	if !strings.Contains(body, "您没有权限查看此任务") {
		t.Fatalf("GET 403 body want 您没有权限查看此任务, got: %s", body)
	}

	reqNested := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqNested.Header.Set("X-Auth-Tenant-Id", "t1")
	reqNested.Header.Set("X-Auth-User-Id", "u-stranger")
	recNested := httptest.NewRecorder()
	handleGetTask(recNested, reqNested, "t1", "u-stranger", taskID)
	if recNested.Code != http.StatusForbidden {
		t.Fatalf("nested GET: expected 403, got %d: %s", recNested.Code, recNested.Body.String())
	}
	nestedBody := recNested.Body.String()
	if strings.Contains(nestedBody, "修改") {
		t.Fatalf("nested GET 403 body must not contain 修改: %s", nestedBody)
	}
	if !strings.Contains(nestedBody, "您没有权限查看此任务") {
		t.Fatalf("nested GET 403 body want 查看文案, got: %s", nestedBody)
	}
}
