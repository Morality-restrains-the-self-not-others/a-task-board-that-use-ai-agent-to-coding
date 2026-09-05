package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestComputeWorkspaceMachineSnapshotCoalescesConcurrentScans(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-sf", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-sf", CommentID: "cmt-sf",
		Platform: "mock", InstanceID: "m-sf", Region: "cn-test",
	}); err != nil {
		t.Fatal(err)
	}

	var scans atomic.Int32
	orig := workspaceMachineSnapshotScan
	t.Cleanup(func() { workspaceMachineSnapshotScan = orig })
	workspaceMachineSnapshotScan = func(companyID, workspaceID string) (*workspaceMachineSnapshot, error) {
		scans.Add(1)
		time.Sleep(40 * time.Millisecond)
		return orig(companyID, workspaceID)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := computeWorkspaceMachineSnapshot("t1", "ws1")
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if n := scans.Load(); n != 1 {
		t.Fatalf("concurrent snapshot scans=%d want 1 (singleflight)", n)
	}
}
