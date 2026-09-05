package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func clearUserdataEnvAgentCfg(t *testing.T) {
	t.Helper()
	prevKey := cfg.UserdataEnvAgentAPIKey
	prevBase := cfg.UserdataEnvAgentBaseURL
	prevModel := cfg.UserdataEnvAgentModel
	prevMax := cfg.UserdataEnvAgentMaxTokens
	prevTemp := cfg.UserdataEnvAgentTemperature
	cfg.UserdataEnvAgentAPIKey = ""
	cfg.UserdataEnvAgentBaseURL = ""
	cfg.UserdataEnvAgentModel = ""
	cfg.UserdataEnvAgentMaxTokens = 1200
	cfg.UserdataEnvAgentTemperature = 0
	t.Cleanup(func() {
		cfg.UserdataEnvAgentAPIKey = prevKey
		cfg.UserdataEnvAgentBaseURL = prevBase
		cfg.UserdataEnvAgentModel = prevModel
		cfg.UserdataEnvAgentMaxTokens = prevMax
		cfg.UserdataEnvAgentTemperature = prevTemp
	})
}

func TestExtractContainerEnvByAgent_NoConfigHeuristic(t *testing.T) {
	clearUserdataEnvAgentCfg(t)
	ud := "export FOO=bar\ndocker run -e HELLO=world img\n"
	env, parser := extractContainerEnvByAgent(ud)
	if parser != "heuristic" {
		t.Fatalf("parser=%q", parser)
	}
	if env["FOO"] != "bar" || env["HELLO"] != "world" {
		t.Fatalf("env=%v", env)
	}
}

func TestExtractContainerEnvByAgent_LLMSuccess(t *testing.T) {
	clearUserdataEnvAgentCfg(t)
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing bearer auth")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"content": "```json\n{\"AGENT_KEY\":\"from-llm\",\"DEBUG\":\"1\"}\n```",
				}},
			},
		})
	}))
	defer llm.Close()
	cfg.UserdataEnvAgentAPIKey = "sk-test"
	cfg.UserdataEnvAgentBaseURL = llm.URL
	cfg.UserdataEnvAgentModel = "test-model"

	env, parser := extractContainerEnvByAgent("export IGNORED=1")
	if parser != "agent" {
		t.Fatalf("parser=%q", parser)
	}
	if env["AGENT_KEY"] != "from-llm" || env["DEBUG"] != "1" {
		t.Fatalf("env=%v", env)
	}
}

func TestExtractContainerEnvByAgent_LLM500Fallback(t *testing.T) {
	clearUserdataEnvAgentCfg(t)
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer llm.Close()
	cfg.UserdataEnvAgentAPIKey = "sk-test"
	cfg.UserdataEnvAgentBaseURL = llm.URL
	cfg.UserdataEnvAgentModel = "test-model"

	ud := "export FALLBACK_OK=yes\n"
	env, parser := extractContainerEnvByAgent(ud)
	if parser != "heuristic" {
		t.Fatalf("parser=%q", parser)
	}
	if env["FALLBACK_OK"] != "yes" {
		t.Fatalf("env=%v", env)
	}
}

func TestNormalizeEnvObjAndStripFence(t *testing.T) {
	got := normalizeEnvObj(map[string]any{"OK": "1", "bad-key": "x", "": "y", "NIL": nil})
	if got["OK"] != "1" || len(got) != 1 {
		t.Fatalf("got=%v", got)
	}
	if s := stripMarkdownJSONFence("```json\n{\"A\":1}\n```"); s != `{"A":1}` {
		t.Fatalf("strip=%q", s)
	}
}
