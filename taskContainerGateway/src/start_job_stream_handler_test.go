package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandleStartJobStream_OK(t *testing.T) {
	var publishHits atomic.Int32
	var upstreamHits atomic.Int32

	prevPublish := jobStreamPublishFunc
	jobStreamPublishFunc = func(ctx context.Context, sc scope, jobID, traceID string, fields map[string]any) {
		publishHits.Add(1)
	}
	t.Cleanup(func() { jobStreamPublishFunc = prevPublish })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		if r.Header.Get("X-Access-Token") != "tok" {
			t.Fatalf("missing or wrong access token: %q", r.Header.Get("X-Access-Token"))
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/events"):
			_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{}, "next_offset": 0})
		case strings.Contains(r.URL.Path, "/jobs/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "job-1", "status": "completed"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	cfg = serviceConfig{
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}
	jobStreamCfg.PollIntervalSec = 0.01
	jobStreamCfg.PollDeadlineSec = 2
	jobStreamCfg.PollConnectSec = 1
	jobStreamCfg.PollReadSec = 1

	mux := http.NewServeMux()
	mountRoutes(mux)

	payload := `{
		"tenant_id":"t1",
		"workspace_id":"w1",
		"task_id":"task1",
		"job_id":"job-1",
		"base_url":"` + upstream.URL + `",
		"access_token":"tok",
		"trace_id":"trace-edit-run"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/start-job-stream/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" || resp["job_id"] != "job-1" {
		t.Fatalf("resp=%v", resp)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if publishHits.Load() > 0 && upstreamHits.Load() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf(
		"expected job-stream poll+publish, publishHits=%d upstreamHits=%d",
		publishHits.Load(),
		upstreamHits.Load(),
	)
}

func TestHandleStartJobStream_ForbiddenWithoutSecret(t *testing.T) {
	cfg = serviceConfig{
		InternalSecret: "secret",
	}
	mux := http.NewServeMux()
	mountRoutes(mux)

	payload := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task1","job_id":"job-1","base_url":"http://x","access_token":"t"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/start-job-stream/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleStartJobStream_BadRequestMissingFields(t *testing.T) {
	cfg = serviceConfig{
		InternalSecret: "secret",
	}
	mux := http.NewServeMux()
	mountRoutes(mux)

	payload := `{"tenant_id":"t1","workspace_id":"w1","task_id":"","job_id":"job-1","base_url":"http://x","access_token":"t"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/start-job-stream/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleStartJobStream_BadRequestMissingTarget(t *testing.T) {
	cfg = serviceConfig{
		InternalSecret: "secret",
	}
	mux := http.NewServeMux()
	mountRoutes(mux)

	payload := `{"tenant_id":"t1","workspace_id":"w1","task_id":"task1","job_id":"job-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/start-job-stream/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "base_url") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
