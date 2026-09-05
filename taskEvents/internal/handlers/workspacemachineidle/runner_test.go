package workspacemachineidle

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecycleOnceCallsInternalAPI(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method=%s want POST", r.Method)
		}
		if r.URL.Path != recyclePath {
			t.Errorf("path=%s want %s", r.URL.Path, recyclePath)
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
		_, _ = w.Write([]byte(`{"status":"success","recycled":0}`))
	}))
	defer srv.Close()

	client := &TaskCloudRecycleClient{
		BaseURL:        srv.URL,
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := RecycleOnce(context.Background(), client); err != nil {
		t.Fatalf("RecycleOnce: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d want 1", calls.Load())
	}
}

func TestRecycleTickIntervalDefaults(t *testing.T) {
	t.Setenv("IDLE_RECYCLE_TICK_SEC", "")
	if got := RecycleTickInterval(); got != 60*time.Second {
		t.Fatalf("empty env: got %v want 60s", got)
	}
	t.Setenv("IDLE_RECYCLE_TICK_SEC", "0")
	if got := RecycleTickInterval(); got != 60*time.Second {
		t.Fatalf("zero env: got %v want 60s", got)
	}
	t.Setenv("IDLE_RECYCLE_TICK_SEC", "-5")
	if got := RecycleTickInterval(); got != 60*time.Second {
		t.Fatalf("negative env: got %v want 60s", got)
	}
	t.Setenv("IDLE_RECYCLE_TICK_SEC", "30")
	if got := RecycleTickInterval(); got != 30*time.Second {
		t.Fatalf("30 env: got %v want 30s", got)
	}
}

func TestStartRecycleTickerInvokesAtLeastOnce(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","recycled":0}`))
	}))
	defer srv.Close()

	client := &TaskCloudRecycleClient{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		StartRecycleTicker(ctx, client, 40*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not stop after cancel")
	}
	if calls.Load() < 1 {
		t.Fatalf("calls=%d want at least 1", calls.Load())
	}
}
