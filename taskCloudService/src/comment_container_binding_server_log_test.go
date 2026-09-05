package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPublishTaskSSEPersistsServerSchedulingToBindingLogs：
// 服务器调度进度（start-vm SSE）须写入评论 binding 启动日志，冷打开可还原。
func TestPublishTaskSSEPersistsServerSchedulingToBindingLogs(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvLog")

	body := `{"comment_id":"cSrv","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvLog/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvLog", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	msg := "[-、task_taskSrvLog_cSrv] 检测到自动创建资源，正在准备前置资源创建任务..."
	if err := publishTaskSSE(context.Background(), "taskSrvLog", "cSrv", map[string]interface{}{
		"status":     "processing",
		"message":    msg,
		"progress":   35,
		"event_name": "server_status_update",
	}); err != nil {
		// Kafka 在单测中常不可用；持久化 binding log 不得依赖 Kafka 成功
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvLog")
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	found := false
	for i := range rows {
		if rows[i].CommentID == "cSrv" && rows[i].Stage == ccbStageServerScheduling && rows[i].Message == msg {
			found = true
			break
		}
	}
	if !found {
		got := make([]string, 0, len(rows))
		for i := range rows {
			got = append(got, fmt.Sprintf("%s:%s", rows[i].Stage, rows[i].Message))
		}
		t.Fatalf("missing server_scheduling log; got=%v", got)
	}
}

// 任务级 start-vm（无 comment_id）须扇出调度文案到活跃 binding，但不得附带同一 trace_id=（多评论须独立 TraceId）。
func TestPublishTaskSSEWithoutCommentIDFansOutToActiveBindingsWithTraceID(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvFan")

	body := `{"comment_id":"cFan","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvFan/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvFan", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	msg := "未找到匹配地域的运行环境"
	if err := publishTaskSSE(context.Background(), "taskSrvFan", "", map[string]interface{}{
		"status":     "error",
		"message":    msg,
		"progress":   0,
		"event_name": "server_status_update",
		"trace_id":   "task_taskSrvFan",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvFan")
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	found := false
	for i := range rows {
		if rows[i].CommentID == "cFan" &&
			rows[i].Stage == ccbStageServerFailed &&
			strings.Contains(rows[i].Message, msg) {
			if strings.Contains(rows[i].Message, "trace_id=") {
				t.Fatalf("fan-out must not copy the same trace_id onto every binding; msg=%s", rows[i].Message)
			}
			found = true
			break
		}
	}
	if !found {
		got := make([]string, 0, len(rows))
		for i := range rows {
			got = append(got, fmt.Sprintf("%s:%s", rows[i].Stage, rows[i].Message))
		}
		t.Fatalf("missing fan-out server_failed log; got=%v", got)
	}
}

func TestPublishTaskSSEWithoutCommentIDSkipsMissingCommentIDMessage(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvSkipCmt")

	body := `{"comment_id":"cSkip","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvSkipCmt/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvSkipCmt", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	if err := publishTaskSSE(context.Background(), "taskSrvSkipCmt", "", map[string]interface{}{
		"status":     "error",
		"message":    "缺少评论ID",
		"progress":   0,
		"event_name": "server_status_update",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvSkipCmt")
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	for i := range rows {
		if strings.Contains(rows[i].Message, "缺少评论ID") {
			t.Fatalf("must not fan-out 缺少评论ID; got=%s:%s", rows[i].Stage, rows[i].Message)
		}
	}
}

// 评论级 start-vm 成功后，failed binding 必须恢复为 starting，避免日志「启动成功」与红条「启动失败」并存。
func TestPublishTaskSSESuccessRecoversFailedBinding(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvRecover")

	body := `{"comment_id":"cRec","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvRecover/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvRecover", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	b, err := loadCommentContainerBinding("t1", "taskSrvRecover", "cRec")
	if err != nil {
		t.Fatal(err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, b.CSCID); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	if err := publishTaskSSE(context.Background(), "taskSrvRecover", "cRec", map[string]interface{}{
		"status":     "success",
		"message":    "aliyun服务器启动成功！",
		"progress":   100,
		"event_name": "server_status_update",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	got, err := loadCommentContainerBinding("t1", "taskSrvRecover", "cRec")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusStarting {
		t.Fatalf("binding status=%s want %s after comment-scoped start success", got.Status, ccbStatusStarting)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvRecover")
	if err != nil {
		t.Fatal(err)
	}
	foundStarted := false
	for i := range rows {
		if rows[i].CommentID == "cRec" && rows[i].Stage == ccbStageServerStarted &&
			strings.Contains(rows[i].Message, "aliyun服务器启动成功") {
			foundStarted = true
			break
		}
	}
	if !foundStarted {
		t.Fatalf("missing server_started log after recover; logs=%+v", rows)
	}
}

// 任务级（无 comment_id）启动成功不得把成功日志扇出到 failed binding，否则评论卡出现「启动成功」却仍红条。
func TestPublishTaskSSESuccessDoesNotFanOutToFailedBinding(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvNoFan")

	body := `{"comment_id":"cNoFan","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvNoFan/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvNoFan", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	b, err := loadCommentContainerBinding("t1", "taskSrvNoFan", "cNoFan")
	if err != nil {
		t.Fatal(err)
	}
	if err := markCommentContainerBindingFailed(b.ID, b.MockContainerName, b.CSCID); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	if err := publishTaskSSE(context.Background(), "taskSrvNoFan", "", map[string]interface{}{
		"status":     "success",
		"message":    "aliyun服务器启动成功！",
		"progress":   100,
		"event_name": "server_status_update",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	got, err := loadCommentContainerBinding("t1", "taskSrvNoFan", "cNoFan")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusFailed {
		t.Fatalf("failed binding must stay failed on task-level success fan-out, got %s", got.Status)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvNoFan")
	if err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if rows[i].CommentID == "cNoFan" && rows[i].Stage == ccbStageServerStarted {
			t.Fatalf("task-level success must not write server_started onto failed binding: %+v", rows[i])
		}
	}
}

// 评论级 start-vm 真正失败时才收口 failed；与「首轮 mock 保持 starting」对照。
func TestPublishTaskSSEErrorMarksStartingBindingFailed(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvErrFail")

	body := `{"comment_id":"cErr","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvErrFail/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvErrFail", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	if err := publishTaskSSE(context.Background(), "taskSrvErrFail", "cErr", map[string]interface{}{
		"status":     "error",
		"message":    "可用区已停售",
		"progress":   0,
		"event_name": "server_status_update",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	got, err := loadCommentContainerBinding("t1", "taskSrvErrFail", "cErr")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusFailed {
		t.Fatalf("binding status=%s want %s after comment-scoped start-vm error", got.Status, ccbStatusFailed)
	}
}

// start-vm 常早于 binding INSERT：错误 SSE 在无行时也必须落库 failed，
// 否则后续 advance 会把卡片卡在 starting，runtime-status 误报「启动已发起」。
func TestPublishTaskSSEErrorCreatesFailedBindingWhenMissing(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t-err-miss"
	workspaceID := "ws-err-miss"
	taskID := "taskErrMiss"
	commentID := "cmtErrMiss"
	errMsg := "调用镜像市场 API 失败: 镜像服务返回错误: 502"

	if err := publishTaskSSE(context.Background(), taskID, commentID, map[string]interface{}{
		"status":       "error",
		"message":      errMsg,
		"progress":     0,
		"event_name":   "server_status_update",
		"tenant_id":    companyID,
		"workspace_id": workspaceID,
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	got, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil || got == nil {
		t.Fatalf("binding must be created on start-vm error before INSERT, err=%v", err)
	}
	if got.Status != ccbStatusFailed {
		t.Fatalf("binding status=%s want %s (must not stay missing/starting)", got.Status, ccbStatusFailed)
	}

	if _, err := advanceCommentContainerBindings(companyID, taskID, workspaceID); err != nil {
		t.Fatalf("advance: %v", err)
	}
	after, err := loadCommentContainerBinding(companyID, taskID, commentID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != ccbStatusFailed {
		t.Fatalf("advance must not reopen failed start-vm binding, got %s", after.Status)
	}

	msg := missingCommentCSCRuntimeMessage(companyID, workspaceID, taskID, commentID)
	if !strings.Contains(msg, errMsg) {
		t.Fatalf("runtime message=%q want contain %q", msg, errMsg)
	}
	if strings.Contains(msg, "启动已发起") {
		t.Fatalf("must not keep in-progress copy after start-vm 4xx: %q", msg)
	}
}

// 无 tenant_id 的错误 SSE 先旁路暂存；随后 INSERT binding 必须 drain 为 failed。
func TestInsertBindingDrainsPendingStartError(t *testing.T) {
	setupCloudTestDB(t)
	taskID := "taskPendErr"
	commentID := "cmtPendErr"
	errMsg := "调用镜像市场 API 失败: 镜像服务返回错误: 502"

	stashPendingBindingStartError(taskID, commentID, map[string]interface{}{
		"status":       "error",
		"message":      errMsg,
		"event_name":   "server_status_update",
		"workspace_id": "ws-pend-err",
	})
	row, err := insertCommentContainerBinding("t1", taskID, commentID, ccbExecutionIndependent, "")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := loadCommentContainerBinding("t1", taskID, commentID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusFailed {
		t.Fatalf("status=%s want %s after drain pending start error (inserted id=%s)", got.Status, ccbStatusFailed, row.ID)
	}
}

// 旧容器镜像克隆失败文案不内嵌 URL：serverSchedulingLogMessage 须把 statusData.repo_url
// 插到「失败 <name>」之后（冒号前），与 onlineServiceJS formatBootstrapCloneRepoFailureMessage
// 的 `<name> <url>:` 格式对齐，冷打开从 binding 日志即可解析出 URL。
func TestServerSchedulingLogMessageInsertsRepoURLAfterFailedRepo(t *testing.T) {
	cases := []struct {
		name    string
		msg     string
		repoURL string
		want    string
	}{
		{
			name:    "insert after failed repo name before colon",
			msg:     "【项目克隆】(1/1) 失败 ram-work: git exit 128",
			repoURL: "https://github.com/foo/ram-work.git",
			want:    "【项目克隆】(1/1) 失败 ram-work https://github.com/foo/ram-work.git: git exit 128",
		},
		{
			name:    "new container already embeds url",
			msg:     "【项目克隆】(1/1) 失败 ram-work https://github.com/foo/ram-work.git: git exit 128",
			repoURL: "https://github.com/foo/ram-work.git",
			want:    "【项目克隆】(1/1) 失败 ram-work https://github.com/foo/ram-work.git: git exit 128",
		},
		{
			name:    "ssh url inserted",
			msg:     "失败 ram-work: git exit 128",
			repoURL: "git@github.com:foo/ram-work.git",
			want:    "失败 ram-work git@github.com:foo/ram-work.git: git exit 128",
		},
		{
			name:    "non-git repo_url ignored",
			msg:     "失败 ram-work: git exit 128",
			repoURL: "ram-work",
			want:    "失败 ram-work: git exit 128",
		},
		{
			name:    "empty repo_url unchanged",
			msg:     "失败 ram-work: git exit 128",
			repoURL: "",
			want:    "失败 ram-work: git exit 128",
		},
		{
			name:    "summary line without per-repo colon unchanged",
			msg:     "【项目克隆】失败：1/1 个仓库均失败：ram-work",
			repoURL: "https://github.com/foo/ram-work.git",
			want:    "【项目克隆】失败：1/1 个仓库均失败：ram-work",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := serverSchedulingLogMessage(map[string]interface{}{
				"message":  tc.msg,
				"repo_url": tc.repoURL,
				"trace_id": "",
			})
			if got != tc.want {
				t.Fatalf("serverSchedulingLogMessage=\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}
