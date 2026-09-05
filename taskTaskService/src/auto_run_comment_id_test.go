package main

import (
	"fmt"
	"strings"
	"testing"
)

func startVmBodyFromCloudCalls(t *testing.T, cloudCalls *[]map[string]interface{}) map[string]interface{} {
	t.Helper()
	if cloudCalls == nil || len(*cloudCalls) == 0 {
		t.Fatal("expected start-vm cloud call")
	}
	body, _ := (*cloudCalls)[0]["body"].(map[string]interface{})
	if body == nil {
		t.Fatalf("start-vm body missing: %#v", (*cloudCalls)[0])
	}
	return body
}

func TestTriggerTaskAutoRunAttachesCommentID(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_cmt_id_t1",
		UserID:      "u1",
		ImageID:     "img1",
		RunTemplate: configuredRunTemplate(),
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	body := startVmBodyFromCloudCalls(t, cloudCalls)
	if body["comment_id"] != "cmt-mock-auto-run" {
		t.Fatalf("comment_id=%v", body["comment_id"])
	}
	if body["parent_comment_id"] != "cmt-mock-auto-run" {
		t.Fatalf("parent_comment_id=%v", body["parent_comment_id"])
	}
	// OPT-20260815-012: 自动运行补齐 container_name（与 cloudcommon.DeriveContainerName 同规则）
	if body["container_name"] != "task_cmt_id_t1_cmt-mock-auto-run" {
		t.Fatalf("container_name=%v", body["container_name"])
	}
}

func TestTriggerTaskAutoRunSkipsStartVMWhenEnsureCommentFails(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		return "", fmt.Errorf("ensure boom")
	}

	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_cmt_id_t2",
		UserID:      "u1",
		ImageID:     "img1",
		RunTemplate: configuredRunTemplate(),
	})
	if err == nil || !strings.Contains(err.Error(), "ensure boom") {
		t.Fatalf("err=%v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("start-vm must not run; calls=%d", len(*cloudCalls))
	}
}

func TestTriggerTaskAutoRunSkipsStartVMWhenCommentIDEmpty(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		return "  ", nil
	}

	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_cmt_id_t3",
		UserID:      "u1",
		ImageID:     "img1",
		RunTemplate: configuredRunTemplate(),
	})
	if err == nil || !strings.Contains(err.Error(), "comment_id") {
		t.Fatalf("err=%v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("start-vm must not run; calls=%d", len(*cloudCalls))
	}
}

func TestAttachStartVmCommentIDWritesBothFields(t *testing.T) {
	body := attachStartVmCommentID(nil, "cmt_x", "task_9")
	if body["comment_id"] != "cmt_x" || body["parent_comment_id"] != "cmt_x" {
		t.Fatalf("body=%v", body)
	}
	if body["container_name"] != "task_9_cmt_x" {
		t.Fatalf("container_name=%v", body["container_name"])
	}
}

func TestAttachStartVmCommentIDContainerNameRules(t *testing.T) {
	// taskID 已带 task_ 前缀 → 不重复加前缀（与 cloudcommon.DeriveContainerName 一致）
	body := attachStartVmCommentID(map[string]interface{}{}, "cmt_a", "task_1566")
	if body["container_name"] != "task_1566_cmt_a" {
		t.Fatalf("带前缀 container_name=%v", body["container_name"])
	}
	// 裸 taskID → 补 task_ 前缀
	body = attachStartVmCommentID(map[string]interface{}{}, "cmt_b", "1567")
	if body["container_name"] != "task_1567_cmt_b" {
		t.Fatalf("裸 taskID container_name=%v", body["container_name"])
	}
	// body 已带 container_name → 尊重显式值，不覆盖
	body = attachStartVmCommentID(map[string]interface{}{"container_name": "custom"}, "cmt_c", "task_8")
	if body["container_name"] != "custom" {
		t.Fatalf("显式 container_name=%v", body["container_name"])
	}
	// comment_id 为空 → 不写任何字段（兼容旧行为）
	body = attachStartVmCommentID(nil, "  ", "task_7")
	if len(body) != 0 {
		t.Fatalf("空 comment_id 不应写 body: %v", body)
	}
}
