package main

import (
	"encoding/json"
	"strings"
)

// subprocessLogGrouper merges multi-line child stderr (e.g. Node.js stack traces)
// into single log entries before forwarding to Loki/Grafana.
type subprocessLogGrouper struct {
	pending []string
}

func newSubprocessLogGrouper() *subprocessLogGrouper {
	return &subprocessLogGrouper{}
}

func (g *subprocessLogGrouper) Feed(line string) []string {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return g.flush()
	}
	if g.hasPending() && isSubprocessLogContinuation(line) {
		g.pending = append(g.pending, line)
		return nil
	}
	out := g.flush()
	g.pending = []string{line}
	return out
}

func (g *subprocessLogGrouper) Flush() []string {
	return g.flush()
}

func (g *subprocessLogGrouper) hasPending() bool {
	return len(g.pending) > 0
}

func (g *subprocessLogGrouper) flush() []string {
	if !g.hasPending() {
		return nil
	}
	block := strings.Join(g.pending, "\n")
	g.pending = nil
	return []string{block}
}

func isSubprocessLogContinuation(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if isStructuredServiceLogLine(trimmed) {
		return false
	}
	if trimmed == "}" || trimmed == "{" {
		return true
	}
	if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
		return true
	}
	return strings.HasPrefix(trimmed, "at ") || strings.HasPrefix(trimmed, "at async ")
}

func isStructuredServiceLogLine(line string) bool {
	if !strings.HasPrefix(line, "{") {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return false
	}
	_, ok := fields["service"]
	return ok
}
