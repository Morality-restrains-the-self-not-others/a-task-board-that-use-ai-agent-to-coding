package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessTokenNeedsServerIssue(t *testing.T) {
	cases := []struct {
		token string
		want  bool
	}{
		{"", true},
		{"__TASK2APP_ACCESS_TOKEN__", true},
		{"real-token-value", false},
		{"${ACCESS_TOKEN}", true},
	}
	for _, tc := range cases {
		if got := accessTokenNeedsServerIssue(tc.token); got != tc.want {
			t.Fatalf("token=%q got=%v want=%v", tc.token, got, tc.want)
		}
	}
}

func TestRelayLifecycleAction(t *testing.T) {
	method, ok := relayLifecycleAction("start")
	if !ok || method != "POST" {
		t.Fatalf("start action method=%s ok=%v", method, ok)
	}
	method, ok = relayLifecycleAction("token-init")
	if !ok || method != "POST" {
		t.Fatalf("token-init action method=%s ok=%v", method, ok)
	}
	method, ok = relayLifecycleAction("env-prepare")
	if !ok || method != "GET" {
		t.Fatalf("env-prepare action method=%s ok=%v", method, ok)
	}
	method, ok = relayLifecycleAction("repo-credentials-precheck")
	if !ok || method != "POST" {
		t.Fatalf("repo-credentials-precheck action method=%s ok=%v", method, ok)
	}
	if _, ok := relayLifecycleAction("health"); ok {
		t.Fatal("health should not be lifecycle action")
	}
	method, goPath, ok := relayGoPath("clear-logs")
	if !ok || method != "POST" {
		t.Fatalf("clear-logs should be proxied POST, method=%s goPath=%q ok=%v", method, goPath, ok)
	}
	got := relayClearLogsGoPath(relayScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"})
	want := "/v1/tenant/t1/workspace/w1/task/task1/clear-logs"
	if got != want {
		t.Fatalf("clear-logs go path=%q want=%q", got, want)
	}
}

func TestParseRelayToTraePathLifecycle(t *testing.T) {
	for _, action := range []string{"register", "start", "stop", "token-init", "env-prepare", "repo-credentials-precheck"} {
		path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/" + action + "/"
		match, ok := parseRelayToTraePath(path)
		if !ok || match.SubAction != action {
			t.Fatalf("path %s match=%+v ok=%v", path, match, ok)
		}
	}
}

func TestRelayEnvPreview(t *testing.T) {
	cfg.RelayTaskAPIOrigin = "http://task-api.example"
	cfg.RelayBusinessAPIOrigin = "http://biz.example/api"
	env := relayEnvPreview()
	if env["TASK_API_ENDPOINT_ORIGIN"] != "http://task-api.example" {
		t.Fatalf("task api=%q", env["TASK_API_ENDPOINT_ORIGIN"])
	}
	if env["BUSINESS_API_ENDPOINT_ORIGIN"] != "http://biz.example/api" {
		t.Fatalf("business=%q", env["BUSINESS_API_ENDPOINT_ORIGIN"])
	}
}

func TestBuildRelayStartEnv(t *testing.T) {
	cfg.RelayTaskAPIOrigin = "http://task-api.example"
	cfg.RelayBusinessAPIOrigin = "http://biz.example/api"
	env := buildRelayStartEnv(scope{TenantID: "t", WorkspaceID: "w", TaskID: "tk"}, map[string]string{
		"BUSINESS_API_ENDPOINT_ORIGIN": "http://override.example/api",
	}, "tok-123", "trace-abc")
	if env["ACCESS_TOKEN"] != "tok-123" {
		t.Fatalf("access token=%q", env["ACCESS_TOKEN"])
	}
	if env["BUSINESS_API_ENDPOINT_ORIGIN"] != "http://override.example/api" {
		t.Fatalf("business override=%q", env["BUSINESS_API_ENDPOINT_ORIGIN"])
	}
	if env["TRACE_ID"] != "trace-abc" {
		t.Fatalf("trace=%q", env["TRACE_ID"])
	}
}

func TestResolveRelayStartImage_DirectImage(t *testing.T) {
	img, id, status, _ := resolveRelayStartImage(context.Background(), "t1", map[string]any{
		"image": "registry.example/app:v1",
	})
	if status != http.StatusOK || img != "registry.example/app:v1" || id != "" {
		t.Fatalf("img=%q id=%q status=%d", img, id, status)
	}
}

func TestResolveRelayStartImage_EmptyLegacy(t *testing.T) {
	img, id, status, _ := resolveRelayStartImage(context.Background(), "t1", map[string]any{})
	if status != http.StatusOK || img != "" || id != "" {
		t.Fatalf("img=%q id=%q status=%d", img, id, status)
	}
}

func TestResolveRelayStartImage_InstalledImageID(t *testing.T) {
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/internal/image/resolve") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("tenant_id") != "t1" || r.URL.Query().Get("installed_image_id") != "img-9" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"image": "registry.example/app:v2"})
	}))
	defer cloud.Close()
	prev := cfg.CloudServiceURL
	cfg.CloudServiceURL = cloud.URL
	t.Cleanup(func() { cfg.CloudServiceURL = prev })

	img, id, status, body := resolveRelayStartImage(context.Background(), "t1", map[string]any{
		"installed_image_id": "img-9",
	})
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, string(body))
	}
	if img != "registry.example/app:v2" || id != "img-9" {
		t.Fatalf("img=%q id=%q", img, id)
	}
}

func TestResolveRelayStartImage_ResolveFailed(t *testing.T) {
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "message": "not found"})
	}))
	defer cloud.Close()
	prev := cfg.CloudServiceURL
	cfg.CloudServiceURL = cloud.URL
	t.Cleanup(func() { cfg.CloudServiceURL = prev })

	_, _, status, _ := resolveRelayStartImage(context.Background(), "t1", map[string]any{
		"installed_image_id": "missing",
	})
	if status != http.StatusNotFound {
		t.Fatalf("status=%d", status)
	}
}

