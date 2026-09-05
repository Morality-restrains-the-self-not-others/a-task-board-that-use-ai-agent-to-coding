package main

import (
	"strings"
	"testing"
)

func TestBuildRunInstancesRequestCommentScopedInstanceName(t *testing.T) {
	taskID := "task_876810593758113792"
	cmt := "cmt_876810842430009344"
	req := buildRunInstancesRequest(map[string]interface{}{
		"task_id":               taskID,
		"comment_id":            cmt,
		"cloud_server_image_id": "m-img",
		"region_id":             "cn-hangzhou",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"hardware_config":       map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	if req == nil || req.InstanceName == nil {
		t.Fatal("InstanceName required")
	}
	want := ecsInstanceName(taskID, cmt)
	if got := *req.InstanceName; got != want {
		t.Fatalf("InstanceName=%q want %q", got, want)
	}

	other := buildRunInstancesRequest(map[string]interface{}{
		"task_id":               taskID,
		"comment_id":            "cmt_876810929306628096",
		"cloud_server_image_id": "m-img",
		"region_id":             "cn-hangzhou",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"hardware_config":       map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	if other == nil || other.InstanceName == nil {
		t.Fatal("second InstanceName required")
	}
	if *other.InstanceName == *req.InstanceName {
		t.Fatalf("two comments must not share InstanceName %q", *req.InstanceName)
	}

	taskLevel := buildRunInstancesRequest(map[string]interface{}{
		"task_id":               taskID,
		"cloud_server_image_id": "m-img",
		"region_id":             "cn-hangzhou",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"hardware_config":       map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	if taskLevel == nil {
		t.Fatal("task-level request required")
	}
	if taskLevel.InstanceName != nil && strings.TrimSpace(*taskLevel.InstanceName) != "" {
		t.Fatalf("empty comment_id must not set task-level InstanceName, got %q", *taskLevel.InstanceName)
	}

	parent := buildRunInstancesRequest(map[string]interface{}{
		"task_id":               taskID,
		"comment_id":            "cmt_child",
		"parent_comment_id":     cmt,
		"cloud_server_image_id": "m-img",
		"region_id":             "cn-hangzhou",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"hardware_config":       map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	if parent == nil || parent.InstanceName == nil {
		t.Fatal("parent_comment_id InstanceName required")
	}
	if got := *parent.InstanceName; got != want {
		t.Fatalf("parent_comment_id InstanceName=%q want %q", got, want)
	}
}
