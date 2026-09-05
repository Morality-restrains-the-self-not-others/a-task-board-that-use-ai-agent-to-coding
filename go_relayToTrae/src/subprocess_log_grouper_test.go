package main

import (
	"strings"
	"testing"
)

func TestSubprocessLogGrouper_MergesNodeStackTrace(t *testing.T) {
	g := newSubprocessLogGrouper()
	lines := []string{
		"[onlineServiceJS] reachability 失败: Error: HTTP 500",
		"    at postJson (file:///app/src/saasTaskCloud.mjs:154:23)",
		"    at async registerReachabilityAfterBootstrap (file:///app/src/reachability.mjs:265:3)",
		"    at async Server.<anonymous> (file:///app/src/server.mjs:1769:9) {",
		"  code: 'ERR_HTTP'",
		"}",
		"[onlineServiceJS] server listening on http://0.0.0.0:8765",
	}

	var out []string
	for _, line := range lines {
		out = append(out, g.Feed(line)...)
	}
	out = append(out, g.Flush()...)

	if len(out) != 2 {
		t.Fatalf("expected 2 log blocks, got %d: %#v", len(out), out)
	}
	if !strings.Contains(out[0], "reachability 失败") {
		t.Fatalf("first block missing error head: %q", out[0])
	}
	if !strings.Contains(out[0], "at postJson") || !strings.Contains(out[0], "code: 'ERR_HTTP'") {
		t.Fatalf("first block missing stack trace: %q", out[0])
	}
	if out[1] != "[onlineServiceJS] server listening on http://0.0.0.0:8765" {
		t.Fatalf("second block = %q", out[1])
	}
}

func TestSubprocessLogGrouper_StructuredJSONNotMergedWithStack(t *testing.T) {
	g := newSubprocessLogGrouper()
	jsonLine := `{"ts":"2026-05-28T10:00:00Z","level":"info","service":"onlineServiceJS","msg":"http_request"}`

	var out []string
	out = append(out, g.Feed("[onlineServiceJS] reachability 失败: Error: HTTP 500")...)
	out = append(out, g.Feed("    at postJson (file:///app/src/saasTaskCloud.mjs:154:23)")...)
	out = append(out, g.Feed(jsonLine)...)
	out = append(out, g.Flush()...)

	if len(out) != 2 {
		t.Fatalf("expected 2 log blocks, got %d: %#v", len(out), out)
	}
	if !strings.Contains(out[0], "at postJson") {
		t.Fatalf("stack not merged into first block: %q", out[0])
	}
	if out[1] != jsonLine {
		t.Fatalf("json line merged incorrectly: %q", out[1])
	}
}

func TestSubprocessLogGrouper_EmptyLineFlushesPending(t *testing.T) {
	g := newSubprocessLogGrouper()
	_ = g.Feed("line one")
	out := g.Feed("")
	if len(out) != 1 || out[0] != "line one" {
		t.Fatalf("expected flush on blank line, got %#v", out)
	}
}
