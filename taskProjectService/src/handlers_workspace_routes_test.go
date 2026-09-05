package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateWorkspace(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"My Workspace","description":"A workspace"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateWorkspace(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create workspace: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var ws map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&ws)
	if ws["name"] != "My Workspace" {
		t.Errorf("expected name=My Workspace, got %v", ws["name"])
	}
}

func TestListWorkspacesMineFilter(t *testing.T) {
	setupTestDB(t)

	create := func(name string) string {
		t.Helper()
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCreateWorkspace(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create %q: %d %s", name, rec.Code, rec.Body.String())
		}
		var ws map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&ws)
		id, _ := ws["id"].(string)
		if id == "" {
			t.Fatalf("missing workspace id for %q", name)
		}
		return id
	}
	openID := create("Open WS")
	lockedID := create("Locked WS")
	otherID := create("Other Locked")

	grant := func(wsID, userID string) {
		t.Helper()
		body := `{"workspace_id":"` + wsID + `","user_id":"` + userID + `","permission":"member"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace-access/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleWorkspaceAccess(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("grant %s→%s: %d %s", userID, wsID, rec.Code, rec.Body.String())
		}
	}
	grant(lockedID, "u1")
	grant(otherID, "u2")

	req := httptest.NewRequest(http.MethodGet, "/api/projects/workspaces/tenant_id/t1/?mine=1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleListWorkspaces(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mine list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var list []map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&list)
	got := map[string]bool{}
	for _, ws := range list {
		id, _ := ws["id"].(string)
		got[id] = true
	}
	if !got[openID] {
		t.Fatalf("expected open workspace %s in mine list", openID)
	}
	if !got[lockedID] {
		t.Fatalf("expected granted workspace %s in mine list", lockedID)
	}
	if got[otherID] {
		t.Fatalf("did not expect other user's locked workspace %s", otherID)
	}
}

func TestWorkspaceAccess(t *testing.T) {
	setupTestDB(t)

	// Create access
	body := `{"workspace_id":"ws-1","user_id":"u1","permission":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace-access/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleWorkspaceAccess(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create access: expected 201, got %d", rec.Code)
	}

	// List access
	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace-access/", nil)
	rec2 := httptest.NewRecorder()
	handleWorkspaceAccess(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list access: expected 200, got %d", rec2.Code)
	}

	// Remove access
	body3 := `{"workspace_id":"ws-1","user_id":"u1"}`
	req3 := httptest.NewRequest(http.MethodDelete, "/api/tenant/t1/workspace-access/", strings.NewReader(body3))
	rec3 := httptest.NewRecorder()
	handleWorkspaceAccess(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("remove access: expected 200, got %d", rec3.Code)
	}
}

func TestBatchDeleteProjects(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"Del A"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pid := created["id"].(string)

	body2 := `{"name":"Del B"}`
	reqB := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body2))
	reqB.Header.Set("X-Auth-Tenant-Id", "t1")
	recB := httptest.NewRecorder()
	handleCreateProject(recB, reqB)
	var createdB map[string]interface{}
	json.NewDecoder(recB.Body).Decode(&createdB)
	pidB := createdB["id"].(string)

	delBody := `{"project_ids":["` + pid + `","` + pidB + `","999999"]}`
	reqDel := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/batch-delete/", strings.NewReader(delBody))
	reqDel.Header.Set("X-Auth-Tenant-Id", "t1")
	recDel := httptest.NewRecorder()
	handleBatchDeleteProjects(recDel, reqDel, "t1")

	if recDel.Code != http.StatusOK {
		t.Fatalf("batch delete: expected 200, got %d: %s", recDel.Code, recDel.Body.String())
	}
	var result map[string]interface{}
	json.NewDecoder(recDel.Body).Decode(&result)
	deleted := result["deleted"].([]interface{})
	errs := result["errors"].([]interface{})
	if len(deleted) != 2 {
		t.Errorf("expected 2 deleted, got %d", len(deleted))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestBatchGetProjectsByIDs(t *testing.T) {
	setupTestDB(t)

	create := func(name string) string {
		t.Helper()
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCreateProject(rec, req)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("create %s: %d %s", name, rec.Code, rec.Body.String())
		}
		var created map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&created)
		id, _ := created["id"].(string)
		if id == "" {
			t.Fatalf("missing id for %s body=%v", name, created)
		}
		return id
	}
	a := create("Batch A")
	b := create("Batch B")

	payload := `{"ids":["` + a + `","` + b + `","missing_proj"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/batch-get/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleBatchGetProjects(rec, req, "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("batch-get: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	projects, _ := body["projects"].([]interface{})
	if len(projects) != 2 {
		t.Fatalf("want 2 projects, got %#v", body)
	}
	names := map[string]bool{}
	for _, raw := range projects {
		row := raw.(map[string]interface{})
		names[fmt.Sprint(row["name"])] = true
	}
	if !names["Batch A"] || !names["Batch B"] {
		t.Fatalf("missing names %#v", names)
	}
}

func TestBatchGetProjectsLiteSkipsDetailRepos(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"Lite Proj","git_repos":["https://git.example/a.git"]}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	recCreate := httptest.NewRecorder()
	handleCreateProject(recCreate, reqCreate)
	var created map[string]interface{}
	json.NewDecoder(recCreate.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("create failed: %s", recCreate.Body.String())
	}

	payload := `{"ids":["` + pid + `"],"lite":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/batch-get/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleBatchGetProjects(rec, req, "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("lite batch-get: %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	projects, _ := resp["projects"].([]interface{})
	if len(projects) != 1 {
		t.Fatalf("want 1 project, got %#v", resp)
	}
	row := projects[0].(map[string]interface{})
	if fmt.Sprint(row["name"]) != "Lite Proj" {
		t.Fatalf("name=%v", row["name"])
	}
	// lite response uses scanProjectRow only — no git_repos enrichment key required
	if _, ok := row["git_repos"]; ok {
		t.Fatalf("lite should not include git_repos from detail, got %#v", row)
	}
}

func TestBatchGetWorkspacesByIDs(t *testing.T) {
	setupTestDB(t)

	createWS := func(name string) string {
		t.Helper()
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCreateWorkspace(rec, req)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("create ws %s: %d %s", name, rec.Code, rec.Body.String())
		}
		var created map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&created)
		id, _ := created["id"].(string)
		if id == "" {
			t.Fatalf("missing ws id for %s", name)
		}
		return id
	}
	a := createWS("WS Alpha")
	b := createWS("WS Beta")

	payload := `{"ids":["` + a + `","` + b + `","missing_ws"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/batch-get/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u-test")
	rec := httptest.NewRecorder()
	handleBatchGetWorkspaces(rec, req, "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("ws batch-get: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	workspaces, _ := body["workspaces"].([]interface{})
	if len(workspaces) != 2 {
		t.Fatalf("want 2 workspaces, got %#v", body)
	}
	names := map[string]bool{}
	for _, raw := range workspaces {
		row := raw.(map[string]interface{})
		names[fmt.Sprint(row["name"])] = true
	}
	if !names["WS Alpha"] || !names["WS Beta"] {
		t.Fatalf("missing names %#v", names)
	}
}

func TestSwitchWorkspace(t *testing.T) {
	setupTestDB(t)

	wsBody := `{"name":"Switch WS"}`
	reqWS := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(wsBody))
	reqWS.Header.Set("X-Auth-Tenant-Id", "t1")
	recWS := httptest.NewRecorder()
	handleCreateWorkspace(recWS, reqWS)
	var ws map[string]interface{}
	json.NewDecoder(recWS.Body).Decode(&ws)
	wsID := ws["id"].(string)

	switchBody := `{"workspace_id":"` + wsID + `"}`
	reqSw := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/switch/", strings.NewReader(switchBody))
	reqSw.Header.Set("X-Auth-Tenant-Id", "t1")
	reqSw.Header.Set("X-Auth-User-Id", "u1")
	recSw := httptest.NewRecorder()
	handleSwitchWorkspace(recSw, reqSw, "t1")

	if recSw.Code != http.StatusOK {
		t.Fatalf("switch: expected 200, got %d: %s", recSw.Code, recSw.Body.String())
	}
	var swResult map[string]interface{}
	json.NewDecoder(recSw.Body).Decode(&swResult)
	if swResult["success"] != true {
		t.Errorf("expected success=true")
	}
}

func TestProjectActionRouting(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/batch-delete/tenant_id/t1/", strings.NewReader(`{"project_ids":[]}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty batch-delete, got %d", rec.Code)
	}
}

// TestSwitchWorkspaceViaConventionRoute — 回归 OPT-20260808-xxx：约定式 URL
// /api/projects/switch/tenant_id/{tid}/ 曾被路由到 handleLegacySwitchWorkspace，
// 其消费 body 后再委托导致二次 readJSONBody 拿到空 body，workspace_id 恒为空、
// 恒返回 400（线上"切换工作空间失败"根因）。路由级测试必须走 mux，直调 handler
// 无法暴露该缺陷。
func TestSwitchWorkspaceViaConventionRoute(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	wsBody := `{"name":"Route Switch WS"}`
	reqWS := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(wsBody))
	reqWS.Header.Set("X-Auth-Tenant-Id", "t1")
	reqWS.Header.Set("X-Auth-User-Id", "u1")
	recWS := httptest.NewRecorder()
	mux.ServeHTTP(recWS, reqWS)
	if recWS.Code != http.StatusOK {
		t.Fatalf("create ws: %d %s", recWS.Code, recWS.Body.String())
	}
	var ws map[string]interface{}
	json.NewDecoder(recWS.Body).Decode(&ws)
	wsID := ws["id"].(string)

	// 与生产前端 WorkspaceSwitcher 完全一致的 URL 形态与 body
	switchBody := `{"workspace_id":"` + wsID + `"}`
	reqSw := httptest.NewRequest(http.MethodPost, "/api/projects/switch/tenant_id/t1/", strings.NewReader(switchBody))
	reqSw.Header.Set("X-Auth-User-Id", "u1")
	recSw := httptest.NewRecorder()
	mux.ServeHTTP(recSw, reqSw)

	if recSw.Code != http.StatusOK {
		t.Fatalf("switch via convention route: expected 200, got %d: %s", recSw.Code, recSw.Body.String())
	}
	var swResult map[string]interface{}
	json.NewDecoder(recSw.Body).Decode(&swResult)
	if swResult["success"] != true {
		t.Errorf("expected success=true, got %#v", swResult)
	}
}

// TestLegacySwitchWorkspaceRoute — legacy /api/switch-workspace 同样曾二次读 body
// 恒 400，修复后须走同一核心逻辑返回 200。
func TestLegacySwitchWorkspaceRoute(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	wsBody := `{"name":"Legacy Switch WS"}`
	reqWS := httptest.NewRequest(http.MethodPost, "/api/projects/workspaces/tenant_id/t1/", strings.NewReader(wsBody))
	reqWS.Header.Set("X-Auth-Tenant-Id", "t1")
	reqWS.Header.Set("X-Auth-User-Id", "u1")
	recWS := httptest.NewRecorder()
	mux.ServeHTTP(recWS, reqWS)
	if recWS.Code != http.StatusOK {
		t.Fatalf("create ws: %d %s", recWS.Code, recWS.Body.String())
	}
	var ws map[string]interface{}
	json.NewDecoder(recWS.Body).Decode(&ws)
	wsID := ws["id"].(string)

	reqSw := httptest.NewRequest(http.MethodPost, "/api/switch-workspace", strings.NewReader(`{"workspace_id":"`+wsID+`"}`))
	reqSw.Header.Set("X-Auth-User-Id", "u1")
	recSw := httptest.NewRecorder()
	mux.ServeHTTP(recSw, reqSw)

	if recSw.Code != http.StatusOK {
		t.Fatalf("legacy switch: expected 200, got %d: %s", recSw.Code, recSw.Body.String())
	}
	var swResult map[string]interface{}
	json.NewDecoder(recSw.Body).Decode(&swResult)
	if swResult["success"] != true {
		t.Errorf("expected success=true, got %#v", swResult)
	}
}
