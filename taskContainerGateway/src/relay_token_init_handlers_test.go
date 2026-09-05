package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRelayTokenInitEnvPreparePrecheck(t *testing.T) {
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/token/init/") && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":        "ok",
				"access_token":  "issued-access-token",
				"refresh_token": "issued-refresh",
				"expires_at":    "2099-01-01T00:00:00Z",
			})
		case strings.HasSuffix(r.URL.Path, "/repo-clone-credentials/") && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"repo_clone_credentials": map[string]any{"repo-a": map[string]string{"clone_url": "https://example/a.git"}},
				"repo_count":             1,
				"trace_id":               "trace-precheck-1",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cred.Close()

	authURL, cloudURL := startAuthCloudHotPathMocks(t, "http://127.0.0.1:9", "tok")
	cfg = serviceConfig{
		TaskAuthURL:            authURL,
		CloudServiceURL:        cloudURL,
		InternalSecret:         "secret",
		CredentialServiceURL:   cred.URL,
		RelayTaskAPIOrigin:     "http://127.0.0.1:8001",
		RelayBusinessAPIOrigin: "http://127.0.0.1:8765",
		ForwardReadSec:         5,
		ForwardConnectSec:      1,
	}

	mux := http.NewServeMux()
	mountRoutes(mux)

	t.Run("env-prepare", func(t *testing.T) {
		path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/env-prepare/"
		req := httptest.NewRequest(http.MethodGet, path+"?task_id=task1", nil)
		req.Header.Set("Authorization", "Token test-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "success" {
			t.Fatalf("status=%v", body["status"])
		}
		env, _ := body["env"].(map[string]any)
		if env["ACCESS_TOKEN"] != task2appAccessTokenPlaceholder {
			t.Fatalf("ACCESS_TOKEN=%v", env["ACCESS_TOKEN"])
		}
		if env["TASK_API_ENDPOINT_ORIGIN"] != "http://127.0.0.1:8001" {
			t.Fatalf("TASK_API=%v", env["TASK_API_ENDPOINT_ORIGIN"])
		}
	})

	t.Run("token-init", func(t *testing.T) {
		path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/token-init/"
		payload := `{"env":{"ACCESS_TOKEN":"__TASK2APP_ACCESS_TOKEN__"},"tenant_id":"t1","workspace_id":"w1","task_id":"task1","comment_id":"cmt1"}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Token test-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "ok" || body["token_initialized"] != true {
			t.Fatalf("body=%v", body)
		}
		preview, _ := body["env_preview"].(map[string]any)
		if preview["BUSINESS_API_ENDPOINT_ORIGIN"] != "http://127.0.0.1:8765" {
			t.Fatalf("env_preview=%v", preview)
		}
	})

	t.Run("repo-credentials-precheck", func(t *testing.T) {
		path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/repo-credentials-precheck/"
		payload := `{"env":{"ACCESS_TOKEN":"__TASK2APP_ACCESS_TOKEN__"},"tenant_id":"t1","workspace_id":"w1","task_id":"task1","comment_id":"cmt1"}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Token test-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != "ok" {
			t.Fatalf("body=%v", body)
		}
		if body["repo_count"] != float64(1) {
			t.Fatalf("repo_count=%v", body["repo_count"])
		}
	})
}

func TestHandleRelayRepoCredentialsPrecheckConflict(t *testing.T) {
	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/token/init/") {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
			return
		}
		if strings.Contains(r.URL.Path, "repo-clone-credentials") {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"detail":                   "incomplete",
				"error_code":               "REPO_CLONE_CREDENTIALS_INCOMPLETE",
				"missing_repo_credentials": []any{map[string]string{"repo": "a"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer cred.Close()

	authURL, cloudURL := startAuthCloudHotPathMocks(t, "http://127.0.0.1:9", "tok")
	cfg = serviceConfig{
		TaskAuthURL:          authURL,
		CloudServiceURL:      cloudURL,
		CredentialServiceURL: cred.URL,
	}
	mux := http.NewServeMux()
	mountRoutes(mux)

	path := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/repo-credentials-precheck/"
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"task_id":"task1","comment_id":"cmt1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token x")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["error_code"] != "REPO_CLONE_CREDENTIALS_INCOMPLETE" {
		t.Fatalf("body=%v", body)
	}
}
