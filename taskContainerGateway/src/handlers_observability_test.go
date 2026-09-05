package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tracelog"
)

func TestHandleContainerCompute_EmitsForwardStageOnAuthUnreachable(t *testing.T) {
	cfg = serviceConfig{
		Host:              "127.0.0.1",
		Port:              8014,
		TaskAuthURL:       "http://127.0.0.1:1",
		CloudServiceURL:   "http://127.0.0.1:1",
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	tracelog.Init("task-container-gateway-test")

	path := "/api/tenant/827923618468040704/workspace/827923618602258432/task/848546827193511936/cloud/compute/container-layer-git-commit/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"layer_id":"layer-1","container_page_url":"http://127.0.0.1:8765/ui/x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-obs-test12345678")
	req.Header.Set(tracelog.ParentSpanHeader, "aabbccddeeff0011")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	_ = w.Close()
	os.Stdout = old
	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(out, `"msg":"forward_stage"`) {
		t.Fatalf("expected forward_stage log, got %q", out)
	}
	if !strings.Contains(out, "trace-obs-test12345678") {
		t.Fatalf("expected trace_id in logs, got %q", out)
	}
	if !strings.Contains(out, "auth_validate") {
		t.Fatalf("expected auth_validate stage, got %q", out)
	}
}

func TestHandleContainerCompute_FullChainMockAuthCloudAndUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tenant/1/workspace/2/task/3/layers/layer-1/git/commit" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer upstream.Close()

	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/container-gateway/validate-session/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user_id": "1", "auth_method": "token", "scope_ok": true,
		})
	}))
	defer auth.Close()

	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "1", "tenant_id": "1", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       auth.URL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	old := os.Stdout
	pr, pw, _ := os.Pipe()
	os.Stdout = pw
	tracelog.Init("task-container-gateway-test")

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-commit/"
	body := `{"layer_id":"layer-1","container_page_url":"` + upstream.URL + `/ui/x"}`
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-fullchain123456")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	_ = pw.Close()
	os.Stdout = old
	var logs strings.Builder
	_, _ = io.Copy(&logs, pr)
	out := logs.String()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	for _, stage := range []string{"auth_validate", "cloud_member", "cloud_lookup", "cloud_resolve", "upstream_forward"} {
		if !strings.Contains(out, `"forward_stage":"`+stage+`"`) {
			t.Fatalf("missing %q in logs: %s", stage, out)
		}
	}
}
