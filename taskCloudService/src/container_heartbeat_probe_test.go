package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tracelog"
)

func TestProbeContainerHeartbeat_PropagatesFullTraceContext(t *testing.T) {
	tracelog.Init("task-cloud-service-test")

	var gotTrace, gotParent, gotTraceparent string
	var gotStatus int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		gotParent = r.Header.Get("X-Parent-Span-Id")
		gotTraceparent = r.Header.Get("traceparent")
		if tracelog.IsTraceIdOnlyRequest(r) {
			gotStatus = http.StatusBadRequest
			http.Error(w, `{"detail":"trace propagation incomplete"}`, http.StatusBadRequest)
			return
		}
		gotStatus = http.StatusOK
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	prevCred := cfg.CredentialServiceURL
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok-probe-test"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID:      "web-1783661000000-probehbtest01",
		SpanID:       "a1b2c3d4e5f67890",
		ParentSpanID: "895219d6c806919a",
	})
	cfgRow := &CloudServerConfig{
		CompanyID:   "850256677331562496",
		WorkspaceID: "861623708318031872",
		TaskID:      "task_probe_hb",
		ServerURL:   upstream.URL,
	}

	ok, ack := probeContainerHeartbeat(ctx, cfgRow, 42, "tok-probe-test")
	if !ok {
		t.Fatalf("probe_ok=false status=%d trace=%q parent=%q tp=%q", gotStatus, gotTrace, gotParent, gotTraceparent)
	}
	if ack == nil || *ack != 42 {
		t.Fatalf("ack=%v want 42", ack)
	}
	if gotTrace != "web-1783661000000-probehbtest01" {
		t.Fatalf("X-Trace-Id=%q", gotTrace)
	}
	if gotParent != "a1b2c3d4e5f67890" {
		t.Fatalf("X-Parent-Span-Id=%q want current span", gotParent)
	}
	if gotTraceparent == "" {
		t.Fatal("traceparent missing")
	}
}

func TestProbeContainerHeartbeat_LogsStructuredFailureOnBadStatus(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id"}`, http.StatusBadRequest)
	}))
	defer upstream.Close()

	prevCred := cfg.CredentialServiceURL
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"tok-probe-fail"}`)
	}))
	defer cred.Close()
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID:      "web-1783662000000-probefail01",
		SpanID:       "c1b2c3d4e5f67890",
		ParentSpanID: "895219d6c806919c",
	})
	cfgRow := &CloudServerConfig{
		CompanyID:   "t1",
		WorkspaceID: "w1",
		TaskID:      "task_probe_fail",
		ServerURL:   upstream.URL,
	}
	ok, ack := probeContainerHeartbeat(ctx, cfgRow, 9, "tok-probe-fail")
	if ok || ack != nil {
		t.Fatalf("expected probe failure, ok=%v ack=%v", ok, ack)
	}
	logged := buf.String()
	if !strings.Contains(logged, `"forward_stage":"container_heartbeat_probe_fail"`) {
		t.Fatalf("missing probe fail stage in log: %s", logged)
	}
	if !strings.Contains(logged, `"reason":"bad_status"`) {
		t.Fatalf("missing reason=bad_status: %s", logged)
	}
	if !strings.Contains(logged, `"status":400`) {
		t.Fatalf("missing status=400: %s", logged)
	}
	if !strings.Contains(logged, "trace propagation incomplete") {
		t.Fatalf("missing body detail: %s", logged)
	}
	if !strings.Contains(logged, `"task_id":"task_probe_fail"`) {
		t.Fatalf("missing task_id: %s", logged)
	}
}

func TestTruncateProbeBody(t *testing.T) {
	short := truncateProbeBody([]byte("  ok  "))
	if short != "ok" {
		t.Fatalf("short=%q", short)
	}
	long := strings.Repeat("x", probeFailBodyMax+20)
	got := truncateProbeBody([]byte(long))
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis, got len=%d", len(got))
	}
	if len([]rune(got)) < probeFailBodyMax {
		t.Fatalf("unexpected truncate len=%d", len(got))
	}
}

func TestHandleContainerHeartbeat_WithTraceContextProbeOK(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	setupCloudTestDB(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tracelog.IsTraceIdOnlyRequest(r) {
			http.Error(w, `{"detail":"trace propagation incomplete"}`, http.StatusBadRequest)
			return
		}
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(path, "validate") {
			_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"w1","task_id":"task1","expires_at":"2099-01-01"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"tok-hb"}`)
	}))
	defer cred.Close()
	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	seedCloudConfig(t, "t1", "w1", "task1", upstream.URL)
	row, err := loadCloudServerConfig("t1", "w1", "task1")
	if err != nil || row == nil {
		t.Fatalf("load config: %v row=%v", err, row)
	}
	resetHeartbeatSession(row)
	resetHeartbeatSessionsForTask("task1")

	postHB := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{
			"access_token": "tok-hb",
			"comment_id":   testLiveCommentID,
			"seq":          1,
			"message":      "unit-test",
		})
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/tenant/t1/workspace/w1/task/task1/comment/cmt1/cloud/server-container-token/heartbeat/",
			strings.NewReader(string(body)),
		)
		req.Header.Set("Content-Type", "application/json")
		ctx := tracelog.ContextWithCorrelation(req.Context(), tracelog.Correlation{
			TraceID:      "web-1783661000000-hbhandler01",
			SpanID:       "b1b2c3d4e5f67891",
			ParentSpanID: "895219d6c806919b",
		})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handleContainerInboundToken(rec, req, "t1", "w1", "task1", "heartbeat")
		return rec
	}
	rec := postHB()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	waitHeartbeatProbeSettled(t, row, 2*time.Second)
	rec2 := postHB()
	if rec2.Code != http.StatusOK {
		t.Fatalf("cached status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec2.Body.String())
	}
	if out["probe_ok"] != true {
		t.Fatalf("probe_ok=%v body=%s", out["probe_ok"], rec2.Body.String())
	}
	if out["bidirectional_ok"] != true {
		t.Fatalf("bidirectional_ok=%v body=%s", out["bidirectional_ok"], rec2.Body.String())
	}
}

func TestHandleContainerHeartbeat_CancelledRequestContextStillOK(t *testing.T) {
	// 模拟 TAS 超时断开：入站 ctx 已 cancel，探测与 SSE 发布仍须完成（WithoutCancel）
	tracelog.Init("task-cloud-service-test")
	setupCloudTestDB(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	// by-scope 不可达：须直接使用传入的 accessToken
	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	seedCloudConfig(t, "t1", "w1", "task_cancel", upstream.URL)
	row, err := loadCloudServerConfig("t1", "w1", "task_cancel")
	if err != nil || row == nil {
		t.Fatalf("load config: %v", err)
	}
	resetHeartbeatSession(row)

	baseCtx, cancel := context.WithCancel(context.Background())
	cancel()
	ctx := tracelog.ContextWithCorrelation(baseCtx, tracelog.Correlation{
		TraceID: "web-1783661000000-hbcancel01",
		SpanID:  "c1b2c3d4e5f67892",
	})
	rec := httptest.NewRecorder()
	handleContainerHeartbeat(rec, ctx, row, map[string]any{
		"access_token": "tok-cancel",
		"seq":          1,
		"message":      "cancelled-ctx",
	}, "t1", "w1", "task_cancel", "tok-cancel")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	waitHeartbeatProbeSettled(t, row, 2*time.Second)
	hbMu.Lock()
	sess := hbSessions[hbKey(row)]
	ok := sess != nil && sess.lastProbeOK && sess.lastDownlinkOK
	hbMu.Unlock()
	if !ok {
		t.Fatalf("background probe did not settle ok after cancelled ctx")
	}
}

func TestHandleContainerHeartbeat_SeqRestartResetsUplink(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	setupCloudTestDB(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "validate") {
			_, _ = io.WriteString(w, `{"valid":true,"company_id":"t1","workspace_id":"w1","task_id":"task_seq","expires_at":"2099-01-01"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"tok"}`)
	}))
	defer cred.Close()
	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = cred.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	seedCloudConfig(t, "t1", "w1", "task_seq", upstream.URL)
	row, _ := loadCloudServerConfig("t1", "w1", "task_seq")
	resetHeartbeatSession(row)

	post := func(seq int) map[string]any {
		body, _ := json.Marshal(map[string]any{"access_token": "tok", "comment_id": testLiveCommentID, "seq": seq})
		req := httptest.NewRequest(http.MethodPost, "/hb", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleContainerInboundToken(rec, req, "t1", "w1", "task_seq", "heartbeat")
		if rec.Code != http.StatusOK {
			t.Fatalf("seq=%d status=%d body=%s", seq, rec.Code, rec.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	if out := post(5); out["uplink_ok"] != true {
		t.Fatalf("first uplink_ok=%v", out["uplink_ok"])
	}
	// 容器重启后 seq 从 1 重计，须仍视为上行可达
	if out := post(1); out["uplink_ok"] != true {
		t.Fatalf("restart uplink_ok=%v body=%v", out["uplink_ok"], out)
	}
}

func TestPublishSSEMessage_IgnoresCancelledContext(t *testing.T) {
	prevKafka := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prevKafka })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := publishSSEMessage(ctx, "task_x", map[string]interface{}{
		"event_name": "container_heartbeat",
		"status":     "partial",
	}); err != nil {
		t.Fatalf("publishSSEMessage with cancelled ctx: %v", err)
	}
}

func waitHeartbeatProbeSettled(t *testing.T, cfgRow *CloudServerConfig, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	taskID := ""
	if cfgRow != nil {
		taskID = cfgRow.TaskID
	}
	for time.Now().Before(deadline) {
		hbMu.Lock()
		settled := false
		for key, sess := range hbSessions {
			if sess == nil || sess.lastProbeAt.IsZero() || sess.probeInFlight {
				continue
			}
			if cfgRow != nil && key == hbKey(cfgRow) {
				settled = true
				break
			}
			if taskID != "" && strings.HasSuffix(key, ":"+taskID) {
				settled = true
				break
			}
		}
		hbMu.Unlock()
		if settled {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("heartbeat probe did not settle within %s", timeout)
}

func TestHandleContainerHeartbeat_ReturnsBeforeSlowProbe(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	setupCloudTestDB(t)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	seedCloudConfig(t, "t1", "w1", "task_fast_hb", upstream.URL)
	row, err := loadCloudServerConfig("t1", "w1", "task_fast_hb")
	if err != nil || row == nil {
		t.Fatalf("load: %v", err)
	}
	resetHeartbeatSession(row)

	start := time.Now()
	rec := httptest.NewRecorder()
	handleContainerHeartbeat(rec, context.Background(), row, map[string]any{
		"access_token": "tok-fast",
		"seq":          1,
		"message":      "fast-path",
	}, "t1", "w1", "task_fast_hb", "tok-fast")
	elapsed := time.Since(start)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("heartbeat HTTP blocked on probe: elapsed=%s", elapsed)
	}
	waitHeartbeatProbeSettled(t, row, 2*time.Second)
}

func TestHandleContainerHeartbeat_ReusesProbeWithinTTL(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	setupCloudTestDB(t)

	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		seq := r.URL.Query().Get("seq")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","ack":`+seq+`}`)
	}))
	defer upstream.Close()

	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.CredentialServiceURL = prevCred })

	seedCloudConfig(t, "t1", "w1", "task_reuse_hb", upstream.URL)
	row, err := loadCloudServerConfig("t1", "w1", "task_reuse_hb")
	if err != nil || row == nil {
		t.Fatalf("load: %v", err)
	}
	resetHeartbeatSession(row)

	post := func(seq int) {
		rec := httptest.NewRecorder()
		handleContainerHeartbeat(rec, context.Background(), row, map[string]any{
			"access_token": "tok-reuse",
			"seq":          seq,
		}, "t1", "w1", "task_reuse_hb", "tok-reuse")
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
	post(1)
	waitHeartbeatProbeSettled(t, row, 2*time.Second)
	post(2)
	time.Sleep(50 * time.Millisecond)
	if hits != 1 {
		t.Fatalf("probe hits=%d want 1 (reuse within TTL)", hits)
	}
}
