package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRuntimeEventLogsAllowedBootstrapEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	cfg := &CloudServerConfig{TaskID: "task_rt_1", CompanyID: "t1", WorkspaceID: "w1"}
	handleRuntimeEvent(
		rec,
		context.Background(),
		cfg,
		map[string]any{
			"event":   "BOOTSTRAP_COMPLETE",
			"phase":   "post_listen",
			"message": "ok",
			"level":   "info",
			"fields":  map[string]any{"layer_id": "L1"},
		},
		"t1",
		"w1",
		"task_rt_1",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true || body["event"] != "BOOTSTRAP_COMPLETE" {
		t.Fatalf("body=%#v", body)
	}
}

func TestHandleRuntimeEventRejectsUnknownEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	handleRuntimeEvent(
		rec,
		context.Background(),
		&CloudServerConfig{TaskID: "task_x"},
		map[string]any{"event": "NOT_A_REAL_EVENT"},
		"t1",
		"w1",
		"task_x",
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unsupported event") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleRuntimeEventRequiresEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	handleRuntimeEvent(rec, context.Background(), &CloudServerConfig{}, map[string]any{}, "t", "w", "task")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestBootstrapRuntimeSSEStatus(t *testing.T) {
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_FAILED", ""); got != "container_bootstrap_failed" {
		t.Fatalf("got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_COMPLETE", ""); got != "container_bootstrap_complete" {
		t.Fatalf("got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_PHASE", ""); got != "" {
		t.Fatalf("PHASE without clone/task_detail must not publish, got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_PHASE", "feature_params_begin"); got != "" {
		t.Fatalf("unlisted PHASE must not publish, got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_PHASE", "clone_begin"); got != "container_bootstrap_progress" {
		t.Fatalf("clone_begin: got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_PHASE", "task_detail_begin"); got != "container_bootstrap_progress" {
		t.Fatalf("task_detail_begin: got=%q", got)
	}
	if got := bootstrapRuntimeSSEStatus("BOOTSTRAP_PHASE", "credentials_recovery_begin"); got != "container_bootstrap_progress" {
		t.Fatalf("credentials_recovery_begin: got=%q", got)
	}
}

func TestHandleRuntimeEventBootstrapFailedStillOK(t *testing.T) {
	rec := httptest.NewRecorder()
	cfg := &CloudServerConfig{TaskID: "task_bf_1", CompanyID: "t1", WorkspaceID: "w1", CommentID: "cmt_1"}
	handleRuntimeEvent(
		rec,
		context.Background(),
		cfg,
		map[string]any{
			"event":   "BOOTSTRAP_FAILED",
			"phase":   "task_detail_or_credentials",
			"message": "repo-clone-credentials 未返回完整；缺失仓库(1): https://github.com/ruandao/somanyad",
			"level":   "error",
		},
		"t1",
		"w1",
		"task_bf_1",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"event":"BOOTSTRAP_FAILED"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestFirstNonEmptyTraceIDPrefersBodyThenFields(t *testing.T) {
	if got := firstNonEmptyTraceID(context.Background(), map[string]any{"trace_id": " from-body "}); got != "from-body" {
		t.Fatalf("body trace_id: got=%q", got)
	}
	if got := firstNonEmptyTraceID(context.Background(), map[string]any{"traceId": "camel"}); got != "camel" {
		t.Fatalf("body traceId: got=%q", got)
	}
	if got := firstNonEmptyTraceID(context.Background(), map[string]any{
		"fields": map[string]any{"trace_id": "from-fields"},
	}); got != "from-fields" {
		t.Fatalf("fields: got=%q", got)
	}
	if got := firstNonEmptyTraceID(context.Background(), map[string]any{}); got != "" {
		t.Fatalf("empty: got=%q", got)
	}
	bodyWins := firstNonEmptyTraceID(context.Background(), map[string]any{
		"trace_id": "body-wins",
		"fields":   map[string]any{"trace_id": "fields-lose"},
	})
	if bodyWins != "body-wins" {
		t.Fatalf("body should win: got=%q", bodyWins)
	}
}
