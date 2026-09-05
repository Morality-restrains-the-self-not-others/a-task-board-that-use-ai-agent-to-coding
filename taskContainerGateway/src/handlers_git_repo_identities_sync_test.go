package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func TestHandleContainerLayerGitRepoIdentitiesSync_PrepareAndUpstream(t *testing.T) {
	var upstreamPath string
	var upstreamBody map[string]any

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Access-Token") != "tok" {
			http.Error(w, "token", http.StatusUnauthorized)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &upstreamBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "layer_id": "layer-1", "applied_count": 1, "results": []any{},
		})
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		case r.URL.Path == "/api/internal/layer-git-repo-identities/prepare" ||
			r.URL.Path == "/api/internal/layer-git-repo-identities/prepare/":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			if body["user_id"] != "42" || body["tenant_id"] != "1" {
				http.Error(w, "bad prepare", http.StatusBadRequest)
				return
			}
			repos, _ := body["repos"].([]any)
			if len(repos) != 1 {
				http.Error(w, "bad repos", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"repos": []map[string]string{{
					"repo_match_key": "github.com/org/my-app",
					"user_name":      "Bob",
					"user_email":     "bob@example.com",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-git-repo-identities-sync/"
	body := `{"layer_id":"layer-1","repos":[{"repo_url":"https://github.com/org/my-app.git","identity_id":"9"}],"container_page_url":"` + upstream.URL + `/ui/x"}`
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-identsync1234567890")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	wantPath := "/api/tenant/1/workspace/2/task/3/layers/layer-1/git/repo-identities/sync"
	if upstreamPath != wantPath {
		t.Fatalf("upstream path=%q want %q", upstreamPath, wantPath)
	}
	repos, _ := upstreamBody["repos"].([]any)
	if len(repos) != 1 {
		t.Fatalf("upstream body=%v", upstreamBody)
	}
	row, _ := repos[0].(map[string]any)
	if row["repo_match_key"] != "github.com/org/my-app" ||
		row["user_name"] != "Bob" ||
		row["user_email"] != "bob@example.com" {
		t.Fatalf("upstream repos row=%v", row)
	}
	if _, hasURL := row["repo_url"]; hasURL {
		t.Fatalf("browser repo_url must not be forwarded: %v", row)
	}
}
