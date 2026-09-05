package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout reroutes os.Stdout for the duration of fn so JSON log lines
// emitted by tracelog can be asserted on.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func parseLogLine(t *testing.T, out string) map[string]string {
	t.Helper()
	var line map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &line); err != nil {
		t.Fatalf("expected a JSON log line, got %q: %v", out, err)
	}
	return line
}

func TestLogGitLabAPIFailureCarriesTraceID(t *testing.T) {
	body := []byte(`{"message":"404 Not Found"}`)
	out := captureStdout(t, func() {
		logGitLabAPIFailure("list-branches", "https://gitlab.daydaymoney.com/team/repo.git", 400, body,
			map[string]string{"X-Trace-Id": "16911ac8-51fc-4b1f-ba60-e3e042c518f0"})
	})
	line := parseLogLine(t, out)
	if line["trace_id"] != "16911ac8-51fc-4b1f-ba60-e3e042c518f0" {
		t.Fatalf("trace_id = %q, want %q (line=%s)", line["trace_id"], "16911ac8-51fc-4b1f-ba60-e3e042c518f0", out)
	}
	if line["op"] != "list-branches" || line["status"] != "400" {
		t.Fatalf("unexpected structured fields: %+v", line)
	}
	if line["msg"] != "404 Not Found" {
		t.Fatalf("msg field = %q, want %q", line["msg"], "404 Not Found")
	}
}

func TestLogGitLabAPIFailureFallsBackToTraceparent(t *testing.T) {
	body := []byte(`{"error":"rate_limit"}`)
	out := captureStdout(t, func() {
		logGitLabAPIFailure("probe-repo", "https://gitlab.daydaymoney.com/a.git", 403, body,
			map[string]string{"traceparent": "00-16911ac851fc4b1fba60e3e042c518f0-0000000000000001-01"})
	})
	line := parseLogLine(t, out)
	if line["trace_id"] != "16911ac851fc4b1fba60e3e042c518f0" {
		t.Fatalf("trace_id = %q, want traceparent-derived id (line=%s)", line["trace_id"], out)
	}
}

func TestLogGitLabAPIFailureWithoutTraceOmitsTraceID(t *testing.T) {
	out := captureStdout(t, func() {
		logGitLabAPIFailure("resolve-commit", "https://gitlab.daydaymoney.com/a.git", 500, []byte("boom"))
	})
	line := parseLogLine(t, out)
	if _, ok := line["trace_id"]; ok {
		t.Fatalf("trace_id present without any trace header: %+v", line)
	}
	if line["msg"] != "boom" {
		t.Fatalf("msg field = %q, want %q", line["msg"], "boom")
	}
}

func TestGitlabProjectResponseAllowsPush(t *testing.T) {
	if gitlabProjectResponseAllowsPush([]byte(`{"permissions":{"project_access":{"access_level":30}}}`)) != true {
		t.Fatal("developer 30")
	}
	if gitlabProjectResponseAllowsPush([]byte(`{"permissions":{"group_access":{"access_level":40},"project_access":null}}`)) != true {
		t.Fatal("group maintainer")
	}
	if gitlabProjectResponseAllowsPush([]byte(`{"permissions":{"project_access":{"access_level":20},"group_access":null}}`)) != false {
		t.Fatal("reporter must not allow push")
	}
	if gitlabProjectResponseAllowsPush([]byte(`{"id":1}`)) != true {
		t.Fatal("missing permissions is unknown")
	}
}
