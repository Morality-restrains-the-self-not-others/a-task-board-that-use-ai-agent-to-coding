package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tracelog"
)

// TestBootstrapCommentCSCRuntime_MockPlatformBlocked 验证 platform=mock/空 → 报错拦截：
// 不写 instance_id、不触发任何启动（本机容器模拟启动已移除）。
func TestBootstrapCommentCSCRuntime_MockPlatformBlocked(t *testing.T) {
	setupCloudTestDB(t)
	prevURL := cfg.ContainerGatewayURL
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() {
		cfg.ContainerGatewayURL = prevURL
		cfg.InternalSecret = prevSecret
	})

	var hits atomic.Int32
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer gw.Close()
	cfg.ContainerGatewayURL = gw.URL

	for i, platform := range []string{"mock", ""} {
		base := CloudServerConfig{
			ID:          genID("csc"),
			CompanyID:   "t1",
			WorkspaceID: "w1",
			TaskID:      fmt.Sprintf("taskBoot%d", i),
			CommentID:   "cmtBoot",
			Platform:    platform,
		}
		if err := upsertCloudServerConfig(base); err != nil {
			t.Fatal(err)
		}
		out, err := bootstrapCommentCSCRuntime(&base)
		if err == nil {
			t.Fatalf("platform=%q: expected error, got nil (out=%+v)", platform, out)
		}
		if !strings.Contains(err.Error(), "配置云平台") {
			t.Fatalf("platform=%q: err=%v", platform, err)
		}
		// 拦截后不得写入 mock- instance_id（保持 CSC 原状，由 reachability/release 收敛）
		var got CloudServerConfig
		if qerr := db.QueryRow(
			`SELECT instance_id, server_url, platform FROM cloud_server_configs WHERE id=?`, base.ID,
		).Scan(&got.InstanceID, &got.ServerURL, &got.Platform); qerr != nil {
			t.Fatal(qerr)
		}
		if strings.HasPrefix(trim(got.InstanceID), "mock-") {
			t.Fatalf("platform=%q: instance_id should stay untouched, got %q", platform, got.InstanceID)
		}
		if trim(got.ServerURL) != "" {
			t.Fatalf("platform=%q: server_url should stay empty, got %q", platform, got.ServerURL)
		}
	}

	// 等待一个调度周期，确认未触发任何网关/启动请求
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if hits.Load() != 0 {
		t.Fatalf("expected no start trigger, got %d hits", hits.Load())
	}
}

// TestLoadStartEventPayload_CommentScopedFallback 验证 OPT-20260814-019：
// 仅有评论级 start 事件（comment_id 非空）而无任务级事件时，loadStartEventPayload
// 仍能取到评论级 payload，评论 CSC bootstrap 不再报「no task-level start event payload」
// 而反复 failed → DLT 空转。
func TestLoadStartEventPayload_CommentScopedFallback(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t1"
	taskID := "taskNoTaskLevelStart"
	commentID := "cmtOnlyStart"

	evData := map[string]interface{}{
		"image_invoker_user_id": "u-1",
		"container_image_id":    "img-comment",
		"vpc_id":                "vpc-1",
		"security_group_id":     "sg-1",
		"vswitch_id":            "vsw-1",
		"region_id":             "cn-hongkong",
	}
	if _, _, err := insertPendingStartEvent(companyID, "w1", taskID, "m1", commentID, evData); err != nil {
		t.Fatal(err)
	}

	// 评论级存在 → 取到评论级 payload（不再 nil）
	got := loadStartEventPayload(companyID, taskID, commentID)
	if got == nil {
		t.Fatalf("expected comment-scoped start event payload, got nil (bootstrap would fail no-task-level)")
	}
	if v := strField(got, "container_image_id"); v != "img-comment" {
		t.Fatalf("expected comment event image, got %q", v)
	}

	// 无任务级事件 → loadTaskStartEventPayload 语义不变，仍返回 nil
	if tl := loadTaskStartEventPayload(companyID, taskID); tl != nil {
		t.Fatalf("expected no task-level payload, got %v", tl)
	}
}

// 第二条 @镜像 评论创建时只有第一条的 comment-scoped start 事件，没有任务级
// comment_id=” 行。bootstrap 必须复用同任务最近一次 start 载荷，否则
// 「no start event payload」→ 不走 start-vm → 无独立启动 TraceId。
func TestLoadStartEventPayload_SiblingCommentFallback(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t1"
	taskID := "taskSiblingStart"
	firstComment := "cmt_first_start"
	secondComment := "cmt_second_no_event"

	evData := map[string]interface{}{
		"image_invoker_user_id": "u-sib",
		"container_image_id":    "img-first-comment",
		"vpc_id":                "vpc-sib",
		"security_group_id":     "sg-sib",
		"vswitch_id":            "vsw-sib",
		"region_id":             "cn-hongkong",
	}
	if _, _, err := insertPendingStartEvent(companyID, "w1", taskID, "m1", firstComment, evData); err != nil {
		t.Fatal(err)
	}

	got := loadStartEventPayload(companyID, taskID, secondComment)
	if got == nil {
		t.Fatal("expected sibling comment start payload, got nil (second @镜像 bootstrap would skip start-vm)")
	}
	if v := strField(got, "container_image_id"); v != "img-first-comment" {
		t.Fatalf("expected first comment image, got %q", v)
	}
	if tl := loadTaskStartEventPayload(companyID, taskID); tl != nil {
		t.Fatalf("task-level payload must stay empty, got %v", tl)
	}
}

// OPT-20260816-044：sibling start 回退必须走结构化 logInfo，且 trace_id 独立（≠ task_id），
// 便于按页面启动 TraceId 在 Loki 检索。
func TestLoadStartEventPayload_SiblingFallbackEmitsTraceID(t *testing.T) {
	setupCloudTestDB(t)
	companyID := "t1"
	taskID := "taskSiblingTrace"
	firstComment := "cmt_first_trace"
	secondComment := "cmt_second_no_event_trace"

	evData := map[string]interface{}{
		"image_invoker_user_id": "u-sib-trace",
		"container_image_id":    "img-first-comment-trace",
		"vpc_id":                "vpc-sib-trace",
		"region_id":             "cn-hongkong",
	}
	if _, _, err := insertPendingStartEvent(companyID, "w1", taskID, "m1", firstComment, evData); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	got := loadStartEventPayload(companyID, taskID, secondComment)
	if got == nil {
		t.Fatal("expected sibling comment start payload, got nil")
	}

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected sibling fallback log line")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("json: %v line=%s", err, line)
	}
	msg, _ := payload["msg"].(string)
	if !strings.Contains(msg, "event=start_event_payload_sibling_fallback") {
		t.Fatalf("msg=%q want sibling fallback event", msg)
	}
	tid, _ := payload["trace_id"].(string)
	if tid == "" {
		t.Fatalf("trace_id missing in %s", line)
	}
	if tid == taskID {
		t.Fatalf("trace_id=%q must not equal task_id", tid)
	}
}

// 回归：评论 CSC bootstrap 自调用 start-vm 不得只带 X-Trace-Id。
// 根因：shareLib tracelog.Middleware RejectTraceIdOnlyHTTP → 400
// 「trace propagation incomplete」，binding 永久 Starting（TraceId 3b98d31be6e65e1f720b4692）。
func TestPostCommentCSCStartVM_SendsParentSpanWithTraceID(t *testing.T) {
	setupCloudTestDB(t)

	var (
		gotTrace       string
		gotParent      string
		gotTraceParent string
		hits           atomic.Int32
	)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port
	prevPort := cfg.Port
	cfg.Port = port
	t.Cleanup(func() { cfg.Port = prevPort })

	srv := &http.Server{Handler: tracelog.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		gotTrace = r.Header.Get(tracelog.Header)
		gotParent = r.Header.Get(tracelog.ParentSpanHeader)
		gotTraceParent = r.Header.Get("traceparent")
		if !strings.Contains(r.URL.Path, "/api/cloud/compute/start-vm/") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	companyID := "t1"
	workspaceID := "w1"
	taskID := "taskBootTraceHdr"
	commentID := "cmtBootTraceHdr"
	startTrace := "boot-start-trace-aabbccdd"

	evData := map[string]interface{}{
		"image_invoker_user_id": "u-boot",
		"container_image_id":    "img-boot",
		"vpc_id":                "vpc-boot",
		"security_group_id":     "sg-boot",
		"vswitch_id":            "vsw-boot",
		"region_id":             "cn-qingdao",
	}
	if _, _, err := insertPendingStartEvent(companyID, workspaceID, taskID, "m1", commentID, evData); err != nil {
		t.Fatal(err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	persistCommentBindingStartTraceID(taskID, commentID, startTrace)
	_ = updateCommentContainerBindingStatus(b.ID, ccbStatusStarting)

	csc := &CloudServerConfig{
		ID:                 "csc-boot-trace",
		CompanyID:          companyID,
		WorkspaceID:        workspaceID,
		TaskID:             taskID,
		CommentID:          commentID,
		Platform:           "aliyun",
		Region:             "cn-qingdao",
		ImageInvokerUserID: "u-boot",
	}
	if err := postCommentCSCStartVM(csc); err != nil {
		t.Fatalf("postCommentCSCStartVM: %v (likely missing X-Parent-Span-Id → RejectTraceIdOnlyHTTP)", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("start-vm hits=%d want 1", hits.Load())
	}
	if gotTrace == "" {
		t.Fatal("X-Trace-Id missing on bootstrap outbound")
	}
	if gotTrace != startTrace {
		t.Fatalf("X-Trace-Id=%q want binding start_trace_id %q", gotTrace, startTrace)
	}
	if gotParent == "" && gotTraceParent == "" {
		t.Fatal("must send X-Parent-Span-Id or traceparent (RejectTraceIdOnlyHTTP)")
	}
}

// 回归：bootstrap 收到 start-vm 4xx 时须把 binding 从 starting 收口为 failed，
// 避免 UI 永久显示「启动中 / Starting」。
func TestTriggerCommentCSCStartBootstrap_MarksFailedOn4xx(t *testing.T) {
	setupCloudTestDB(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port
	prevPort := cfg.Port
	cfg.Port = port
	t.Cleanup(func() { cfg.Port = prevPort })

	srv := &http.Server{Handler: tracelog.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"forced 400 for permanent fail test"}`))
	}))}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	companyID := "t1"
	workspaceID := "w1"
	taskID := "taskBootFail4xx"
	commentID := "cmtBootFail4xx"
	evData := map[string]interface{}{
		"image_invoker_user_id": "u-fail",
		"container_image_id":    "img-fail",
		"vpc_id":                "vpc-fail",
		"security_group_id":     "sg-fail",
		"vswitch_id":            "vsw-fail",
		"region_id":             "cn-qingdao",
	}
	if _, _, err := insertPendingStartEvent(companyID, workspaceID, taskID, "m1", commentID, evData); err != nil {
		t.Fatal(err)
	}
	b, err := insertCommentContainerBinding(companyID, taskID, commentID, ccbExecutionIndependent, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := markCommentContainerBindingStartingWithCSC(b.ID, "mock-fail", "csc-fail-4xx"); err != nil {
		t.Fatal(err)
	}

	csc := &CloudServerConfig{
		ID:                 "csc-fail-4xx",
		CompanyID:          companyID,
		WorkspaceID:        workspaceID,
		TaskID:             taskID,
		CommentID:          commentID,
		Platform:           "aliyun",
		Region:             "cn-qingdao",
		ImageInvokerUserID: "u-fail",
	}
	triggerCommentCSCStartBootstrapAsync(csc)

	deadline := time.Now().Add(3 * time.Second)
	var got *CommentContainerBinding
	for time.Now().Before(deadline) {
		got, err = loadCommentContainerBinding(companyID, taskID, commentID)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil && got.Status == ccbStatusFailed {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if got == nil || got.Status != ccbStatusFailed {
		t.Fatalf("binding status=%v want failed after start-vm 4xx", got)
	}
}

func TestIsCommentCSCStartBootstrapPermanent(t *testing.T) {
	if !isCommentCSCStartBootstrapPermanent(fmt.Errorf(`start-vm status=400 body={"detail":"trace propagation incomplete"}`)) {
		t.Fatal("400 must be permanent")
	}
	if isCommentCSCStartBootstrapPermanent(fmt.Errorf("start-vm status=503 body=unavailable")) {
		t.Fatal("503 must not be treated as permanent")
	}
	if isCommentCSCStartBootstrapPermanent(fmt.Errorf("no start event payload (task-level or comment-scoped, event_type=start) for comment csc bootstrap")) {
		t.Fatal("missing payload must not be permanent: 任务级云平台 / start-vm 可能尚未落库，mock 首轮须保持 starting（OPT-20260812-010）")
	}
}
