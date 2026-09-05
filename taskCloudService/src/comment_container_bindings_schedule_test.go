package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCcbIsBlockedByDependenciesAllPreviousAndSpecific(t *testing.T) {
	rows := []CommentContainerBinding{
		{CommentID: "c1", Status: ccbStatusCompleted, ExecutionMode: ccbExecutionWaitPrevious},
		{CommentID: "c2", Status: ccbStatusRunning, ExecutionMode: ccbExecutionIndependent},
		{CommentID: "c3", Status: ccbStatusPending, ExecutionMode: ccbExecutionWaitPrevious, DependsOnCommentID: ""},
		{CommentID: "c4", Status: ccbStatusPending, ExecutionMode: ccbExecutionWaitPrevious, DependsOnCommentID: "c1"},
	}
	if !ccbIsBlockedByDependencies(rows, 2) {
		t.Fatal("c3 should wait for all previous including running c2")
	}
	if ccbIsBlockedByDependencies(rows, 3) {
		t.Fatal("c4 depends only on completed c1")
	}
	rows[3].DependsOnCommentID = "c1,c2"
	if !ccbIsBlockedByDependencies(rows, 3) {
		t.Fatal("c4 should wait for c2 still running")
	}
}

func TestEnsureCommentCloudServerConfigDistinctPerComment(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "taskMulti", "http://mock/ui/tok/")

	c1, err := ensureCommentCloudServerConfig("t1", "ws1", "taskMulti", "c1")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := ensureCommentCloudServerConfig("t1", "ws1", "taskMulti", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if c1.ID == c2.ID {
		t.Fatalf("expected distinct csc ids, both %s", c1.ID)
	}
	if c1.CommentID != "c1" || c2.CommentID != "c2" {
		t.Fatalf("comment_id mismatch c1=%q c2=%q", c1.CommentID, c2.CommentID)
	}
	taskLevel, err := loadCloudServerConfig("t1", "ws1", "taskMulti")
	if err != nil {
		t.Fatal(err)
	}
	if taskLevel.ID != "cfg1" {
		t.Fatalf("task-level csc=%s want cfg1", taskLevel.ID)
	}
}

func TestCommentContainerBindingsDemoteDuplicateCSCOnAdvance(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "taskDup", "http://mock/ui/tok/")
	if _, _, err := insertPendingStartEvent("t1", "ws1", "taskDup", "m1", "", map[string]interface{}{
		"image_invoker_user_id": "u-1",
		"container_image_id":    "img-mock",
		"vpc_id":                "vpc-1",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"region_id":             "cn-test",
	}); err != nil {
		t.Fatalf("seed task start event: %v", err)
	}

	// Install a mock container gateway so async bootstrap calls succeed.
	prevURL := cfg.ContainerGatewayURL
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	t.Cleanup(func() {
		gw.Close()
		cfg.ContainerGatewayURL = prevURL
		cfg.InternalSecret = prevSecret
	})
	cfg.ContainerGatewayURL = gw.URL

	_, err := insertCommentContainerBinding("t1", "taskDup", "c1", ccbExecutionWaitPrevious, "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond) // ensure c1.created_at < c2.created_at for stable ordering
	_, err = insertCommentContainerBinding("t1", "taskDup", "c2", ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	b1, err := loadCommentContainerBinding("t1", "taskDup", "c1")
	if err != nil {
		t.Fatal(err)
	}
	b2, err := loadCommentContainerBinding("t1", "taskDup", "c2")
	if err != nil {
		t.Fatal(err)
	}
	// 模拟历史错误：两个 live binding 都挂了同一 CSC
	if err := markCommentContainerBindingRunning(b1.ID, "mock_c1", "cfg1"); err != nil {
		t.Fatal(err)
	}
	if err := markCommentContainerBindingRunning(b2.ID, "mock_c2", "cfg1"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/task/taskDup/comment-container-bindings/advance/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskDup", "", "advance/")
	if rec.Code != http.StatusOK {
		t.Fatalf("advance status=%d body=%s", rec.Code, rec.Body.String())
	}

	b1, err = loadCommentContainerBinding("t1", "taskDup", "c1")
	if err != nil {
		t.Fatal(err)
	}
	b2, err = loadCommentContainerBinding("t1", "taskDup", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if b1.Status != ccbStatusRunning || b1.CSCID != "cfg1" {
		t.Fatalf("keeper=%+v", b1)
	}
	// demote 后 advance 会为 c2 ensure 独立 CSC 并保持 starting（待 reachability）
	if b2.Status != ccbStatusStarting || trim(b2.CSCID) == "" || b2.CSCID == "cfg1" {
		t.Fatalf("re-attached=%+v want starting with distinct csc", b2)
	}
}
