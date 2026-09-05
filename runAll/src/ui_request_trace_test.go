package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTraceTestRunner(t *testing.T) (*StatusStore, *Runner) {
	t.Helper()
	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "saas-backend", Command: "echo running"},
			}},
		},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	return store, runner
}

func lastRunallStructured(t *testing.T, runner *Runner) map[string]interface{} {
	t.Helper()
	entries := runner.logRepository.Tail("runall", 5)
	if len(entries) == 0 {
		t.Fatalf("expected a structured runall log entry")
	}
	last := entries[len(entries)-1].Message
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(last), &payload); err != nil {
		t.Fatalf("structured log not valid JSON: %v: %s", err, last)
	}
	return payload
}

func TestUIRequestTraceLogging_ErrorWritesTraceID(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONErrorWithStatus(w, http.StatusConflict, "已有全部重新编译进行中")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/build-all", nil)
	req.Header.Set("X-Trace-Id", "runall-1787172683744-jw88w2lfbj")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	payload := lastRunallStructured(t, runner)
	if payload["trace_id"] != "runall-1787172683744-jw88w2lfbj" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
	if payload["status"] != float64(http.StatusConflict) {
		t.Fatalf("status = %v", payload["status"])
	}
	if payload["path"] != "/api/build-all" {
		t.Fatalf("path = %v", payload["path"])
	}
	if payload["method"] != http.MethodPost {
		t.Fatalf("method = %v", payload["method"])
	}
	if payload["level"] != "error" {
		t.Fatalf("level = %v", payload["level"])
	}
}

func TestUIRequestTraceLogging_MutatingSuccessLogged(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/restart", nil)
	req.Header.Set("X-Trace-Id", "runall-restart-trace")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	payload := lastRunallStructured(t, runner)
	if payload["trace_id"] != "runall-restart-trace" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
	if payload["level"] != "info" {
		t.Fatalf("level = %v", payload["level"])
	}
}

func TestUIRequestTraceLogging_SkipGETSuccess(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/progress", nil)
	req.Header.Set("X-Trace-Id", "runall-poll-noise")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if entries := runner.logRepository.Tail("runall", 5); len(entries) != 0 {
		t.Fatalf("expected no runall log entry for GET success, got %d", len(entries))
	}
}

func TestUIRequestTraceLogging_SkipNoTraceID(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONErrorWithStatus(w, http.StatusConflict, "no trace id")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/build-all", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if entries := runner.logRepository.Tail("runall", 5); len(entries) != 0 {
		t.Fatalf("expected no runall log entry without X-Trace-Id, got %d", len(entries))
	}
}

func TestUIRequestTraceLogging_ErrorGETLogged(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, "boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/dev/logs", nil)
	req.Header.Set("X-Trace-Id", "runall-get-error")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	payload := lastRunallStructured(t, runner)
	if payload["trace_id"] != "runall-get-error" {
		t.Fatalf("trace_id = %v", payload["trace_id"])
	}
}

func TestUIRequestTraceLogging_PreservesFlusher(t *testing.T) {
	_, runner := newTraceTestRunner(t)

	handler := uiRequestTraceLogging(runner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Errorf("response writer no longer implements http.Flusher (SSE)")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/build-all/progress", nil)
	req.Header.Set("X-Trace-Id", "runall-sse")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestUIRequestTraceLogging_NilRunnerPassThrough(t *testing.T) {
	handler := uiRequestTraceLogging(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/build-all", nil)
	req.Header.Set("X-Trace-Id", "runall-nil-runner")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
