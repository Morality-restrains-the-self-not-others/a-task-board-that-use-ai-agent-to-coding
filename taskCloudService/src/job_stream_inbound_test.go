package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildJobStreamSSEStatusDataStep(t *testing.T) {
	got := buildJobStreamSSEStatusData(map[string]any{
		"job_id":  "J1",
		"phase":   "step",
		"message": "step 2: bash ls",
		"seq":     float64(4),
		"event": map[string]any{
			"step_number":      float64(2),
			"delivery_summary": "bash ls",
			"state":            "completed",
		},
		"job_status": "running",
		"layer_id":   "L1",
	}, "J1")
	if got["status"] != "container_job_stream" {
		t.Fatalf("status=%v", got["status"])
	}
	if got["job_id"] != "J1" || got["phase"] != "step" {
		t.Fatalf("got=%v", got)
	}
	if got["seq"] != 4 {
		t.Fatalf("seq=%v", got["seq"])
	}
	if got["step_number"] != 2 {
		t.Fatalf("step_number=%v", got["step_number"])
	}
	if got["delivery_summary"] != "bash ls" {
		t.Fatalf("delivery_summary=%v", got["delivery_summary"])
	}
	if got["job_status"] != "running" || got["layer_id"] != "L1" {
		t.Fatalf("got=%v", got)
	}
}

func TestHandleJobStreamPushRequiresJobID(t *testing.T) {
	rec := httptest.NewRecorder()
	handleJobStreamPush(rec, context.Background(), &CloudServerConfig{CommentID: "cmt-a"}, map[string]any{
		"phase": "step",
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleJobStreamPushPublishesSSEMessage(t *testing.T) {
	var gotTask, gotComment string
	var gotData map[string]interface{}
	prev := publishJobStreamSSE
	publishJobStreamSSE = func(ctx context.Context, taskID, commentID string, statusData map[string]interface{}) error {
		gotTask, gotComment, gotData = taskID, commentID, statusData
		return nil
	}
	t.Cleanup(func() { publishJobStreamSSE = prev })

	rec := httptest.NewRecorder()
	handleJobStreamPush(rec, context.Background(), &CloudServerConfig{
		TaskID: "task1", CommentID: "cmt-a",
	}, map[string]any{
		"job_id":  "J9",
		"phase":   "step",
		"message": "step 1: think",
		"seq":     float64(1),
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotTask != "task1" || gotComment != "cmt-a" {
		t.Fatalf("task=%q comment=%q", gotTask, gotComment)
	}
	if gotData["status"] != "container_job_stream" || gotData["job_id"] != "J9" {
		t.Fatalf("status_data=%v", gotData)
	}
	if gotData["company_id"] != "t1" || gotData["workspace_id"] != "w1" || gotData["comment_id"] != "cmt-a" {
		t.Fatalf("scope status_data=%v", gotData)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleJobStreamPushKafkaFailure(t *testing.T) {
	prev := publishJobStreamSSE
	publishJobStreamSSE = func(ctx context.Context, taskID, commentID string, statusData map[string]interface{}) error {
		return errors.New("broker down")
	}
	t.Cleanup(func() { publishJobStreamSSE = prev })

	rec := httptest.NewRecorder()
	handleJobStreamPush(rec, context.Background(), &CloudServerConfig{CommentID: "cmt-a"}, map[string]any{
		"job_id": "J1", "phase": "step",
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
