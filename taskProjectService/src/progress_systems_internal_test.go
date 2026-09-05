package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doInternalFirstProgressColumn(tenantID, workspaceID string) *httptest.ResponseRecorder {
	body := `{"tenant_id":"` + tenantID + `","workspace_id":"` + workspaceID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/progress-columns/first", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalResolveFirstProgressColumn(rec, req)
	return rec
}

func TestInternalResolveFirstProgressColumnWorkspaceBinding(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	if _, err := db.Exec("INSERT INTO project_progress_systems(id,name) VALUES(?,?)", "sys-1", "系统A"); err != nil {
		t.Fatalf("insert progress system: %v", err)
	}
	// 乱序插入：order_num=1 的 col-2 应被解析为第一列。
	if _, err := db.Exec(
		"INSERT INTO project_progress_columns(id,system_id,name,order_num) VALUES(?,?,?,?),(?,?,?,?),(?,?,?,?)",
		"col-1", "sys-1", "待办", 2, "col-2", "sys-1", "进行中", 1, "col-3", "sys-1", "完成", 3,
	); err != nil {
		t.Fatalf("insert progress columns: %v", err)
	}
	if _, err := db.Exec(
		"INSERT INTO project_progress_systems_workspace(id,tenant_id,workspace_id,target_type,target_id) VALUES(?,?,?,?,?)",
		"psw-1", "t1", "ws1", "system", "sys-1",
	); err != nil {
		t.Fatalf("insert workspace binding: %v", err)
	}

	rec := doInternalFirstProgressColumn("t1", "ws1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["progress_column_id"] != "col-2" {
		t.Fatalf("expected first column col-2 (order_num=1), got %v", out["progress_column_id"])
	}
	if out["progress_column_name"] != "进行中" {
		t.Fatalf("expected first column name 进行中, got %v", out["progress_column_name"])
	}
}

func TestInternalResolveFirstProgressColumnSystemDefaultFallback(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	// 无 workspace binding 时落到系统默认进度体系（007_seed_defaults 已种 pc_def_sys_1）。
	rec := doInternalFirstProgressColumn("t1", "ws1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["progress_column_id"] == "" {
		t.Fatalf("expected system-default first column, got empty")
	}
}
