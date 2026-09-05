package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskCloudService/domain"
)

func TestHandleJobStepFullPushPersists(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", PathRule: domain.DefaultStepFullPathRule}
	rec := httptest.NewRecorder()
	handleJobStepFullPush(rec, context.Background(), &CloudServerConfig{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a",
	}, map[string]any{
		"job_id":     "J9",
		"job_status": "completed",
		"layer_id":   "L1",
		"steps": []any{
			map[string]any{"step_number": float64(1), "delivery_summary": "think", "state": "completed"},
		},
	}, "t1", "ws1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetJobExecutionLogHydratesFromStepFull(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", PathRule: domain.DefaultStepFullPathRule}
	if _, err := persistStepFullArchive(context.Background(), domain.StepFullIDs{
		WorkspaceID: "w1", TaskID: "task1", CommentID: "cmt-a", JobID: "J1", LayerID: "L1",
	}, "t1", "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "full think", "llm": "kept"},
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/container-job-execution-log/tenant_id/t1/workspace_id/w1/task_id/task1/comment_id/cmt-a/?job_id=J1",
		nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "w1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-job-execution-log/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["source"] != "saas_cos" {
		t.Fatalf("source=%v body=%s", body["source"], rec.Body.String())
	}
	steps := body["steps"].(map[string]any)["steps"].([]any)
	if len(steps) != 1 {
		t.Fatalf("steps=%v", steps)
	}
	first := steps[0].(map[string]any)
	if first["delivery_summary"] != "full think" || first["llm"] != "kept" {
		t.Fatalf("step=%v", first)
	}
}

func TestHandleJobStepFullPushRequiresJobID(t *testing.T) {
	rec := httptest.NewRecorder()
	handleJobStepFullPush(rec, context.Background(), &CloudServerConfig{CommentID: "cmt-a"}, map[string]any{
		"steps": []any{},
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}
