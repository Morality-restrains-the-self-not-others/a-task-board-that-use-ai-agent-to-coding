package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleInternalTasksTerminalKinds(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	insert := `INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		VALUES(?,?,?,?,?,'medium',0,'ws1','u1','',?,?, '','',0,'company','','','',?,?)`
	for _, row := range []struct {
		id, col string
		done    int
	}{
		{"task-cancel-1", "col-cancel", 0},
		{"task-done-1", "col-done", 0},
		{"task-open-1", "col-wip", 0},
		{"task-flag-1", "col-wip", 1},
	} {
		if _, err := db.Exec(insert, row.id, "t1", "t", "", row.done, row.col, "", now, now); err != nil {
			t.Fatal(err)
		}
	}

	prevFn := progressColumnNameByIDFn
	progressColumnNameByIDFn = func(tenantID, workspaceID string, columnIDs []string) map[string]string {
		return map[string]string{
			"col-cancel": "已取消",
			"col-done":   "已完成",
			"col-wip":    "进行中",
		}
	}
	t.Cleanup(func() { progressColumnNameByIDFn = prevFn })

	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	body := `{"task_ids":["task-cancel-1","task-done-1","task-open-1","task-flag-1","task-missing"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/terminal-kinds/", strings.NewReader(body))
	req.Header.Set("X-Internal-Secret", "sec")
	rec := httptest.NewRecorder()
	handleInternalTasksTerminalKinds(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Kinds map[string]string `json:"kinds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Kinds["task-cancel-1"] != "cancelled" {
		t.Fatalf("cancel kind=%q", out.Kinds["task-cancel-1"])
	}
	if out.Kinds["task-done-1"] != "completed" {
		t.Fatalf("done kind=%q", out.Kinds["task-done-1"])
	}
	if out.Kinds["task-open-1"] != "" {
		t.Fatalf("open kind=%q want empty", out.Kinds["task-open-1"])
	}
	if out.Kinds["task-flag-1"] != "completed" {
		t.Fatalf("flag kind=%q", out.Kinds["task-flag-1"])
	}
	if _, ok := out.Kinds["task-missing"]; ok {
		t.Fatalf("missing task should be omitted, got %v", out.Kinds)
	}
}

func TestHandleInternalTasksTerminalKindsRejectsMissingSecret(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/terminal-kinds/", strings.NewReader(`{"task_ids":["x"]}`))
	rec := httptest.NewRecorder()
	handleInternalTasksTerminalKinds(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}
