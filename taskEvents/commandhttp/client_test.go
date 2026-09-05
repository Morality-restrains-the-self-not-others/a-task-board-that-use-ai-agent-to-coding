package commandhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskEvents/domain"
	"tracelog"
)

func TestDispatchOutcomes(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sec")
	cmd := domain.DomainCommand{
		EventType: "USER_CREATED",
		Path:      "/api/internal/task-events/accounts/user-created/",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: []byte(`{"user_id":1}`)},
	}
	out, err := c.Dispatch(context.Background(), cmd)
	if out != domain.DispatchRetryable || err == nil {
		t.Fatalf("first call: out=%v err=%v", out, err)
	}
	out, err = c.Dispatch(context.Background(), cmd)
	if out != domain.DispatchSuccess || err != nil {
		t.Fatalf("second call: out=%v err=%v", out, err)
	}
}

// TestDispatchAppliesOutboundHeaders 断言出站请求带 X-Trace-Id + X-Parent-Span-Id + traceparent，
// 即使 ctx 仅 background()（未走 Middleware）也会补齐 span 关联（OPT-20260817-022）。
func TestDispatchAppliesOutboundHeaders(t *testing.T) {
	var gotHeader http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sec")
	cmd := domain.DomainCommand{
		EventType: "USER_CREATED",
		Path:      "/api/internal/task-events/accounts/user-created/",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: []byte(`{"user_id":1}`)},
	}
	out, err := c.Dispatch(context.Background(), cmd)
	if out != domain.DispatchSuccess || err != nil {
		t.Fatalf("dispatch: out=%v err=%v", out, err)
	}
	if tid := gotHeader.Get(tracelog.Header); tid == "" {
		t.Fatalf("missing %s header", tracelog.Header)
	}
	if parent := gotHeader.Get(tracelog.ParentSpanHeader); parent == "" {
		t.Fatalf("missing %s header — downstream RejectTraceIdOnlyHTTP 会 400", tracelog.ParentSpanHeader)
	}
	if tp := gotHeader.Get(tracelog.TraceParentHeader); tp == "" {
		t.Fatalf("missing %s header", tracelog.TraceParentHeader)
	}
}
