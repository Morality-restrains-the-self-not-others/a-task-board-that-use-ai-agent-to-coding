package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// startAuthCloudHotPathMocks serves taskAuth validate-session + taskCloudService
// tenant-member / lookup / container-target for zero-Django hot-path tests.
// baseURL/token are returned by container-target (usually the upstream mock URL).
func startAuthCloudHotPathMocks(t *testing.T, baseURL, accessToken string) (authURL, cloudURL string) {
	t.Helper()
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/container-gateway/validate-session/" &&
			r.URL.Path != "/api/internal/container-gateway/validate-session" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user_id": "42", "auth_method": "token", "scope_ok": true,
		})
	}))
	t.Cleanup(auth.Close)

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
				"base_url": baseURL, "access_token": accessToken,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(cloud.Close)
	return auth.URL, cloud.URL
}
