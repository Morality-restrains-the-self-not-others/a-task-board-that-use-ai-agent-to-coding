package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchAIPublicRuntimeUserdataBodyRetries502Then200(t *testing.T) {
	prevBase, prevClient, prevRetries, prevDelay := snapshotAIProviderUserdataTestState()
	defer restoreAIProviderUserdataTestState(prevBase, prevClient, prevRetries, prevDelay)
	aiProviderUserdataMaxRetries = 2
	aiProviderUserdataRetryDelay = 10 * time.Millisecond

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"detail":"upstream error"}`))
			return
		}
		_, _ = w.Write([]byte(`{"content":"echo hello"}`))
	}))
	defer srv.Close()
	cfg.AIProviderBaseURL = srv.URL
	aiProviderHTTP = srv.Client()

	content, err := fetchAIPublicRuntimeUserdataBody("img1", "", "aliyun", "cn-shanghai")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if content != "echo hello" {
		t.Fatalf("content=%q want %q", content, "echo hello")
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d want 3 (initial + 2 retries)", attempts)
	}
}

func TestFetchAIPublicRuntimeUserdataBodyExhaustsRetries(t *testing.T) {
	prevBase, prevClient, prevRetries, prevDelay := snapshotAIProviderUserdataTestState()
	defer restoreAIProviderUserdataTestState(prevBase, prevClient, prevRetries, prevDelay)
	aiProviderUserdataMaxRetries = 2
	aiProviderUserdataRetryDelay = 10 * time.Millisecond

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"detail":"busy"}`))
	}))
	defer srv.Close()
	cfg.AIProviderBaseURL = srv.URL
	aiProviderHTTP = srv.Client()

	_, err := fetchAIPublicRuntimeUserdataBody("img1", "", "aliyun", "cn-shanghai")
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err=%q should surface last upstream detail", err.Error())
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d want 3 (initial + 2 retries)", attempts)
	}
}

func TestFetchAIPublicRuntimeUserdataBodyDoesNotRetry400(t *testing.T) {
	prevBase, prevClient, prevRetries, prevDelay := snapshotAIProviderUserdataTestState()
	defer restoreAIProviderUserdataTestState(prevBase, prevClient, prevRetries, prevDelay)
	aiProviderUserdataMaxRetries = 2
	aiProviderUserdataRetryDelay = 10 * time.Millisecond

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"bad request"}`))
	}))
	defer srv.Close()
	cfg.AIProviderBaseURL = srv.URL
	aiProviderHTTP = srv.Client()

	_, err := fetchAIPublicRuntimeUserdataBody("img1", "", "aliyun", "cn-shanghai")
	if err == nil {
		t.Fatal("expected 400 error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("err=%q should surface upstream detail", err.Error())
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d want 1 (400 must not retry)", attempts)
	}
}

func TestIsRetryableImageMarketError(t *testing.T) {
	if !isRetryableImageMarketError(http.StatusBadGateway, nil) {
		t.Fatal("502 should be retryable")
	}
	if !isRetryableImageMarketError(http.StatusServiceUnavailable, nil) {
		t.Fatal("503 should be retryable")
	}
	if !isRetryableImageMarketError(http.StatusGatewayTimeout, nil) {
		t.Fatal("504 should be retryable")
	}
	if isRetryableImageMarketError(http.StatusBadRequest, nil) {
		t.Fatal("400 must not be retryable")
	}
	if isRetryableImageMarketError(http.StatusNotFound, nil) {
		t.Fatal("404 must not be retryable")
	}
	var timeoutErr netErr = timeoutError{}
	if !isRetryableImageMarketError(0, timeoutErr) {
		t.Fatal("timeout error should be retryable")
	}
	if isRetryableImageMarketError(0, errors.New("connection refused")) {
		t.Fatal("connection refused must not be retryable")
	}
	if isRetryableImageMarketError(0, errors.New("no such host")) {
		t.Fatal("no such host must not be retryable")
	}
	if !isRetryableImageMarketError(0, errors.New("read tcp: connection reset by peer")) {
		t.Fatal("connection reset should be retryable")
	}
	if !isRetryableImageMarketError(0, io.EOF) {
		t.Fatal("EOF should be retryable")
	}
}

func snapshotAIProviderUserdataTestState() (string, *http.Client, int, time.Duration) {
	return cfg.AIProviderBaseURL, aiProviderHTTP, aiProviderUserdataMaxRetries, aiProviderUserdataRetryDelay
}

func restoreAIProviderUserdataTestState(base string, client *http.Client, retries int, delay time.Duration) {
	cfg.AIProviderBaseURL = base
	aiProviderHTTP = client
	aiProviderUserdataMaxRetries = retries
	aiProviderUserdataRetryDelay = delay
}

// netErr 包装 net.Error 接口供 isRetryableImageMarketError 测试。
type netErr interface {
	error
	netError
}

type netError interface {
	Timeout() bool
	Temporary() bool
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "net/http: request canceled (Client.Timeout exceeded)" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
