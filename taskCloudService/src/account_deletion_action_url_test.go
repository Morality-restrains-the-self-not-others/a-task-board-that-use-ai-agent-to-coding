package main

import "testing"

func TestCloudResourceActionURLUsesTaskDetailPage(t *testing.T) {
	got := cloudResourceActionURL("t1", "w1", "task_1")
	want := "/tenant/t1/workspace/w1/task-detail/task_1/"
	if got != want {
		t.Fatalf("cloudResourceActionURL = %q, want %q (/task/ 不是前端页，会被 catch-all 踢回首页)", got, want)
	}
}

func TestCloudResourceActionURLFallbackWhenMissingIDs(t *testing.T) {
	if got := cloudResourceActionURL("t1", "w1", ""); got != "/profile/" {
		t.Fatalf("empty task: got %q", got)
	}
}
