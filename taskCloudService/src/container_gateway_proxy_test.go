package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

func TestCopyProxyHeaders_ForwardsSpanPropagation(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	src := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/container-layer-files/", nil)
	src.Header.Set("Authorization", "Token test-token")
	src.Header.Set("X-User-Id", "user-1")
	src.Header.Set("Accept", "application/json")
	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{
		TraceID:      "web-1783583558132-0qwxrrap6d09",
		SpanID:       "a1b2c3d4e5f67890",
		ParentSpanID: "895219d6c806919a",
	})
	src = src.WithContext(ctx)

	dst := httptest.NewRequest(http.MethodGet, "http://container-gateway.example/api/tenant/t1/workspace/w1/task/task1/cloud/compute/container-layer-files/", nil)
	copyProxyHeaders(src, dst)

	if got := dst.Header.Get("X-Trace-Id"); got != "web-1783583558132-0qwxrrap6d09" {
		t.Fatalf("X-Trace-Id = %q", got)
	}
	if got := dst.Header.Get("X-Parent-Span-Id"); got != "a1b2c3d4e5f67890" {
		t.Fatalf("X-Parent-Span-Id = %q want current span as parent", got)
	}
	if got := dst.Header.Get("traceparent"); got == "" {
		t.Fatal("traceparent missing")
	}
	if got := dst.Header.Get("Authorization"); got != "Token test-token" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestCopyProxyHeaders_ForwardsAuthUserIdWithContainerGatewaySecret(t *testing.T) {
	prev := cfg.ContainerGatewayInternalSecret
	cfg.ContainerGatewayInternalSecret = "tcg-secret"
	t.Cleanup(func() { cfg.ContainerGatewayInternalSecret = prev })
	src := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/container-layer-git-push/", nil)
	src.Header.Set("X-Auth-User-Id", "877397583960502272")
	dst := httptest.NewRequest(http.MethodPost, "http://gw.example/api/cloud/compute/container-layer-git-push/", nil)
	copyProxyHeaders(src, dst)
	if got := dst.Header.Get("X-TaskContainerGateway-Internal-Secret"); got != "tcg-secret" {
		t.Fatalf("X-TaskContainerGateway-Internal-Secret=%q", got)
	}
	if got := dst.Header.Get("X-Auth-User-Id"); got != "877397583960502272" {
		t.Fatalf("X-Auth-User-Id=%q want browser user forwarded with internal secret", got)
	}
}

func TestCopyProxyHeaders_SetsContainerGatewayInternalSecret(t *testing.T) {
	prev := cfg.ContainerGatewayInternalSecret
	cfg.ContainerGatewayInternalSecret = "tcg-secret"
	t.Cleanup(func() { cfg.ContainerGatewayInternalSecret = prev })
	src := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/container-layer-graph/", nil)
	dst := httptest.NewRequest(http.MethodGet, "http://gw.example/api/cloud/compute/container-layer-graph/", nil)
	copyProxyHeaders(src, dst)
	if got := dst.Header.Get("X-TaskContainerGateway-Internal-Secret"); got != "tcg-secret" {
		t.Fatalf("X-TaskContainerGateway-Internal-Secret=%q", got)
	}
}

func TestProxyContainerGatewayRequest_PropagatesTraceToUpstream(t *testing.T) {
	tracelog.Init("task-cloud-service-test")
	var gotTrace, gotParent, gotTraceparent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		gotParent = r.Header.Get("X-Parent-Span-Id")
		gotTraceparent = r.Header.Get("traceparent")
		if tracelog.IsTraceIdOnlyRequest(r) {
			http.Error(w, `{"detail":"trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"files":[]}`)
	}))
	defer upstream.Close()

	prevURL := cfg.ContainerGatewayURL
	cfg.ContainerGatewayURL = upstream.URL
	defer func() { cfg.ContainerGatewayURL = prevURL }()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562496/workspace/861623708318031872/task/task_12675381068363715869/cloud/compute/container-layer-files/?layer_id=20260709_155237_0e547f&max_files=3000&path=",
		nil,
	)
	req.Header.Set("X-Trace-Id", "web-1783583558132-0qwxrrap6d09")
	req.Header.Set("X-Parent-Span-Id", "895219d6c806919a")
	req.Header.Set("traceparent", "00-web17835835581320qwxrrap6d090000-895219d6c806919a-01")
	req.Header.Set("Authorization", "Token a8266ccd793329242ddeb0f269211c62924c383c")
	ctx := tracelog.ContextWithCorrelation(req.Context(), tracelog.Correlation{
		TraceID:      "web-1783583558132-0qwxrrap6d09",
		SpanID:       "2376ffeb427b6948",
		ParentSpanID: "895219d6c806919a",
	})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	proxyContainerGatewayRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotTrace != "web-1783583558132-0qwxrrap6d09" {
		t.Fatalf("upstream X-Trace-Id = %q", gotTrace)
	}
	if gotParent != "2376ffeb427b6948" {
		t.Fatalf("upstream X-Parent-Span-Id = %q want cloud-service span", gotParent)
	}
	if gotTraceparent == "" {
		t.Fatal("upstream traceparent missing")
	}
}

func TestIsContainerOutboundComputeSub_IncludesGitPush(t *testing.T) {
	cases := []struct {
		sub  string
		want bool
	}{
		{"compute/container-layer-git-push", true},
		{"compute/container-layer-git-push-auth-context", true},
		{"compute/container-layer-git-commit", true},
		{"compute/container-job-edit-run", true},
		{"compute/container-job-edit-run/extra", true},
		{"compute/container-auto-run-steps", true},
		{"compute/container-auto-run-steps/", true},
		{"compute/container-task-lifecycle-shutdown", true},
		{"compute/container-task-lifecycle-closing-soon", true},
		{"compute/start-vm", false},
	}
	for _, tc := range cases {
		if got := isContainerOutboundComputeSub(tc.sub); got != tc.want {
			t.Fatalf("sub=%q got=%v want=%v", tc.sub, got, tc.want)
		}
	}
}
