package cloud_csc_reconcile

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSweepOnceCallsOrphanThenLeaked(t *testing.T) {
	var mu sync.Mutex
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method=%s want POST", r.Method)
		}
		if got := r.Header.Get("X-Internal-Secret"); got != "test-secret" {
			t.Errorf("X-Internal-Secret=%q want test-secret", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "{}" {
			t.Errorf("body=%q want {}", string(body))
		}
		mu.Lock()
		hits = append(hits, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","cleaned":0}`))
	}))
	defer srv.Close()

	client := &TaskCloudSweepClient{
		BaseURL:        srv.URL,
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := SweepOnce(context.Background(), client); err != nil {
		t.Fatalf("SweepOnce: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(hits) != 4 {
		t.Fatalf("hits=%v want [orphan leaked runtime stale-starting]", hits)
	}
	// 先标 orphan，再清 leaked —— 顺序是流水线契约。
	if hits[0] != orphanPath {
		t.Errorf("first hit=%q want %q (orphan 必须先打标)", hits[0], orphanPath)
	}
	if hits[1] != leakedPath {
		t.Errorf("second hit=%q want %q", hits[1], leakedPath)
	}
	if hits[2] != runtimePath {
		t.Errorf("third hit=%q want %q", hits[2], runtimePath)
	}
	if hits[3] != staleStartingPath {
		t.Errorf("fourth hit=%q want %q", hits[3], staleStartingPath)
	}
}

func TestSweepOncePropagatesOrphanError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error"}`))
	}))
	defer srv.Close()

	client := &TaskCloudSweepClient{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}
	if err := SweepOnce(context.Background(), client); err == nil {
		t.Fatal("expected error when orphan endpoint fails")
	}
}

func TestSweepIntervalDefaults(t *testing.T) {
	t.Setenv("CLOUD_CSC_RECONCILE_TICK_SEC", "")
	if got := SweepInterval(); got != 300*time.Second {
		t.Fatalf("empty env: got %v want 300s", got)
	}
	t.Setenv("CLOUD_CSC_RECONCILE_TICK_SEC", "0")
	if got := SweepInterval(); got != 300*time.Second {
		t.Fatalf("zero env: got %v want 300s", got)
	}
	t.Setenv("CLOUD_CSC_RECONCILE_TICK_SEC", "-5")
	if got := SweepInterval(); got != 300*time.Second {
		t.Fatalf("negative env: got %v want 300s", got)
	}
	t.Setenv("CLOUD_CSC_RECONCILE_TICK_SEC", "30")
	if got := SweepInterval(); got != 30*time.Second {
		t.Fatalf("30 env: got %v want 30s", got)
	}
}

func TestStartSweepTickerInvokesAtLeastOnce(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","cleaned":0}`))
	}))
	defer srv.Close()

	client := &TaskCloudSweepClient{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		StartSweepTicker(ctx, client, 40*time.Millisecond)
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
		t.Fatal("StartSweepTicker did not return after cancel")
	}
}
