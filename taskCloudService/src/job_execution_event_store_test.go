package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShouldPersistJobStreamPhase(t *testing.T) {
	for _, p := range []string{"step", "start", "running", "completed", "failed", "interrupted", "STEP"} {
		if !shouldPersistJobStreamPhase(p) {
			t.Fatalf("want persist %q", p)
		}
	}
	for _, p := range []string{"chunk", "error", "", "pending"} {
		if shouldPersistJobStreamPhase(p) {
			t.Fatalf("want skip %q", p)
		}
	}
}

func TestReconstructJobExecutionLogPagesSteps(t *testing.T) {
	events := []jobExecutionEventRow{
		{Phase: "start", JobStatus: "running", LayerID: "L1", Seq: 0},
		{Phase: "step", StepNumber: 1, DeliverySummary: "think", StepState: "completed", Seq: 1},
		{Phase: "chunk", Message: "stdout", Seq: 2},
		{Phase: "step", StepNumber: 2, DeliverySummary: "bash ls", StepState: "completed", Seq: 3},
		{Phase: "completed", JobStatus: "completed", Seq: 4},
	}
	payload := reconstructJobExecutionLog(events, "J1", 0, 20)
	job, _ := payload["job"].(map[string]any)
	if job["id"] != "J1" || job["status"] != "completed" || job["layer_id"] != "L1" {
		t.Fatalf("job=%v", job)
	}
	stepsWrap, _ := payload["steps"].(map[string]any)
	steps, _ := stepsWrap["steps"].([]map[string]any)
	if len(steps) != 2 || steps[0]["step_number"] != 1 || steps[1]["step_number"] != 2 {
		t.Fatalf("steps=%v", steps)
	}
	page := reconstructJobExecutionLog(events, "J1", 1, 20)
	pageSteps := page["steps"].(map[string]any)["steps"].([]map[string]any)
	if len(pageSteps) != 1 || pageSteps[0]["step_number"] != 2 {
		t.Fatalf("page=%v", pageSteps)
	}
}

func TestInsertAndGetJobExecutionLogFromDB(t *testing.T) {
	setupCloudTestDB(t)
	created, err := insertJobExecutionEvent(jobExecutionEventRow{
		CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1", CommentID: "cmt-a",
		JobID: "J1", Seq: 1, Phase: "step", Message: "step 1: think",
		StepNumber: 1, DeliverySummary: "think", StepState: "completed", JobStatus: "running", LayerID: "L1",
	})
	if err != nil || !created {
		t.Fatalf("insert step: created=%v err=%v", created, err)
	}
	dup, err := insertJobExecutionEvent(jobExecutionEventRow{
		CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1", CommentID: "cmt-a",
		JobID: "J1", Seq: 1, Phase: "step",
	})
	if err != nil || dup {
		t.Fatalf("dup should no-op created=%v err=%v", dup, err)
	}
	skipped, err := insertJobExecutionEvent(jobExecutionEventRow{
		TaskID: "task1", JobID: "J1", Seq: 2, Phase: "chunk", Message: "x",
	})
	if err != nil || skipped {
		t.Fatalf("chunk should skip created=%v err=%v", skipped, err)
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
	if body["source"] != "saas_db" {
		t.Fatalf("source=%v body=%s", body["source"], rec.Body.String())
	}
	steps := body["steps"].(map[string]any)["steps"].([]any)
	if len(steps) != 1 {
		t.Fatalf("steps=%v", steps)
	}
}

func TestInternalPersistJobExecutionEvent(t *testing.T) {
	setupCloudTestDB(t)
	raw := `{"company_id":"t1","workspace_id":"w1","task_id":"task1","comment_id":"cmt-a","job_id":"J2","seq":3,"phase":"completed","job_status":"completed"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/job-execution-events/", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalJobExecutionEvents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListJobExecutionEventsScopedByWorkspace(t *testing.T) {
	setupCloudTestDB(t)
	// 同一 task_id+job_id 出现在两个 workspace，查询必须按 workspace 隔离。
	insertJobExecutionEvent(jobExecutionEventRow{
		CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1", CommentID: "cmt-a",
		JobID: "J1", Seq: 1, Phase: "step", Message: "w1 only", StepNumber: 1,
		StepState: "completed", JobStatus: "running", LayerID: "L1",
	})
	insertJobExecutionEvent(jobExecutionEventRow{
		CompanyID: "t1", WorkspaceID: "w2", TaskID: "task1", CommentID: "cmt-b",
		JobID: "J1", Seq: 2, Phase: "step", Message: "w2 only", StepNumber: 1,
		StepState: "completed", JobStatus: "running", LayerID: "L1",
	})

	eventsW1, err := listJobExecutionEvents("w1", "task1", "cmt-a", "J1")
	if err != nil {
		t.Fatalf("w1 err=%v", err)
	}
	if len(eventsW1) != 1 || eventsW1[0].WorkspaceID != "w1" || eventsW1[0].Message != "w1 only" {
		t.Fatalf("w1 events=%+v", eventsW1)
	}
	eventsW2, err := listJobExecutionEvents("w2", "task1", "cmt-b", "J1")
	if err != nil {
		t.Fatalf("w2 err=%v", err)
	}
	if len(eventsW2) != 1 || eventsW2[0].WorkspaceID != "w2" || eventsW2[0].Message != "w2 only" {
		t.Fatalf("w2 events=%+v", eventsW2)
	}
	// 空 workspace 必须报错
	if _, err := listJobExecutionEvents("", "task1", "cmt-a", "J1"); err == nil {
		t.Fatal("empty workspace should error")
	}
}

func TestJobExecutionEventShardCrossWorkspaceRouting(t *testing.T) {
	setupCloudTestDB(t)
	// w1→04 片、w2→14 片（CRC32 % 16 不同）：事件必须落到各自分片，跨片读写互不可见。
	if jobExecutionEventShardIndex("w1") == jobExecutionEventShardIndex("w2") {
		t.Fatalf("w1/w2 unexpectedly share shard %d", jobExecutionEventShardIndex("w1"))
	}
	for _, w := range []string{"w1", "w2"} {
		created, err := insertJobExecutionEvent(jobExecutionEventRow{
			CompanyID: "t1", WorkspaceID: w, TaskID: "task-shard", CommentID: "cmt-" + w,
			JobID: "J1", Seq: 1, Phase: "step", Message: "msg-" + w, StepNumber: 1,
			StepState: "completed", JobStatus: "running", LayerID: "L1",
		})
		if err != nil || !created {
			t.Fatalf("insert %s: created=%v err=%v", w, created, err)
		}
	}

	tableW1, _ := jobExecutionEventTable("w1")
	tableW2, _ := jobExecutionEventTable("w2")
	if tableW1 == tableW2 {
		t.Fatalf("shard tables unexpectedly equal %s", tableW1)
	}
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM `+tableW1+` WHERE task_id = 'task-shard' AND workspace_id = 'w1'`).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("shard %s w1 rows=%d want 1", tableW1, cnt)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM `+tableW1+` WHERE task_id = 'task-shard' AND workspace_id = 'w2'`).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("shard %s w2 rows=%d want 0 (cross-shard isolation)", tableW1, cnt)
	}

	// latestJobIDForComment 按 workspace 选片：w1 查自己命中，w2 片查 w1 评论为空（跨片隔离）。
	jobW1, err := latestJobIDForComment("w1", "task-shard", "cmt-w1")
	if err != nil || jobW1 != "J1" {
		t.Fatalf("latest w1=%q err=%v", jobW1, err)
	}
	jobLeak, err := latestJobIDForComment("w2", "task-shard", "cmt-w1")
	if err != nil {
		t.Fatalf("latest cross err=%v", err)
	}
	if jobLeak != "" {
		t.Fatalf("latest via w2 shard = %q, want empty", jobLeak)
	}
}

func TestHandleContainerJobExecutionLogRequiresWorkspace(t *testing.T) {
	setupCloudTestDB(t)
	// 层图对齐：无 workspace_id 直接 400，不得按 task_id+job_id 扫全库。
	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/container-job-execution-log/tenant_id/t1/task_id/task1/?job_id=J1",
		nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-job-execution-log/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "workspace_id and task_id are required") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
