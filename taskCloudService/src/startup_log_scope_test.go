package main

import "testing"

func TestWithStartupImageLogScopeIncludesInstanceAndContainer(t *testing.T) {
	out := withStartupImageLogScope(map[string]interface{}{
		"task_id":     "t1",
		"comment_id":  "c1",
		"instance_id": "i-xyz",
	}, map[string]interface{}{
		"status":  "processing",
		"message": "前置资源创建任务已提交，正在等待处理...",
	})
	want := "[i-xyz、task_t1_c1] 前置资源创建任务已提交，正在等待处理..."
	if out["message"] != want {
		t.Fatalf("message=%q want %q", out["message"], want)
	}
	if out["container_name"] != "task_t1_c1" {
		t.Fatalf("container_name=%v", out["container_name"])
	}
}

func TestWithStartupImageLogScopeDashForMissingInstance(t *testing.T) {
	out := withStartupImageLogScope(map[string]interface{}{
		"container_name": "task2app-container",
	}, map[string]interface{}{
		"message": "正在处理自动创建资源任务...",
	})
	want := "[-、task2app-container] 正在处理自动创建资源任务..."
	if out["message"] != want {
		t.Fatalf("message=%q want %q", out["message"], want)
	}
}
