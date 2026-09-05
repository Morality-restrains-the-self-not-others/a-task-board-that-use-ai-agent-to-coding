package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// OPT-20260816-027：孤儿对账改走 HTTP 批量存在性检查，替代跨库直连。
func TestHandleInternalTasksExist(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		VALUES(?,?,?,?,0,'medium',0,'ws1','u1','','','','','',0,'company','','','',?,?)`,
		"task-exist-1", "t1", "title", "", now, now); err != nil {
		t.Fatal(err)
	}

	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	body := `{"task_ids":["task-exist-1","task-missing-99"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/exists/", strings.NewReader(body))
	req.Header.Set("X-Internal-Secret", "sec")
	rec := httptest.NewRecorder()
	handleInternalTasksExist(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Exists map[string]bool `json:"exists"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Exists["task-exist-1"] {
		t.Fatalf("task-exist-1 should exist, got %v", out.Exists)
	}
	if out.Exists["task-missing-99"] {
		t.Fatalf("task-missing-99 should not exist, got %v", out.Exists)
	}
}

func TestHandleInternalTasksExistRejectsMissingSecret(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/exists/", strings.NewReader(`{"task_ids":["x"]}`))
	rec := httptest.NewRecorder()
	handleInternalTasksExist(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}
