package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260827-038：负号雪花 workspace ID（旧 genID UnixNano 溢出产物）必须作为
// 不透明字符串在 DB 查询与 JSON 往返中原样保留，禁止任何整型解析/改写。
func TestAssociateWorkspaceNegativeSnowflakeIDStringTransit(t *testing.T) {
	setupTestDB(t)

	const negWSID = "ws_-2309487803472456748"
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id, name, company_id) VALUES(?,?,?)`,
		negWSID, "用户的工作空间", "t1",
	); err != nil {
		t.Fatalf("insert negative workspace: %v", err)
	}

	projBody := `{"name":"Proj NegWS"}`
	projReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(projBody))
	projReq.Header.Set("X-Auth-Tenant-Id", "t1")
	projRec := httptest.NewRecorder()
	handleCreateProject(projRec, projReq)
	if projRec.Code != http.StatusCreated {
		t.Fatalf("create project: %d %s", projRec.Code, projRec.Body.String())
	}
	var proj map[string]interface{}
	json.NewDecoder(projRec.Body).Decode(&proj)
	pid := proj["id"].(string)

	addBody := `{"workspace_id":"` + negWSID + `"}`
	addReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(addBody))
	addReq.Header.Set("X-Auth-Tenant-Id", "t1")
	addRec := httptest.NewRecorder()
	handleAssociateWorkspace(addRec, addReq, "t1", pid)
	if addRec.Code != http.StatusOK {
		t.Fatalf("associate negative ws: %d %s", addRec.Code, addRec.Body.String())
	}
	var afterAdd map[string]interface{}
	json.NewDecoder(addRec.Body).Decode(&afterAdd)
	wsList, _ := afterAdd["workspaces"].([]interface{})
	if len(wsList) != 1 || wsList[0] != negWSID {
		t.Fatalf("after add workspaces=%v want [%s] (negative snowflake must survive as opaque string)", wsList, negWSID)
	}

	// 移除也应按字符串精确匹配。
	rmBody := `{"workspace_id":"` + negWSID + `","action":"remove"}`
	rmReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(rmBody))
	rmReq.Header.Set("X-Auth-Tenant-Id", "t1")
	rmRec := httptest.NewRecorder()
	handleAssociateWorkspace(rmRec, rmReq, "t1", pid)
	if rmRec.Code != http.StatusOK {
		t.Fatalf("remove negative ws: %d %s", rmRec.Code, rmRec.Body.String())
	}
	var afterRm map[string]interface{}
	json.NewDecoder(rmRec.Body).Decode(&afterRm)
	wsAfterRm, _ := afterRm["workspaces"].([]interface{})
	if len(wsAfterRm) != 0 {
		t.Fatalf("after remove workspaces=%v want []", wsAfterRm)
	}
}

func TestAssociateWorkspaceAddAndRemove(t *testing.T) {
	setupTestDB(t)

	wsBody := `{"name":"WS Assoc"}`
	wsReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/", strings.NewReader(wsBody))
	wsReq.Header.Set("X-Auth-Tenant-Id", "t1")
	wsRec := httptest.NewRecorder()
	handleCreateWorkspace(wsRec, wsReq)
	if wsRec.Code != http.StatusOK {
		t.Fatalf("create workspace: %d %s", wsRec.Code, wsRec.Body.String())
	}
	var ws map[string]interface{}
	json.NewDecoder(wsRec.Body).Decode(&ws)
	wsID := ws["id"].(string)

	projBody := `{"name":"Proj Assoc"}`
	projReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(projBody))
	projReq.Header.Set("X-Auth-Tenant-Id", "t1")
	projRec := httptest.NewRecorder()
	handleCreateProject(projRec, projReq)
	if projRec.Code != http.StatusCreated {
		t.Fatalf("create project: %d %s", projRec.Code, projRec.Body.String())
	}
	var proj map[string]interface{}
	json.NewDecoder(projRec.Body).Decode(&proj)
	pid := proj["id"].(string)

	addBody := `{"workspace_id":"` + wsID + `"}`
	addReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(addBody))
	addReq.Header.Set("X-Auth-Tenant-Id", "t1")
	addRec := httptest.NewRecorder()
	handleAssociateWorkspace(addRec, addReq, "t1", pid)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add: %d %s", addRec.Code, addRec.Body.String())
	}
	var afterAdd map[string]interface{}
	json.NewDecoder(addRec.Body).Decode(&afterAdd)
	wsList, _ := afterAdd["workspaces"].([]interface{})
	if len(wsList) != 1 || wsList[0] != wsID {
		t.Fatalf("after add workspaces=%v want [%s]", wsList, wsID)
	}

	// idempotent add
	addRec2 := httptest.NewRecorder()
	addReq2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(addBody))
	handleAssociateWorkspace(addRec2, addReq2, "t1", pid)
	if addRec2.Code != http.StatusOK {
		t.Fatalf("idempotent add: %d %s", addRec2.Code, addRec2.Body.String())
	}

	rmBody := `{"workspace_id":"` + wsID + `","action":"remove"}`
	rmReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(rmBody))
	rmRec := httptest.NewRecorder()
	handleAssociateWorkspace(rmRec, rmReq, "t1", pid)
	if rmRec.Code != http.StatusOK {
		t.Fatalf("remove: %d %s", rmRec.Code, rmRec.Body.String())
	}
	var afterRm map[string]interface{}
	json.NewDecoder(rmRec.Body).Decode(&afterRm)
	wsList2, _ := afterRm["workspaces"].([]interface{})
	if len(wsList2) != 0 {
		t.Fatalf("after remove workspaces=%v want []", wsList2)
	}
}

func TestAssociateWorkspaceViaProjectsRoute(t *testing.T) {
	setupTestDB(t)

	wsBody := `{"name":"WS Route"}`
	wsReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/", strings.NewReader(wsBody))
	wsReq.Header.Set("X-Auth-Tenant-Id", "t1")
	wsRec := httptest.NewRecorder()
	handleCreateWorkspace(wsRec, wsReq)
	var ws map[string]interface{}
	json.NewDecoder(wsRec.Body).Decode(&ws)
	wsID := ws["id"].(string)

	projBody := `{"name":"Proj Route"}`
	projReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(projBody))
	projReq.Header.Set("X-Auth-Tenant-Id", "t1")
	projRec := httptest.NewRecorder()
	handleCreateProject(projRec, projReq)
	var proj map[string]interface{}
	json.NewDecoder(projRec.Body).Decode(&proj)
	pid := proj["id"].(string)

	body := `{"workspace_id":"` + wsID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/"+pid+"/associateWorkspace/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{pid, "associateWorkspace"})
	if rec.Code != http.StatusOK {
		t.Fatalf("route: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "endpoint not yet implemented") {
		t.Fatalf("still unimplemented: %s", rec.Body.String())
	}
}

func TestWriteNotImplementedIncludesTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Trace-Id", "tid-501-assoc-1")
	rec := httptest.NewRecorder()
	writeNotImplemented(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["trace_id"] != "tid-501-assoc-1" {
		t.Fatalf("body=%v", body)
	}
}
