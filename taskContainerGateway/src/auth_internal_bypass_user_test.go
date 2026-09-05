package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func TestAuthorizeContainerRequest_InternalBypassUsesForwardedAuthUser(t *testing.T) {
	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg.InternalSecret = "gw-secret"
	cfg.TaskAuthURL = "http://127.0.0.1:1" // must not be called
	cfg.CloudServiceURL = "http://127.0.0.1:1"

	req := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/container-layer-git-push/", nil)
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "gw-secret")
	req.Header.Set("X-Auth-User-Id", "877397583960502272")

	session, status, _ := authorizeContainerRequest(context.Background(), req, scope{TenantID: "t1"})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if session.UserID != "877397583960502272" {
		t.Fatalf("UserID=%q want forwarded browser user, not internal_gateway sentinel", session.UserID)
	}
	if session.AuthMethod != "internal_secret" {
		t.Fatalf("AuthMethod=%q", session.AuthMethod)
	}
}

func TestForwardedTrustedUserID_IgnoresSentinelHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-User-Id", "internal_gateway")
	req.Header.Set("X-User-Id", "internal")
	if got := forwardedTrustedUserID(req); got != "" {
		t.Fatalf("got=%q want empty so caller can fall back", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Auth-User-Id", "internal")
	req2.Header.Set("X-User-Id", "877397583960502272")
	if got := forwardedTrustedUserID(req2); got != "877397583960502272" {
		t.Fatalf("got=%q want X-User-Id after skipping internal sentinel", got)
	}
}

func TestAuthorizeContainerRequest_InternalBypassFallsBackWithoutUserHeader(t *testing.T) {
	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg.InternalSecret = "gw-secret"
	cfg.TaskAuthURL = "http://127.0.0.1:1"
	cfg.CloudServiceURL = "http://127.0.0.1:1"

	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/container-layer-graph/", nil)
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "gw-secret")

	session, status, _ := authorizeContainerRequest(context.Background(), req, scope{TenantID: "t1"})
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if session.UserID != "internal_gateway" {
		t.Fatalf("UserID=%q want internal_gateway when Cloud omitted user headers", session.UserID)
	}
}

func TestHandleContainerLayerGitPush_InternalBypassForwardsUserToPrepare(t *testing.T) {
	var gotPrepareUser string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pushed": true})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{"company_member_id": "m1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/prepare"):
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			gotPrepareUser, _ = body["user_id"].(string)
			if gotPrepareUser == "internal_gateway" {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok": false, "status": 404, "detail": "Git 身份不存在或不属于当前租户",
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                    true,
				"use_oauth_access_push": true,
				"push_body":             map[string]any{"target_branch": "main"},
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/layer-git-push/complete"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/cloud/compute/container-layer-git-push/tenant_id/t1/workspace_id/w1/task_id/task1/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(
		`{"layer_id":"L1","identity_id":"gi_877397592462356480","prefer_container_remote":false}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TaskContainerGateway-Internal-Secret", "secret")
	req.Header.Set("X-Auth-User-Id", "877397583960502272")
	req.Header.Set(tracelog.Header, "trace-internal-bypass-user-01")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s prepare_user=%q", rec.Code, rec.Body.String(), gotPrepareUser)
	}
	if gotPrepareUser != "877397583960502272" {
		t.Fatalf("prepare user_id=%q want forwarded auth user (this is the 404 Git-identity bug)", gotPrepareUser)
	}
}
