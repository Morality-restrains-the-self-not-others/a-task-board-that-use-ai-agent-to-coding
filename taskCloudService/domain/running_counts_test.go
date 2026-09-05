package domain

import "testing"

func TestComputeRunningCountsIgnoresTemplateAndStarting(t *testing.T) {
	got := ComputeRunningCounts([]CommentRuntime{
		{CommentID: "", MachineStarted: true, ContainerReachable: true},
		{CommentID: "cmt-a", MachineStarted: true, ContainerReachable: false},
		{CommentID: "cmt-b", MachineStarted: true, ContainerReachable: true},
		{CommentID: "cmt-c", MachineStarted: false, ContainerReachable: false},
	})
	if got.Machines != 2 || got.Containers != 1 {
		t.Fatalf("got machines=%d containers=%d want 2/1", got.Machines, got.Containers)
	}
}

func TestComputeRunningCountsEmpty(t *testing.T) {
	got := ComputeRunningCounts(nil)
	if got.Machines != 0 || got.Containers != 0 {
		t.Fatalf("empty want 0/0 got %+v", got)
	}
}
