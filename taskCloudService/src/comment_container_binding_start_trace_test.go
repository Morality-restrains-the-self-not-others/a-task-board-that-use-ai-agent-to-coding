package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPersistCommentBindingStartTraceIDIndependentPerComment(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskTrace")

	create := func(commentID string) {
		t.Helper()
		body := `{"comment_id":"` + commentID + `","execution_mode":"independent"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/taskTrace/comment-container-bindings/",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "taskTrace", "", "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", commentID, rec.Code, rec.Body.String())
		}
	}
	create("c1")
	create("c2")

	persistCommentBindingStartTraceID("taskTrace", "c1", "trace-c1-start")
	persistCommentBindingStartTraceID("taskTrace", "c2", "trace-c2-start")
	persistCommentBindingStartTraceID("taskTrace", "", "trace-must-not-fan-out")
	persistCommentBindingStartTraceID("taskTrace", "c1", "taskTrace")

	listReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/task/taskTrace/comment-container-bindings/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(listRec, listReq, "t1", "taskTrace", "", "")
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
	got := map[string]string{}
	for _, raw := range bindings {
		item, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("item=%v", raw)
		}
		cid := fmt.Sprint(item["comment_id"])
		got[cid] = fmt.Sprint(item["start_trace_id"])
	}
	if got["c1"] != "trace-c1-start" {
		t.Fatalf("c1 start_trace_id=%q (must keep independent id, reject task_id overwrite)", got["c1"])
	}
	if got["c2"] != "trace-c2-start" {
		t.Fatalf("c2 start_trace_id=%q", got["c2"])
	}
	if got["c1"] == got["c2"] {
		t.Fatalf("two comments must not share start_trace_id %q", got["c1"])
	}
}

func listBindingStartTraceIDs(t *testing.T, taskID string) map[string]string {
	t.Helper()
	listReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/task/"+taskID+"/comment-container-bindings/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(listRec, listReq, "t1", taskID, "", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var listBody map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	bindings, ok := listBody["bindings"].([]interface{})
	if !ok {
		t.Fatalf("bindings=%v", listBody["bindings"])
	}
	got := map[string]string{}
	for _, raw := range bindings {
		item, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("item=%v", raw)
		}
		got[fmt.Sprint(item["comment_id"])] = fmt.Sprint(item["start_trace_id"])
	}
	return got
}

func TestPersistCommentBindingStartTraceIDBeforeBindingRowExists(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskLateBind")

	persistCommentBindingStartTraceID("taskLateBind", "c-late", "trace-before-row")

	body := `{"comment_id":"c-late","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskLateBind/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskLateBind", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	got := listBindingStartTraceIDs(t, "taskLateBind")
	if got["c-late"] != "trace-before-row" {
		t.Fatalf("c-late start_trace_id=%q want trace-before-row (persist raced before INSERT)", got["c-late"])
	}
}

func TestPersistStartVmInstanceBindingFillsEmptyStartTraceID(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskHealTrace")
	body := `{"comment_id":"cmt_heal","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskHealTrace/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskHealTrace", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	now := "2026-08-16 00:00:00"
	if _, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, region, zone_id, authorization_id, created_at, updated_at)
		VALUES ('csc-heal-trace', 't1', 'ws1', 'taskHealTrace', 'cmt_heal', 'aliyun', '', 'cn-hangzhou', 'cn-hangzhou-i', 'auth1', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}

	if err := persistStartVmInstanceBinding(map[string]interface{}{
		"task_id":    "taskHealTrace",
		"comment_id": "cmt_heal",
		"csc_id":     "csc-heal-trace",
		"trace_id":   "heal-run-trace-001",
	}, "i-heal-1", "req-heal"); err != nil {
		t.Fatalf("persist instance: %v", err)
	}

	got := listBindingStartTraceIDs(t, "taskHealTrace")
	if got["cmt_heal"] != "heal-run-trace-001" {
		t.Fatalf("cmt_heal start_trace_id=%q want heal-run-trace-001 (heal/bind must persist independent trace)", got["cmt_heal"])
	}
}

func TestEnsureCommentBindingStartTraceIDKeepsExisting(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskKeepTrace")
	body := `{"comment_id":"cmt_keep","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskKeepTrace/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskKeepTrace", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	persistCommentBindingStartTraceID("taskKeepTrace", "cmt_keep", "already-independent")
	got := ensureCommentBindingStartTraceID("taskKeepTrace", "cmt_keep", "should-not-replace")
	if got != "already-independent" {
		t.Fatalf("got %q want already-independent", got)
	}
}

// OPT-20260816-035: 暂存改落 MySQL 旁路表后，模拟进程重启（清空进程内 map）仍能回填。
func TestPendingBindingStartTraceBackfillsAcrossRestart(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskRestartTrace")

	// 1. start-vm persist 早于 binding INSERT：UPDATE 0 行 → 暂存旁路表。
	persistCommentBindingStartTraceID("taskRestartTrace", "c-restart", "trace-before-restart")
	if got := loadBindingStartTraceID("taskRestartTrace", "c-restart"); got != "" {
		t.Fatalf("binding row should not exist yet, got start_trace_id=%q", got)
	}
	if got := loadPendingBindingStartTraceID("taskRestartTrace", "c-restart"); got != "trace-before-restart" {
		t.Fatalf("pending trace=%q want trace-before-restart", got)
	}

	// 2. 模拟进程重启：清空进程内 map，只剩 MySQL 旁路表。
	pendingBindingStartTrace = sync.Map{}

	// 3. 创建 binding → INSERT 后 drain 应从旁路表回填。
	body := `{"comment_id":"c-restart","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskRestartTrace/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskRestartTrace", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	got := listBindingStartTraceIDs(t, "taskRestartTrace")
	if got["c-restart"] != "trace-before-restart" {
		t.Fatalf("c-restart start_trace_id=%q want trace-before-restart (cross-restart backfill)", got["c-restart"])
	}
	if pending := loadPendingBindingStartTraceID("taskRestartTrace", "c-restart"); pending != "" {
		t.Fatalf("pending side row must be drained after backfill, still got %q", pending)
	}
}

// OPT-20260816-035: 旁路表存的是 task_id 的陈旧值应被清理而不写库。
func TestPendingBindingStartTraceRejectsTaskIDValue(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskJunkTrace")

	stashPendingBindingStartTraceID("taskJunkTrace", "c-junk", "taskJunkTrace")
	// 直接建行：drain 时遇到等于 task_id 的陈旧值应删除而非写入。
	if _, err := db.Exec(
		`INSERT INTO cloud_comment_container_bindings(id, company_id, task_id, comment_id, execution_mode, status, mock_container_name, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		"ccb-junk", "t1", "taskJunkTrace", "c-junk", "independent", "pending",
		buildCommentMockContainerName("taskJunkTrace", "c-junk"),
		"2026-08-17 00:00:00", "2026-08-17 00:00:00",
	); err != nil {
		t.Fatal(err)
	}
	drainPendingBindingStartTraceID("taskJunkTrace", "c-junk")

	if got := loadBindingStartTraceID("taskJunkTrace", "c-junk"); got != "" {
		t.Fatalf("task_id must not be written as start_trace_id, got %q", got)
	}
	if pending := loadPendingBindingStartTraceID("taskJunkTrace", "c-junk"); pending != "" {
		t.Fatalf("stale pending row must be deleted, still got %q", pending)
	}
}

func TestListBindingsBackfillsEmptyStartTraceWhenCSCPresent(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskListBackfill")
	body := `{"comment_id":"cmt_bf","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskListBackfill/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskListBackfill", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := db.Exec(
		`UPDATE cloud_comment_container_bindings SET csc_id=? WHERE task_id=? AND comment_id=?`,
		"csc_bf_1", "taskListBackfill", "cmt_bf",
	); err != nil {
		t.Fatal(err)
	}

	got := listBindingStartTraceIDs(t, "taskListBackfill")
	tid := got["cmt_bf"]
	if tid == "" || tid == "taskListBackfill" {
		t.Fatalf("list must backfill independent start_trace_id, got %q", tid)
	}
}
