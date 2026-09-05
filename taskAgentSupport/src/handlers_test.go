package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCloudInboundPathRejectsWithoutComment(t *testing.T) {
	paths := []string{
		"/api/tenant/827923618468040704/workspace/827923618602258432/task/847744505890045952/cloud/relay-to-trae/status-push/",
		"/api/tenant/t1/workspace/w1/task/task1/cloud/server-container-token/register-reachability/",
		"/api/tenant/t1/workspace/w1/task/task1/cloud/model-budget-usage/",
		"/api/tenant/t1/workspace/w1/task/task1/cloud/server-container-token/feature-params-env/",
		"/api/tenant/t1/workspace/w1/task/task1/comment/-/cloud/server-container-token/heartbeat/",
	}
	for _, path := range paths {
		if _, ok := parseCloudInboundPath(path); ok {
			t.Fatalf("expected reject for %q", path)
		}
	}
}

func TestParseCloudInboundPathRequiresComment(t *testing.T) {
	cases := []struct {
		path   string
		action string
		cid    string
	}{
		{
			path:   "/api/tenant/827923618468040704/workspace/827923618602258432/task/847744505890045952/comment/cmt_9/cloud/relay-to-trae/status-push/",
			action: "relay-status-push",
			cid:    "cmt_9",
		},
		{
			path:   "/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/server-container-token/register-reachability/",
			action: "register-reachability",
			cid:    "cmt_1",
		},
		{
			path:   "/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/model-budget-usage/",
			action: "model-budget-usage",
			cid:    "cmt_1",
		},
		{
			path:   "/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/server-container-token/feature-params-env/",
			action: "feature-params-env",
			cid:    "cmt_1",
		},
	}
	for _, c := range cases {
		match, ok := parseCloudInboundPath(c.path)
		if !ok {
			t.Fatalf("expected path to match: %s", c.path)
		}
		if match.Action != c.action {
			t.Fatalf("action=%q want %s", match.Action, c.action)
		}
		if match.CommentID != c.cid {
			t.Fatalf("comment=%q want %s", match.CommentID, c.cid)
		}
	}
}

func TestHandleCloudInboundRejectsLegacyPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/workspace/w1/task/task1/cloud/server-container-token/heartbeat/",
		strings.NewReader(`{"access_token":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleCloudInbound(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%s", rec.Code, rec.Body.String())
	}
}

func TestCloudServiceInboundPathRequiresComment(t *testing.T) {
	prev := cfg.TaskCloudServiceURL
	cfg.TaskCloudServiceURL = "http://cloud:8018"
	defer func() { cfg.TaskCloudServiceURL = prev }()

	cases := []struct {
		action string
		want   string
	}{
		{"feature-params-env", "http://cloud:8018/api/cloud/server-container-token/feature-params-env/tenant_id/t/workspace_id/w/task_id/x/comment_id/cmt_1"},
		{"relay-status-push", "http://cloud:8018/api/cloud/relay-to-trae/status-push/tenant_id/t/workspace_id/w/task_id/x/comment_id/cmt_1"},
		{"model-budget-usage", "http://cloud:8018/api/cloud/model-budget-usage/tenant_id/t/workspace_id/w/task_id/x/comment_id/cmt_1"},
		{"heartbeat", "http://cloud:8018/api/cloud/server-container-token/heartbeat/tenant_id/t/workspace_id/w/task_id/x/comment_id/cmt_1"},
	}
	for _, c := range cases {
		got := cloudServiceInboundPath(c.action, "t", "w", "x", "cmt_1")
		if got != c.want {
			t.Fatalf("action=%s path=%q want %q", c.action, got, c.want)
		}
	}
}

func TestCloudServiceInboundPathRejectsEmptyComment(t *testing.T) {
	prev := cfg.TaskCloudServiceURL
	cfg.TaskCloudServiceURL = "http://cloud:8018"
	defer func() { cfg.TaskCloudServiceURL = prev }()
	if got := cloudServiceInboundPath("heartbeat", "t", "w", "x", ""); got != "" {
		t.Fatalf("empty comment must not build URL, got %q", got)
	}
	if got := cloudServiceInboundPath("heartbeat", "t", "w", "x", "-"); got != "" {
		t.Fatalf("dash comment must not build URL, got %q", got)
	}
}

func TestParseCloudInboundPathWithCommentSegment(t *testing.T) {
	path := "/api/tenant/t1/workspace/w1/task/task1/comment/cmt_1/cloud/server-container-token/heartbeat/"
	match, ok := parseCloudInboundPath(path)
	if !ok {
		t.Fatal("expected comment-scoped path to match")
	}
	if match.Action != "heartbeat" {
		t.Fatalf("action=%q want heartbeat", match.Action)
	}
	if match.CommentID != "cmt_1" || match.TaskID != "task1" {
		t.Fatalf("unexpected scope: %+v", match)
	}
}

func TestNestedTenantCloudPrefixWithComment(t *testing.T) {
	got := nestedTenantCloudPrefix("http://cred:8015", "t", "w", "x", "cmt_1")
	want := "http://cred:8015/api/tenant/t/workspace/w/task/x/comment/cmt_1/cloud"
	if got != want {
		t.Fatalf("prefix=%q want %q", got, want)
	}
}

func TestNestedTenantCloudPrefixRejectsEmptyComment(t *testing.T) {
	if got := nestedTenantCloudPrefix("http://cred:8015", "t", "w", "x", ""); got != "" {
		t.Fatalf("empty comment must not build prefix, got %q", got)
	}
	if got := nestedTenantCloudPrefix("http://cred:8015", "t", "w", "x", "-"); got != "" {
		t.Fatalf("dash comment must not build prefix, got %q", got)
	}
}

func TestForwardsToCredentialService(t *testing.T) {
	wantTrue := []string{
		"exchange-refresh",
		"refresh-access",
		"task-detail",
		"repo-clone-credentials",
		"layer-github-oauth-access-tokens",
	}
	for _, a := range wantTrue {
		if !forwardsToCredentialService(a) {
			t.Fatalf("%s should forward to credential service", a)
		}
	}
	wantFalse := []string{"heartbeat", "register-reachability", "git-clone-progress", "boot-progress", "relay-status-push", "feature-params-env", "model-budget-usage"}
	for _, a := range wantFalse {
		if forwardsToCredentialService(a) {
			t.Fatalf("%s should not forward to credential service", a)
		}
	}
}

func TestForwardsToCloudService(t *testing.T) {
	wantTrue := []string{
		"register-reachability",
		"heartbeat",
		"git-clone-progress",
		"boot-progress",
		"layer-graph-push",
		"layer-changes-push",
		"feature-params-env",
		"relay-status-push",
		"model-budget-usage",
		"request-machine-release",
		"runtime-event",
		"job-stream-push",
		"job-step-full-push",
	}
	for _, a := range wantTrue {
		if !forwardsToCloudService(a) {
			t.Fatalf("%s should forward to cloud service", a)
		}
	}
	wantFalse := []string{"task-detail", "layer-github-oauth-access-tokens"}
	for _, a := range wantFalse {
		if forwardsToCloudService(a) {
			t.Fatalf("%s should not forward to cloud service", a)
		}
	}
}
