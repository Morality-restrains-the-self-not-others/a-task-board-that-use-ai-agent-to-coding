package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestCommentContainerBindingLogsTimeline OPT-20260809-011 回归：
// 评论容器启动阶段事件完整落库，list API 按时间升序返回 logs 字段。
// 阶段序列：pending → starting → csc_allocated → running → completed。
func TestCommentContainerBindingLogsTimeline(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskLogs")

	// 1. create → 初始 pending 日志
	body := `{"comment_id":"cl1","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskLogs/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskLogs", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	logs := func() []map[string]interface{} {
		t.Helper()
		rows, err := listCommentContainerBindingLogs("t1", "taskLogs")
		if err != nil {
			t.Fatalf("list logs: %v", err)
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for i := range rows {
			out = append(out, commentContainerBindingLogToJSON(&rows[i]))
		}
		return out
	}

	assertHasStage := func(stage string) {
		t.Helper()
		for _, l := range logs() {
			if fmt.Sprint(l["stage"]) == stage {
				return
			}
		}
		t.Fatalf("missing log stage=%s, got=%v", stage, logs())
	}

	assertHasStage(ccbStatusPending)

	advance := func() {
		t.Helper()
		advReq := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/taskLogs/comment-container-bindings/advance/", nil)
		advReq.Header.Set("X-Auth-Tenant-Id", "t1")
		advRec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(advRec, advReq, "t1", "taskLogs", "", "advance/")
		if advRec.Code != http.StatusOK {
			t.Fatalf("advance status=%d body=%s", advRec.Code, advRec.Body.String())
		}
	}

	// 2. advance → starting + csc_allocated（云侧未登记 reachability，保持 starting）
	advance()
	assertHasStage(ccbStatusStarting)
	assertHasStage(ccbStageCSCAllocated)

	// 3. 模拟 register-reachability → promote running
	b, err := loadCommentContainerBinding("t1", "taskLogs", "cl1")
	if err != nil {
		t.Fatal(err)
	}
	csc, err := loadCloudServerConfigByID(b.CSCID)
	if err != nil {
		t.Fatalf("load csc: %v", err)
	}
	csc.ServerURL = "http://127.0.0.1:9876/"
	csc.InstanceID = "mock-cl1"
	if err := upsertCloudServerConfig(*csc); err != nil {
		t.Fatalf("upsert csc: %v", err)
	}
	advance()
	assertHasStage(ccbStatusRunning)

	// 4. complete → completed 日志
	compReq := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskLogs/comment-container-bindings/cl1/complete/", nil)
	compReq.Header.Set("X-Auth-Tenant-Id", "t1")
	compRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(compRec, compReq, "t1", "taskLogs", "", "cl1/complete")
	if compRec.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", compRec.Code, compRec.Body.String())
	}
	assertHasStage(ccbStatusCompleted)

	// 5. list API 返回 logs 字段且阶段按时间升序
	listReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/task/taskLogs/comment-container-bindings/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(listRec, listReq, "t1", "taskLogs", "", "")
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
	item := bindings[0].(map[string]interface{})
	apiLogs, ok := item["logs"].([]interface{})
	if !ok || len(apiLogs) == 0 {
		t.Fatalf("list logs field missing/empty: %v", item["logs"])
	}
	var stages []string
	for _, raw := range apiLogs {
		l := raw.(map[string]interface{})
		stages = append(stages, fmt.Sprint(l["stage"]))
	}
	wantSeq := []string{ccbStatusPending, ccbStatusStarting, ccbStageCSCAllocated, ccbStatusRunning, ccbStatusCompleted}
	if len(stages) != len(wantSeq) {
		t.Fatalf("stages=%v want %v", stages, wantSeq)
	}
	for i, want := range wantSeq {
		if stages[i] != want {
			t.Fatalf("stage[%d]=%s want %s (seq=%v)", i, stages[i], want, stages)
		}
	}
	firstLog := apiLogs[0].(map[string]interface{})
	ca := fmt.Sprint(firstLog["created_at"])
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`).MatchString(ca) {
		t.Fatalf("logs[0].created_at=%q want RFC3339 UTC with Z", ca)
	}
	parsed, err := time.Parse(time.RFC3339, ca)
	if err != nil {
		t.Fatalf("created_at parse: %v", err)
	}
	if parsed.Location() != time.UTC && parsed.UTC().Format(time.RFC3339) != ca {
		t.Fatalf("created_at not UTC: %s loc=%s", ca, parsed.Location())
	}
	if d := time.Since(parsed.UTC()).Abs(); d > 2*time.Minute {
		t.Fatalf("created_at %s too far from now UTC (delta=%s)", ca, d)
	}
}

// TestCommentContainerBindingMockKeepsStarting：首轮 advance 时评论 CSC 常为 mock
// （任务级云平台 / start-vm 尚未落库）。不得因此收口 failed，否则后续阿里云启动成功
// 会出现「日志成功 + 红条失败」。mock 仅表示暂不可启机，binding 保持 starting。
func TestCommentContainerBindingMockKeepsStarting(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "ws1", "taskPermFail", "")

	body := `{"comment_id":"cPermFail","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskPermFail/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskPermFail", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	advReq := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskPermFail/comment-container-bindings/advance/", nil)
	advReq.Header.Set("X-Auth-Tenant-Id", "t1")
	advRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(advRec, advReq, "t1", "taskPermFail", "", "advance/")
	if advRec.Code != http.StatusOK {
		t.Fatalf("advance status=%d body=%s", advRec.Code, advRec.Body.String())
	}

	// 等待异步 bootstrap 尝试落定：首轮无 start 事件载荷时，即使 start-vm 自调用
	// 立即失败也不得收口 failed（保持 starting，等待任务级 start 事件落库）。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := listCommentContainerBindingLogs("t1", "taskPermFail")
		if err != nil {
			t.Fatal(err)
		}
		failed := false
		for i := range rows {
			if rows[i].Stage == ccbStatusFailed {
				failed = true
			}
		}
		if failed {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}

	b, err := loadCommentContainerBinding("t1", "taskPermFail", "cPermFail")
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want %s (mock 首轮不得收口 failed)", b.Status, ccbStatusStarting)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskPermFail")
	if err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if rows[i].Stage == ccbStatusFailed {
			t.Fatalf("must not write failed stage on first mock advance; logs=%+v", rows)
		}
	}
}
