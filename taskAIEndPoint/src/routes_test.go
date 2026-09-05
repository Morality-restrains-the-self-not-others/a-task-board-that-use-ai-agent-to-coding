package main

import "testing"

func TestParseLlmProxyPath(t *testing.T) {
	match, ok := parseLlmProxyPath("/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1/chat/completions")
	if !ok {
		t.Fatal("expected match")
	}
	if match.TenantID != "t1" || match.WorkspaceID != "w1" || match.TaskID != "k1" || match.Provider != "deepseek" {
		t.Fatalf("unexpected match: %+v", match)
	}
	if match.SubPath != "chat/completions" {
		t.Fatalf("subpath=%q", match.SubPath)
	}
}

func TestEnsureUpstreamV1BaseNoDoubleV1(t *testing.T) {
	got := ensureUpstreamV1Base("https://api.deepseek.com/v1")
	if got != "https://api.deepseek.com/v1" {
		t.Fatalf("got=%q", got)
	}
	got = ensureUpstreamV1Base("https://api.deepseek.com/v1/")
	if got != "https://api.deepseek.com/v1" {
		t.Fatalf("trailing got=%q", got)
	}
	got = ensureUpstreamV1Base("https://api.deepseek.com")
	if got != "https://api.deepseek.com/v1" {
		t.Fatalf("append got=%q", got)
	}
	url := fmtUpstreamURL(ensureUpstreamV1Base("https://api.deepseek.com/v1"), "chat/completions")
	if url != "https://api.deepseek.com/v1/chat/completions" {
		t.Fatalf("upstream=%q", url)
	}
}

func TestParseUsageFromJSON(t *testing.T) {
	raw := []byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20}}`)
	inTok, outTok := parseUsageFromJSON(raw)
	if inTok != 10 || outTok != 20 {
		t.Fatalf("tokens in=%d out=%d", inTok, outTok)
	}
}
