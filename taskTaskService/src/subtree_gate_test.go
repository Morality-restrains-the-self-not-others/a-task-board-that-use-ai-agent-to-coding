package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func stubProgressColumns(t *testing.T, m map[string]string) {
	t.Helper()
	prev := progressColumnNameByIDFn
	progressColumnNameByIDFn = func(tenantID, workspaceID string, columnIDs []string) map[string]string {
		out := map[string]string{}
		for _, id := range columnIDs {
			if name, ok := m[id]; ok {
				out[id] = name
			}
		}
		return out
	}
	t.Cleanup(func() { progressColumnNameByIDFn = prev })
}

func insertTreeTask(t *testing.T, id, title, parent, col string, completed bool) {
	t.Helper()
	now := time.Now().UTC()
	c := 0
	if completed {
		c = 1
	}
	_, err := db.Exec(`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		VALUES(?,?,?,?,?,'medium',0,'ws1','u1','',?,?,?,'',0,'company','','','',?,?)`,
		id, "t1", title, "", c, col, parent, "", now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubtreeShowsChildAndGrandchild(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubProgressColumns(t, map[string]string{
		"col-wip":  "进行中",
		"col-done": "已完成",
	})

	insertTreeTask(t, "root", "Root", "", "col-wip", false)
	insertTreeTask(t, "child", "Child", "root", "col-wip", false)
	insertTreeTask(t, "gc", "Grand", "child", "col-done", true)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/root/subtree/?max_depth=2", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&payload)
	summary, _ := payload["summary"].(map[string]interface{})
	if int(summary["total"].(float64)) != 2 {
		t.Fatalf("total=%v", summary["total"])
	}
	if int(summary["settled"].(float64)) != 1 {
		t.Fatalf("settled=%v", summary["settled"])
	}
	nodes, _ := payload["nodes"].([]interface{})
	if len(nodes) != 2 {
		t.Fatalf("nodes=%d", len(nodes))
	}
}

func TestTerminalGateBlocksOpenChild(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubProgressColumns(t, map[string]string{
		"col-wip":  "进行中",
		"col-done": "已完成",
	})

	insertTreeTask(t, "root2", "Root", "", "col-wip", false)
	insertTreeTask(t, "child2", "Child", "root2", "col-wip", false)

	patch := `{"progress_column_id":"col-done"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/root2/", strings.NewReader(patch))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req.Header.Set("X-Task-Test-Progress-Column-Name", "已完成")
	req.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-done")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	if body["code"] != errCodeDescendantsNotTerminal {
		t.Fatalf("code=%v", body["code"])
	}
}

func TestTerminalGateAllowsWhenDescendantsSettled(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubProgressColumns(t, map[string]string{
		"col-wip":    "进行中",
		"col-done":   "已完成",
		"col-cancel": "已取消",
	})

	insertTreeTask(t, "root3", "Root", "", "col-wip", false)
	insertTreeTask(t, "child3", "Child", "root3", "col-done", true)
	insertTreeTask(t, "gc3", "Grand", "child3", "col-cancel", false)

	patch := `{"progress_column_id":"col-done"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/root3/", strings.NewReader(patch))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req.Header.Set("X-Task-Test-Progress-Column-Name", "已完成")
	req.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-done")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestTerminalGateNoChildrenPasses(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubProgressColumns(t, map[string]string{"col-done": "已完成"})

	insertTreeTask(t, "lonely", "Lonely", "", "col-wip", false)

	patch := `{"progress_column_id":"col-done"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/lonely/", strings.NewReader(patch))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req.Header.Set("X-Task-Test-Progress-Column-Name", "已完成")
	req.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-done")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}
