package main

import "testing"

func TestBuildStopVmEventDataRemapsMockLabelOnAliyunInstance(t *testing.T) {
	data := buildStopVmEventData("875", "ws1", "task-1", &CloudServerConfig{
		Platform: "mock", InstanceID: "i-real-ecs", Region: "cn-hongkong",
		AuthorizationID: "cpa_1", WorkspaceID: "ws1",
	}, "")
	if got := data["cloud_platform_type"]; got != "aliyun" {
		t.Fatalf("stale mock on ECS must publish aliyun, got %v", got)
	}
}

func TestBuildStopVmEventDataKeepsMockForMockInstance(t *testing.T) {
	data := buildStopVmEventData("875", "ws1", "task-1", &CloudServerConfig{
		Platform: "mock", InstanceID: "mock-e2e-1", Region: "cn-hongkong",
	}, "user_stop")
	if got := data["cloud_platform_type"]; got != "mock" {
		t.Fatalf("true mock must stay mock, got %v", got)
	}
}

func TestBuildStopVmEventDataIncludesCommentID(t *testing.T) {
	data := buildStopVmEventData("875", "ws1", "task-1", &CloudServerConfig{
		Platform: "mock", InstanceID: "mock-e2e-1", CommentID: "cmt_live",
	}, "instruction_idle")
	if got := data["comment_id"]; got != "cmt_live" {
		t.Fatalf("comment_id=%v want cmt_live", got)
	}
	if got := data["stop_reason"]; got != "instruction_idle" {
		t.Fatalf("stop_reason=%v", got)
	}
	if got := data["stop_reason_label"]; got != "容器指令空闲超时回收" {
		t.Fatalf("stop_reason_label=%v", got)
	}
}

func TestResolveStopEventPlatformPrefersAuth(t *testing.T) {
	if got := resolveStopEventPlatform("mock", "i-abc", "aliyun"); got != "aliyun" {
		t.Fatalf("auth platform must win, got %q", got)
	}
	if got := resolveStopEventPlatform("mock", "mock-e2e-1", ""); got != "mock" {
		t.Fatalf("true mock without auth stays mock, got %q", got)
	}
}
