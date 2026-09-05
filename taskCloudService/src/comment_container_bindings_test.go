package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupCommentContainerBindingTest configures the test DB, seeds a cloud config,
// installs a mock container gateway so async bootstrap calls succeed, and seeds a
// task-level start event so comment CSC bootstrap can derive image/network/hardware
// params (comment_csc_bootstrap.go loadStartEventPayload contract).
func setupCommentContainerBindingTest(t *testing.T, taskID string) {
	t.Helper()
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", taskID, "http://mock/ui/tok/")
	if _, _, err := insertPendingStartEvent("t1", "ws1", taskID, "m1", "", map[string]interface{}{
		"image_invoker_user_id": "u-1",
		"container_image_id":    "img-mock",
		"vpc_id":                "vpc-1",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"region_id":             "cn-test",
	}); err != nil {
		t.Fatalf("seed task start event: %v", err)
	}

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
}

func TestCommentContainerBindingsIndependentParallel(t *testing.T) {
	setupCommentContainerBindingTest(t, "task1")

	createBinding := func(commentID string) {
		t.Helper()
		body := `{"comment_id":"` + commentID + `","execution_mode":"independent"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/task1/comment-container-bindings/",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "task1", "", "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", commentID, rec.Code, rec.Body.String())
		}
	}

	createBinding("c1")
	createBinding("c2")

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/task1/comment-container-bindings/advance/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "task1", "", "advance/")
	if rec.Code != http.StatusOK {
		t.Fatalf("advance status=%d body=%s", rec.Code, rec.Body.String())
	}

	var advBody map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &advBody); err != nil {
		t.Fatal(err)
	}
	advanced, ok := advBody["advanced"].([]interface{})
	if !ok || len(advanced) != 2 {
		t.Fatalf("advanced=%v want 2 parallel", advBody["advanced"])
	}

	listReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/task/task1/comment-container-bindings/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(listRec, listReq, "t1", "task1", "", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	var listBody map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	bindings, ok := listBody["bindings"].([]interface{})
	if !ok || len(bindings) != 2 {
		t.Fatalf("bindings=%v", listBody["bindings"])
	}
	cscIDs := map[string]struct{}{}
	for _, raw := range bindings {
		item, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("binding item=%v", raw)
		}
		wantMock := buildCommentMockContainerName("task1", item["comment_id"].(string))
		if item["mock_container_name"] != wantMock {
			t.Fatalf("mock=%v want %s", item["mock_container_name"], wantMock)
		}
		st := fmt.Sprint(item["status"])
		if st != ccbStatusStarting {
			t.Fatalf("comment_id=%v status=%v want starting until reachability", item["comment_id"], st)
		}
		csc := trim(fmt.Sprint(item["csc_id"]))
		if csc == "" || csc == "<nil>" {
			t.Fatalf("comment_id=%v missing dedicated csc_id", item["comment_id"])
		}
		if _, dup := cscIDs[csc]; dup {
			t.Fatalf("duplicate csc_id=%s across parallel comments", csc)
		}
		cscIDs[csc] = struct{}{}
	}
	if len(cscIDs) != 2 {
		t.Fatalf("want 2 distinct csc_ids, got %d", len(cscIDs))
	}
}

func TestCommentContainerBindingsWaitPreviousBlocksUntilComplete(t *testing.T) {
	setupCommentContainerBindingTest(t, "task2")

	createBinding := func(commentID, mode string) {
		t.Helper()
		body := `{"comment_id":"` + commentID + `","execution_mode":"` + mode + `"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/task2/comment-container-bindings/",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "task2", "", "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", commentID, rec.Code, rec.Body.String())
		}
	}

	advance := func() map[string]interface{} {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/task2/comment-container-bindings/advance/", nil)
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "task2", "", "advance/")
		if rec.Code != http.StatusOK {
			t.Fatalf("advance status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}

	complete := func(commentID string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/task2/comment-container-bindings/"+commentID+"/complete/", nil)
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "task2", "", commentID+"/complete")
		if rec.Code != http.StatusOK {
			t.Fatalf("complete %s status=%d body=%s", commentID, rec.Code, rec.Body.String())
		}
	}

	createBinding("c1", ccbExecutionWaitPrevious)
	time.Sleep(1500 * time.Millisecond) // ensure c1.created_at < c2.created_at for stable ordering
	createBinding("c2", ccbExecutionWaitPrevious)

	first := advance()
	adv1, _ := first["advanced"].([]interface{})
	if len(adv1) != 1 {
		t.Fatalf("first advance advanced=%v want 1", first["advanced"])
	}
	b1, err := loadCommentContainerBinding("t1", "task2", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if b1.Status != ccbStatusStarting {
		t.Fatalf("c1 status=%s want starting (awaiting reachability)", b1.Status)
	}
	if trim(b1.CSCID) == "" {
		t.Fatalf("c1 missing csc_id while starting")
	}
	// 模拟 register-reachability：写入 server_url 后再次 advance 升为 running
	csc1, err := loadCloudServerConfigByID(b1.CSCID)
	if err != nil {
		t.Fatalf("load csc: %v", err)
	}
	csc1.ServerURL = "http://127.0.0.1:8765/"
	csc1.InstanceID = "mock-c1"
	if err := upsertCloudServerConfig(*csc1); err != nil {
		t.Fatalf("upsert csc: %v", err)
	}
	promote := advance()
	advPromote, _ := promote["advanced"].([]interface{})
	if len(advPromote) != 1 {
		t.Fatalf("promote advanced=%v want 1", promote["advanced"])
	}
	b1, err = loadCommentContainerBinding("t1", "task2", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if b1.Status != ccbStatusRunning {
		t.Fatalf("c1 status=%s want running after reachability", b1.Status)
	}
	b2, err := loadCommentContainerBinding("t1", "task2", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if b2.Status != ccbStatusWaitingPrevious {
		t.Fatalf("c2 status=%s want waiting_previous", b2.Status)
	}

	second := advance()
	adv2, _ := second["advanced"].([]interface{})
	if len(adv2) != 0 {
		t.Fatalf("second advance advanced=%v want 0", second["advanced"])
	}
	b2, err = loadCommentContainerBinding("t1", "task2", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if b2.Status != ccbStatusWaitingPrevious {
		t.Fatalf("c2 still blocked status=%s", b2.Status)
	}

	complete("c1")
	third := advance()
	adv3, _ := third["advanced"].([]interface{})
	if len(adv3) != 1 {
		t.Fatalf("third advance advanced=%v want 1", third["advanced"])
	}
	b2, err = loadCommentContainerBinding("t1", "task2", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if b2.Status != ccbStatusStarting {
		t.Fatalf("c2 status=%s want starting after predecessor complete", b2.Status)
	}
	if trim(b2.CSCID) == "" {
		t.Fatalf("c2 missing csc_id")
	}
	csc2, err := loadCloudServerConfigByID(b2.CSCID)
	if err != nil {
		t.Fatalf("load c2 csc: %v", err)
	}
	csc2.ServerURL = "http://127.0.0.1:8766/"
	csc2.InstanceID = "mock-c2"
	if err := upsertCloudServerConfig(*csc2); err != nil {
		t.Fatalf("upsert c2 csc: %v", err)
	}
	fourth := advance()
	adv4, _ := fourth["advanced"].([]interface{})
	if len(adv4) != 1 {
		t.Fatalf("fourth advance advanced=%v want 1", fourth["advanced"])
	}
	b2, err = loadCommentContainerBinding("t1", "task2", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if b2.Status != ccbStatusRunning {
		t.Fatalf("c2 status=%s want running after reachability", b2.Status)
	}
}

func TestCommentContainerBindingsEnsureUpsertsExecutionMode(t *testing.T) {
	setupCloudTestDB(t)

	body1 := `{"comment_id":"cUpsert","execution_mode":"wait_previous"}`
	req1 := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskUpsert/comment-container-bindings/",
		strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Auth-Tenant-Id", "t1")
	rec1 := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec1, req1, "t1", "taskUpsert", "", "")
	if rec1.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec1.Code, rec1.Body.String())
	}

	// 模拟已被串行阻塞
	b, err := loadCommentContainerBinding("t1", "taskUpsert", "cUpsert")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusWaitingPrevious); err != nil {
		t.Fatal(err)
	}

	body2 := `{"comment_id":"cUpsert","execution_mode":"independent"}`
	req2 := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskUpsert/comment-container-bindings/",
		strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	rec2 := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec2, req2, "t1", "taskUpsert", "", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("ensure upsert status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	updated, err := loadCommentContainerBinding("t1", "taskUpsert", "cUpsert")
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutionMode != ccbExecutionIndependent {
		t.Fatalf("execution_mode=%s want independent", updated.ExecutionMode)
	}
	if updated.Status != ccbStatusPending {
		t.Fatalf("status=%s want pending after mode flip from waiting_previous", updated.Status)
	}
}

func TestBuildCommentMockContainerName(t *testing.T) {
	got := buildCommentMockContainerName("13908509172117356865", "cmt_42")
	want := "task_13908509172117356865_cmt_42"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 任务 ID 已含 task_ 时不得二次拼接
	got2 := buildCommentMockContainerName("task_15666874162351520866", "cmt_15666877397783348868")
	want2 := "task_15666874162351520866_cmt_15666877397783348868"
	if got2 != want2 {
		t.Fatalf("prefixed task id: got %q want %q", got2, want2)
	}
	// bare 短 id（无 task_）仍加一层前缀
	got3 := buildCommentMockContainerName("task1", "cNew")
	want3 := "task_task1_cNew"
	if got3 != want3 {
		t.Fatalf("bare task1: got %q want %q", got3, want3)
	}
}

func TestCommentIDFromContainerNameAcceptsCanonicalAndLegacy(t *testing.T) {
	tid := "task_15666874162351520866"
	cid := "cmt_15666877397783348868"
	canonical := tid + "_" + cid
	legacy := "task_" + tid + "_" + cid
	if got := commentIDFromContainerName(tid, canonical); got != cid {
		t.Fatalf("canonical: got %q want %q", got, cid)
	}
	if got := commentIDFromContainerName(tid, legacy); got != cid {
		t.Fatalf("legacy: got %q want %q", got, cid)
	}
	if got := commentIDFromContainerName("task1", "task_task1_cNew"); got != "cNew" {
		t.Fatalf("bare task1: got %q", got)
	}
}

func TestResolveCommentMockContainerNameRewritesLegacyDoublePrefix(t *testing.T) {
	tid := "task_15666874162351520866"
	cid := "cmt_9"
	legacy := "task_" + tid + "_" + cid
	want := tid + "_" + cid
	if got := resolveCommentMockContainerName(tid, cid, legacy); got != want {
		t.Fatalf("legacy rewrite: got %q want %q", got, want)
	}
	if got := resolveCommentMockContainerName(tid, cid, want); got != want {
		t.Fatalf("canonical keep: got %q", got)
	}
	if got := resolveCommentMockContainerName("task1", "cNew", ""); got != "task_task1_cNew" {
		t.Fatalf("empty derive: got %q", got)
	}
}

func TestCommentContainerBindingsCreatePersistsContainerName(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"comment_id":"cNew","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/task1/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "task1", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	binding, _ := resp["binding"].(map[string]interface{})
	if binding == nil {
		t.Fatalf("create body=%s missing binding", rec.Body.String())
	}
	want := buildCommentMockContainerName("task1", "cNew")
	if binding["mock_container_name"] != want {
		t.Fatalf("mock_container_name=%v want %s", binding["mock_container_name"], want)
	}
	if binding["container_name"] != want {
		t.Fatalf("container_name=%v want %s", binding["container_name"], want)
	}
}

func TestCommentContainerBindingsWorkspaceComputeQueryTaskID(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "taskQ", "http://mock/ui/tok/")

	body := `{"comment_id":"cq1","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/ws1/cloud/compute/comment-container-bindings/?task_id=taskQ",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/comment-container-bindings/")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create via workspace compute status=%d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/comment-container-bindings/?task_id=taskQ", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Workspace-Id", "ws1")
	listRec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(listRec, listReq, "cloud/compute/comment-container-bindings/")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var listBody map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	bindings, _ := listBody["bindings"].([]interface{})
	if len(bindings) != 1 {
		t.Fatalf("bindings=%v want 1", listBody["bindings"])
	}
}
