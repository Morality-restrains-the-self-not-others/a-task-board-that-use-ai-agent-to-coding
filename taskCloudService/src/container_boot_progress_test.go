package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleBootProgressPublishesProcessingSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	handleBootProgress(
		rec,
		nil,
		&CloudServerConfig{CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		map[string]any{
			"progress": float64(35),
			"message":  "安装容器运行时...",
			"status":   "processing",
		},
		"t1", "w1", "task1",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["ok"] != true {
		t.Fatalf("body=%v", out)
	}
	if out["progress"] != float64(35) {
		t.Fatalf("progress=%v", out["progress"])
	}
}

func TestHandleBootProgressCapsAt99AndRewritesSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	handleBootProgress(
		rec,
		nil,
		&CloudServerConfig{TaskID: "task1"},
		map[string]any{
			"progress": float64(100),
			"message":  "done",
			"status":   "success",
		},
		"t1", "w1", "task1",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"progress":99`) {
		t.Fatalf("expected progress capped to 99, body=%s", rec.Body.String())
	}
}

func TestBuildBootProgressStatusDataIncludesCommentScope(t *testing.T) {
	out := buildBootProgressStatusData(
		&CloudServerConfig{
			TaskID:     "task1",
			CommentID:  "cmt_9",
			InstanceID: "i-abc",
		},
		map[string]any{
			"progress":       float64(35),
			"message":        "安装容器运行时...",
			"status":         "processing",
			"container_name": "task_task1_cmt_9",
		},
		"task1",
	)
	if out["phase"] != "userdata_boot" {
		t.Fatalf("phase=%v", out["phase"])
	}
	if out["comment_id"] != "cmt_9" {
		t.Fatalf("comment_id=%v", out["comment_id"])
	}
	msg, _ := out["message"].(string)
	if !strings.Contains(msg, "安装容器运行时") {
		t.Fatalf("message=%q", msg)
	}
	if !strings.Contains(msg, "i-abc") && !strings.Contains(msg, "task_task1_cmt_9") {
		t.Fatalf("expected log_label scope in message, got %q", msg)
	}
	if out["status"] != "processing" {
		t.Fatalf("status=%v", out["status"])
	}
}

func TestBuildBootProgressStatusDataRewritesSuccess(t *testing.T) {
	out := buildBootProgressStatusData(
		&CloudServerConfig{TaskID: "task1", CommentID: "c1"},
		map[string]any{"progress": float64(100), "message": "done", "status": "success"},
		"task1",
	)
	if out["status"] != "processing" {
		t.Fatalf("status=%v want processing", out["status"])
	}
	if out["progress"] != 99 {
		t.Fatalf("progress=%v want 99", out["progress"])
	}
}

func TestResolveBootProgressCSCKeepsCommentRow(t *testing.T) {
	in := &CloudServerConfig{CommentID: "c1", InstanceID: "i-1"}
	got := resolveBootProgressCSC(in, "t1", "w1", "task1")
	if got != in || got.CommentID != "c1" {
		t.Fatalf("got=%+v", got)
	}
}

// 回归：token 落到任务级 CSC 且已挂 instance（attach/reuse 常见）时，
// 仍应切到带 comment_id 的评论级 CSC，否则 boot-progress SSE 无 comment_id，
// 评论「启动日志」只剩容器调度三行、看不到 UserData 步骤。
func TestResolveBootProgressCSCSwitchesTaskLevelWithInstanceToCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg-task-shared", "t1", "ws1", "task-bp-switch", "", "aliyun", "i-shared-1", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed task csc: %v", err)
	}
	_, err = db.Exec(
		`INSERT INTO cloud_server_configs(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"cfg-cmt-shared", "t1", "ws1", "task-bp-switch", "cmt_boot", "aliyun", "i-shared-1", "cn-test", "cn-test-a", "test",
	)
	if err != nil {
		t.Fatalf("seed comment csc: %v", err)
	}

	taskLevel := &CloudServerConfig{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-bp-switch",
		CommentID: "", InstanceID: "i-shared-1",
	}
	got := resolveBootProgressCSC(taskLevel, "t1", "ws1", "task-bp-switch")
	if got == nil || got.CommentID != "cmt_boot" {
		t.Fatalf("want comment CSC cmt_boot, got=%+v", got)
	}
}
