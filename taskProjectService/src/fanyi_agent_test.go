package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"tracelog"
)

// TestFanyiAgentConfCleanedUp asserts the tracked conf matches what the Go code
// actually reads (OPT-20260901-027): max_tokens ≤ title cap 128, and the
// unused max_retries / parallel_tool_calls / top_k keys are gone.
func TestFanyiAgentConfCleanedUp(t *testing.T) {
	var root string
	for dir := "."; ; {
		abs, err := filepath.Abs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(abs, "conf", "base.yaml")); err == nil {
			root = abs
			break
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			t.Fatal("conf/base.yaml not found walking up from test file")
		}
		dir = filepath.Join(dir, "..")
	}
	raw, err := os.ReadFile(filepath.Join(root, "conf", "taskProjectService", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		AIAgentConfig struct {
			FanyiAgent map[string]any `yaml:"fanyi_agent"`
		} `yaml:"ai_agent_config"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	f := cfg.AIAgentConfig.FanyiAgent
	if f == nil {
		t.Fatal("ai_agent_config.fanyi_agent missing in conf/taskProjectService/config.yaml")
	}
	maxTokens, _ := f["max_tokens"].(int)
	if maxTokens > fanyiTitleMaxTokens {
		t.Fatalf("conf max_tokens=%d exceeds title cap %d", maxTokens, fanyiTitleMaxTokens)
	}
	for _, gone := range []string{"max_retries", "parallel_tool_calls", "top_k"} {
		if _, ok := f[gone]; ok {
			t.Fatalf("fanyi_agent.%s must be removed from tracked conf (unused)", gone)
		}
	}
}

func TestFanyiTitleTimeoutCoversObservedDeepseekLatency(t *testing.T) {
	if fanyiTitleTimeout < 12*time.Second {
		t.Fatalf("fanyiTitleTimeout=%s too low; live successes took 11.5s", fanyiTitleTimeout)
	}
	if fanyiTitleTimeout > 20*time.Second {
		t.Fatalf("fanyiTitleTimeout=%s too high for create-task UX", fanyiTitleTimeout)
	}
}

func TestTranslateTitleWithFanyiAgentTruncatedJSONIsNotUnmarshalDump(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"`))
	}))
	t.Cleanup(srv.Close)

	oldCfg := fanyiAgentCfg
	oldClient := fanyiHTTPClient
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	fanyiHTTPClient = tracelog.DirectClient(2 * time.Second)
	t.Cleanup(func() {
		fanyiAgentCfg = oldCfg
		fanyiHTTPClient = oldClient
	})

	_, err := translateTitleWithFanyiAgent(context.Background(), "修复登录")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "unexpected end of JSON input") {
		t.Fatalf("truncated body should not surface json.Unmarshal: %s", msg)
	}
	if !strings.Contains(msg, "响应无效") {
		t.Fatalf("want 响应无效, got %s", msg)
	}
}

func TestTranslateTitleWithFanyiAgentEmptyBodyIsNotJSONParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	oldCfg := fanyiAgentCfg
	oldClient := fanyiHTTPClient
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	fanyiHTTPClient = tracelog.DirectClient(2 * time.Second)
	t.Cleanup(func() {
		fanyiAgentCfg = oldCfg
		fanyiHTTPClient = oldClient
	})

	_, err := translateTitleWithFanyiAgent(context.Background(), "修复登录")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "unexpected end of JSON input") {
		t.Fatalf("empty body should not surface json.Unmarshal: %s", msg)
	}
	if !strings.Contains(msg, "响应为空") {
		t.Fatalf("want 响应为空, got %s", msg)
	}
}

func TestTranslateTitleWithFanyiAgentTimeoutIsNotJSONParseError(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"too late"}}]}`))
	}))
	t.Cleanup(srv.Close)

	oldCfg := fanyiAgentCfg
	oldClient := fanyiHTTPClient
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	fanyiHTTPClient = tracelog.DirectClient(150 * time.Millisecond)
	t.Cleanup(func() {
		fanyiAgentCfg = oldCfg
		fanyiHTTPClient = oldClient
	})

	_, err := translateTitleWithFanyiAgent(context.Background(), "修复登录")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	<-started
	msg := err.Error()
	if strings.Contains(msg, "unexpected end of JSON input") {
		t.Fatalf("timeout should not surface json.Unmarshal: %s", msg)
	}
	if !strings.Contains(msg, "超时") && !strings.Contains(msg, "请求失败") {
		t.Fatalf("want timeout classification, got %s", msg)
	}
}

func TestTranslateTitleWithFanyiAgentCapsMaxTokens(t *testing.T) {
	var gotMax int
	var thinkingType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Errorf("payload json: %v", err)
		}
		if n, ok := payload["max_tokens"].(float64); ok {
			gotMax = int(n)
		}
		stream, _ := payload["stream"].(bool)
		if stream {
			t.Error("stream must be false for title translation")
		}
		if th, ok := payload["thinking"].(map[string]interface{}); ok {
			thinkingType, _ = th["type"].(string)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Fix Login"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	oldCfg := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{
		APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 4096, Temperature: 0.1, TopP: 1,
	}
	t.Cleanup(func() { fanyiAgentCfg = oldCfg })

	got, err := translateTitleWithFanyiAgent(context.Background(), "修复登录")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if got == "" {
		t.Fatal("empty translation")
	}
	const wantCap = 128
	if gotMax != wantCap {
		t.Fatalf("max_tokens=%d want cap %d", gotMax, wantCap)
	}
	if thinkingType != "disabled" {
		t.Fatalf("thinking.type=%q want disabled (deepseek-v4 thinking burns short max_tokens)", thinkingType)
	}
}

func TestTranslateTitleWithFanyiAgentEmptyContentClassifiesLengthBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"finish_reason": "length",
					"message": map[string]string{
						"content":           "",
						"reasoning_content": strings.Repeat("think ", 40),
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	oldCfg := fanyiAgentCfg
	fanyiAgentCfg = FanyiAgentConfig{APIKey: "k", BaseURL: srv.URL, Model: "m", MaxTokens: 64}
	t.Cleanup(func() { fanyiAgentCfg = oldCfg })

	_, err := translateTitleWithFanyiAgent(context.Background(), "修复登录")
	if err == nil {
		t.Fatal("expected empty-content error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "返回内容为空") {
		t.Fatalf("want 返回内容为空, got %s", msg)
	}
	if !strings.Contains(msg, "finish_reason=length") {
		t.Fatalf("want finish_reason=length in log-facing err, got %s", msg)
	}
}
