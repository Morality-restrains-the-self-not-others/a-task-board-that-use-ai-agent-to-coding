package queued_auto_run_scan

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatchOnceCallsInternalAPI(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method=%s want POST", r.Method)
		}
		if r.URL.Path != dispatchOncePath {
			t.Errorf("path=%s want %s", r.URL.Path, dispatchOncePath)
		}
		if got := r.Header.Get("X-Internal-Secret"); got != "test-secret" {
			t.Errorf("X-Internal-Secret=%q want test-secret", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "{}" {
			t.Errorf("body=%q want {}", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	client := &TaskTaskServiceDispatchClient{
		BaseURL:        srv.URL,
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := DispatchOnce(context.Background(), client); err != nil {
		t.Fatalf("DispatchOnce: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d want 1", calls.Load())
	}
}

func TestDispatchOncePropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error"}`))
	}))
	defer srv.Close()

	client := &TaskTaskServiceDispatchClient{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}
	if err := DispatchOnce(context.Background(), client); err == nil {
		t.Fatal("expected error on non-2xx")
	}
}

func TestDispatchIntervalDefaults(t *testing.T) {
	t.Setenv("QUEUED_AUTO_RUN_SCAN_TICK_SEC", "")
	if got := DispatchInterval(); got != 30*time.Second {
		t.Fatalf("empty env: got %v want 30s", got)
	}
	t.Setenv("QUEUED_AUTO_RUN_SCAN_TICK_SEC", "0")
	if got := DispatchInterval(); got != 30*time.Second {
		t.Fatalf("zero env: got %v want 30s", got)
	}
	t.Setenv("QUEUED_AUTO_RUN_SCAN_TICK_SEC", "-5")
	if got := DispatchInterval(); got != 30*time.Second {
		t.Fatalf("negative env: got %v want 30s", got)
	}
	t.Setenv("QUEUED_AUTO_RUN_SCAN_TICK_SEC", "10")
	if got := DispatchInterval(); got != 10*time.Second {
		t.Fatalf("10 env: got %v want 10s", got)
	}
}

func TestStartDispatchTickerInvokesAtLeastOnce(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	client := &TaskTaskServiceDispatchClient{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		StartDispatchTicker(ctx, client, 40*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if calls.Load() < 1 {
		t.Fatalf("calls=%d want at least 1", calls.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartDispatchTicker did not return after cancel")
	}
}
