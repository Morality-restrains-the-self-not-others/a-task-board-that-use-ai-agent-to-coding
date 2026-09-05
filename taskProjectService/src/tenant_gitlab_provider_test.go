package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRepoHostMatchesOriginPathA(t *testing.T) {
	origin := "http://115.29.110.74"
	repo := "http://115.29.110.74/example-user/somanyad.git"
	if !repoHostMatchesOrigin(repo, origin) {
		t.Fatalf("Path A repo should match origin %s", origin)
	}
	if repoHostMatchesOrigin("https://gitlab.daydaymoney.com/g/r.git", origin) {
		t.Fatal("platform GitLab must not match Path A origin")
	}
}

func TestMatchRepoProviderUsesTenantPathANotDefault(t *testing.T) {
	initProviderTestConfig(t)
	oldLookup := lookupTenantGitLabConn
	oldResolver := providerResolver
	t.Cleanup(func() {
		lookupTenantGitLabConn = oldLookup
		providerResolver = oldResolver
	})
	providerResolver = &ProviderResolver{}
	lookupTenantGitLabConn = func(tenantID string, trace map[string]string) *tenantGitLabConn {
		if tenantID != "877397588196749312" {
			t.Fatalf("tenantID=%q", tenantID)
		}
		return &tenantGitLabConn{
			Configured:  true,
			Active:      true,
			ProviderKey: "gitlab:tenant-877397588196749312",
			BaseURL:     "http://115.29.110.74",
		}
	}

	repo := "http://115.29.110.74/example-user/somanyad.git"
	match := matchRepoProvider(repo, "877397588196749312")
	if match.ProviderKey != "gitlab:tenant-877397588196749312" {
		t.Fatalf("ProviderKey=%q, want gitlab:tenant-… (must not borrow gitlab:default)", match.ProviderKey)
	}
}

// TestTenantGitLabConnCacheInvalidatedAfterChange verifies that after a tenant
// connection PUT/DELETE the in-process cache is dropped so the next
// matchRepoProvider uses the new origin instead of the stale cached base_url.
func TestTenantGitLabConnCacheInvalidatedAfterChange(t *testing.T) {
	initProviderTestConfig(t)
	providerResolver = &ProviderResolver{}
	oldClient := gitHTTPClient
	t.Cleanup(func() {
		gitHTTPClient = oldClient
		tenantGitLabCacheMu.Lock()
		tenantGitLabCache = map[string]tenantGitLabCacheEntry{}
		tenantGitLabCacheMu.Unlock()
	})

	var baseURL atomic.Value
	baseURL.Store("http://gitlab-a.example.com")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tenantGitLabConn{
			Configured:  true,
			Active:      true,
			ProviderKey: "gitlab:tenant-T1",
			BaseURL:     baseURL.Load().(string),
		})
	}))
	t.Cleanup(srv.Close)

	gitHTTPClient = srv.Client()
	oldBase := cfg.GitoauthBaseURL
	cfg.GitoauthBaseURL = srv.URL
	t.Cleanup(func() { cfg.GitoauthBaseURL = oldBase })

	// Prime the cache with base_url A.
	conn := defaultLookupTenantGitLabConn("T1", nil)
	if conn == nil || conn.BaseURL != "http://gitlab-a.example.com" {
		t.Fatalf("prime lookup conn=%+v want base_url A", conn)
	}
	if got := matchRepoProvider("http://gitlab-a.example.com/g/r.git", "T1").ProviderKey; got != "gitlab:tenant-T1" {
		t.Fatalf("before invalidate ProviderKey=%q want gitlab:tenant-T1", got)
	}

	// Admin PUT changes base_url to B; invalidation drops the cache.
	baseURL.Store("http://gitlab-b.example.com")
	invalidateTenantGitLabConnCache("T1")

	if got := matchRepoProvider("http://gitlab-b.example.com/g/r.git", "T1").ProviderKey; got != "gitlab:tenant-T1" {
		t.Fatalf("after invalidate ProviderKey=%q want gitlab:tenant-T1 (new origin)", got)
	}
	if got := matchRepoProvider("http://gitlab-a.example.com/g/r.git", "T1").ProviderKey; got != "" {
		t.Fatalf("stale origin must no longer match tenant conn, got %q", got)
	}
}

func TestInternalTenantGitLabCacheInvalidateHandler(t *testing.T) {
	invalidateTenantGitLabConnCache("") // clear
	tenantGitLabCacheMu.Lock()
	tenantGitLabCache["T2"] = tenantGitLabCacheEntry{conn: &tenantGitLabConn{Configured: true}, expiresAt: time.Now().Add(5 * time.Second)}
	tenantGitLabCacheMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskproject/tenant-gitlab-cache/invalidate?company_id=T2", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantGitLabCacheInvalidate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	tenantGitLabCacheMu.Lock()
	_, still := tenantGitLabCache["T2"]
	tenantGitLabCacheMu.Unlock()
	if still {
		t.Fatal("cache entry T2 not dropped by invalidate handler")
	}
}

func TestMatchProviderUnknownDotGitDoesNotBorrowDefault(t *testing.T) {
	resolver := &ProviderResolver{}
	key := resolver.ResolveProvider("http://115.29.110.74/example-user/somanyad.git")
	if key == "gitlab:default" {
		t.Fatal("unmatched Path A *.git must not borrow gitlab:default")
	}
	if !resolver.IsGitLabRepo("http://115.29.110.74/example-user/somanyad.git") {
		t.Fatal("Path A *.git must still be treated as GitLab")
	}
}
