package aliyun

import "testing"

func TestECSInstanceNameCommentScoped(t *testing.T) {
	taskID := "task_876810593758113792"
	java := ECSInstanceName(taskID, "cmt_java")
	lisp := ECSInstanceName(taskID, "cmt_lisp")
	if java != taskID+"_cmt_java" {
		t.Fatalf("java=%q", java)
	}
	if lisp != taskID+"_cmt_lisp" {
		t.Fatalf("lisp=%q", lisp)
	}
	if java == lisp {
		t.Fatal("two comments must not share InstanceName")
	}
	if ECSInstanceName(taskID, "") != "" {
		t.Fatalf("empty comment must not fall back to task id, got %q", ECSInstanceName(taskID, ""))
	}
	if ECSInstanceName("13908509172117356865", "cmt_42") != "task_13908509172117356865_cmt_42" {
		t.Fatalf("missing task_ prefix: %q", ECSInstanceName("13908509172117356865", "cmt_42"))
	}
	if names := ECSInstanceNames(taskID, "cmt_java"); len(names) != 1 || names[0] != java {
		t.Fatalf("lookup must be comment-only, got %v", names)
	}
	if names := ECSInstanceNames(taskID, ""); names != nil {
		t.Fatalf("empty comment lookup must be empty, got %v", names)
	}
}
