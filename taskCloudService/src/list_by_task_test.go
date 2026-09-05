package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalListByTaskOmitsTemplateRow(t *testing.T) {
	setupCloudTestDB(t)
	seedTaskAndCommentCSC(t, "task-list", "cmt-list", "i-tpl-ignored", "i-cmt-live")

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/list-by-task/?tenant_id=t1&workspace_id=ws1&task_id=task-list", nil)
	rec := httptest.NewRecorder()
	handleInternalListByTask(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	raw, _ := out["comments"].([]interface{})
	if len(raw) != 1 {
		t.Fatalf("comments=%d want 1 (template omitted) body=%v", len(raw), out)
	}
	cmt, _ := raw[0].(map[string]interface{})
	if cmt["comment_id"] != "cmt-list" {
		t.Fatalf("comment_id=%v", cmt["comment_id"])
	}
	if cmt["instance_id"] != "i-cmt-live" {
		t.Fatalf("instance_id=%v", cmt["instance_id"])
	}
	if out["task_id"] != "task-list" {
		t.Fatalf("task_id=%v", out["task_id"])
	}
}

func TestInternalListByTaskEmptyWhenOnlyTemplate(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "task-only-tpl", "")
	_, _ = db.Exec(`DELETE FROM cloud_server_configs WHERE task_id=? AND TRIM(COALESCE(comment_id,''))!=''`, "task-only-tpl")

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/list-by-task/?tenant_id=t1&workspace_id=ws1&task_id=task-only-tpl", nil)
	rec := httptest.NewRecorder()
	handleInternalListByTask(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	raw, _ := out["comments"].([]interface{})
	if len(raw) != 0 {
		t.Fatalf("comments=%d want 0", len(raw))
	}
}

func TestInternalListByTaskRequiresTaskID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-config/list-by-task/?tenant_id=t1", nil)
	rec := httptest.NewRecorder()
	handleInternalListByTask(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rec.Code)
	}
}
