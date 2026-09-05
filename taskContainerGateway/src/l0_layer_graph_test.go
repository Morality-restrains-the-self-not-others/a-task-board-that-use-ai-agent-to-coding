package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleContainerLayerGraph_JobsFailDegradesEmptyJobs(t *testing.T) {
	var sawLayers, sawJobs bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Access-Token") != "tok" {
			http.Error(w, "token", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/layers") && r.Method == http.MethodGet:
			sawLayers = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"layers": []any{
					map[string]any{"layer_id": "L1", "job_status": "completed"},
				},
				"layers_root":        "/layers",
				"bootstrap_layer_id": "L1",
			})
		case strings.HasSuffix(r.URL.Path, "/jobs") && r.Method == http.MethodGet:
			sawJobs = true
			http.Error(w, `{"detail":"timeout"}`, http.StatusGatewayTimeout)
		default:
			http.NotFound(w, r)
		}
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

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-layer-graph/"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Token test-token-abc12345")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !sawLayers || !sawJobs {
		t.Fatalf("upstream layers=%v jobs=%v", sawLayers, sawJobs)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	layers, _ := body["layers"].([]any)
	if len(layers) != 1 {
		t.Fatalf("layers=%v", body["layers"])
	}
	jobs, _ := body["jobs"].([]any)
	if jobs == nil {
		t.Fatalf("jobs should be empty array, got nil body=%v", body)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected empty jobs on degrade, got %v", jobs)
	}
	if body["bootstrap_layer_id"] != "L1" {
		t.Fatalf("bootstrap=%v", body["bootstrap_layer_id"])
	}
}
