package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestUpsertRelayStartupSessionViaCloudOnTokenInitAndStart(t *testing.T) {
	var mu sync.Mutex
	var upserts []map[string]any

	cloudUpsert := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/relay-startup-session/upsert/":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			mu.Lock()
			upserts = append(upserts, body)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "workflow_id": body["workflow_id"]})
		case r.URL.Path == "/api/internal/runtime-session/open/" ||
			r.URL.Path == "/api/internal/runtime-session/open":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": "http://127.0.0.1:9", "access_token": "tok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloudUpsert.Close()

	cred := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/token/init/") {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "issued-access-token",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer cred.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, "http://127.0.0.1:9", "tok")

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/start" {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		http.NotFound(w, r)
	}))
	defer relay.Close()

	cfg = serviceConfig{
		TaskAuthURL:            authURL,
		CloudServiceURL:        cloudUpsert.URL,
		InternalSecret:         "secret",
		CredentialServiceURL:   cred.URL,
		CloudInternalSecret:    "cloud-secret",
		RelayToTraeURL:         relay.URL,
		RelayTaskAPIOrigin:     "http://127.0.0.1:8001",
		RelayBusinessAPIOrigin: "http://127.0.0.1:8765",
		ForwardReadSec:         5,
		ForwardConnectSec:      1,
	}
	mux := http.NewServeMux()
	mountRoutes(mux)

	tokenInitPath := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/token-init/"
	req := httptest.NewRequest(http.MethodPost, tokenInitPath, strings.NewReader(`{"comment_id":"cmt1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token t")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("token-init status=%d body=%s", rec.Code, rec.Body.String())
	}

	startPath := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/start/"
	req = httptest.NewRequest(http.MethodPost, startPath, strings.NewReader(`{"comment_id":"cmt1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token t")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(upserts) < 1 {
		t.Fatalf("expected cloud upserts, got %d", len(upserts))
	}
}
