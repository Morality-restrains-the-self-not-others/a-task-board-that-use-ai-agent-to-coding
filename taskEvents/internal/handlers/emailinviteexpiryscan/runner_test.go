package emailinviteexpiryscan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewTaskAuthExpireClientDefaultBaseURLIs8003(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "")
	t.Setenv("SHARED_INTERNAL_SECRET", "")
	c := NewTaskAuthExpireClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("default BaseURL = %q, want http://127.0.0.1:8003", c.BaseURL)
	}
}

func TestNewTaskAuthExpireClientUsesEnvAndTrimsSlash(t *testing.T) {
	t.Setenv("TASK_AUTH_INTERNAL_URL", "http://127.0.0.1:8003/")
	t.Setenv("TASK_AUTH_INTERNAL_SECRET", "sekret")
	c := NewTaskAuthExpireClient()
	if c.BaseURL != "http://127.0.0.1:8003" {
		t.Fatalf("BaseURL = %q, want trimmed http://127.0.0.1:8003", c.BaseURL)
	}
	if c.InternalSecret != "sekret" {
		t.Fatalf("InternalSecret = %q, want sekret", c.InternalSecret)
	}
}

func TestTaskAuthExpireClientExpireOnceHitsExpireDuePath(t *testing.T) {
	var gotPath, gotSecret, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotSecret = r.Header.Get("X-TaskAuth-Internal-Secret")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"expired":3}`))
	}))
	defer srv.Close()

	c := &TaskAuthExpireClient{
		BaseURL:        strings.TrimRight(srv.URL, "/"),
		InternalSecret: "test-secret",
		HTTPClient:     srv.Client(),
	}
	if err := c.ExpireOnce(context.Background()); err != nil {
		t.Fatalf("ExpireOnce() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotPath != expireDuePath {
		t.Fatalf("request path = %q, want %q", gotPath, expireDuePath)
	}
	if gotSecret != "test-secret" {
		t.Fatalf("X-TaskAuth-Internal-Secret = %q, want test-secret", gotSecret)
	}
}

func TestTaskAuthExpireClientExpireOnceReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`oops`))
	}))
	defer srv.Close()

	c := &TaskAuthExpireClient{
		BaseURL:    strings.TrimRight(srv.URL, "/"),
		HTTPClient: srv.Client(),
	}
	if err := c.ExpireOnce(context.Background()); err == nil {
		t.Fatal("ExpireOnce() expected error on HTTP 500, got nil")
	}
}

func TestExpireTickIntervalDefaultAndFallback(t *testing.T) {
	t.Setenv("EMAIL_INVITE_EXPIRY_SCAN_TICK_SEC", "")
	if got := ExpireTickInterval(); got != time.Hour {
		t.Fatalf("default tick = %v, want 1h", got)
	}
	t.Setenv("EMAIL_INVITE_EXPIRY_SCAN_TICK_SEC", "0")
	if got := ExpireTickInterval(); got != time.Hour {
		t.Fatalf("fallback tick = %v, want 1h", got)
	}
	t.Setenv("EMAIL_INVITE_EXPIRY_SCAN_TICK_SEC", "120")
	if got := ExpireTickInterval(); got != 120*time.Second {
		t.Fatalf("configured tick = %v, want 120s", got)
	}
}

type countingExpireClient struct {
	ch chan struct{}
}

func (c *countingExpireClient) ExpireOnce(ctx context.Context) error {
	select {
	case c.ch <- struct{}{}:
	default:
	}
	return nil
}

func TestStartExpireTickerFiresImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &countingExpireClient{ch: make(chan struct{}, 1)}
	done := make(chan struct{})
	go func() {
		StartExpireTicker(ctx, client, time.Hour)
		close(done)
	}()
	select {
	case <-client.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("StartExpireTicker did not fire immediately")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartExpireTicker did not stop after cancel")
	}
}
