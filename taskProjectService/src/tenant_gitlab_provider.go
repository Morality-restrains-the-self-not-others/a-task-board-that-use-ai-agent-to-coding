package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type tenantGitLabConn struct {
	Configured  bool   `json:"configured"`
	Active      bool   `json:"active"`
	ProviderKey string `json:"provider_key"`
	BaseURL     string `json:"base_url"`
}

type tenantGitLabCacheEntry struct {
	conn      *tenantGitLabConn
	expiresAt time.Time
}

var (
	tenantGitLabCacheMu sync.Mutex
	tenantGitLabCache   = map[string]tenantGitLabCacheEntry{}
)

// lookupTenantGitLabConn fetches the tenant Path A GitLab connection.
// Tests replace this hook.
var lookupTenantGitLabConn = defaultLookupTenantGitLabConn

func requestTenantID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := strings.TrimSpace(r.Header.Get("X-Auth-Tenant-Id")); v != "" {
		return v
	}
	return strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
}

func repoHostMatchesOrigin(repoURL, origin string) bool {
	rh, rn := extractHostNetloc(repoURL)
	oh, on := extractHostNetloc(origin)
	if rn != "" && on != "" && strings.EqualFold(rn, on) {
		return true
	}
	return rh != "" && oh != "" && strings.EqualFold(rh, oh)
}

func yamlExactProviderMatch(repoURL string, match providerMatch) bool {
	key := strings.TrimSpace(match.ProviderKey)
	if key == "" || key == "gitlab:default" {
		return false
	}
	if providerResolver == nil {
		return false
	}
	host, netloc := extractHostNetloc(repoURL)
	for _, e := range providerResolver.entries {
		if e.ProviderKey != key {
			continue
		}
		if e.Netloc != "" && netloc != "" && e.Netloc == netloc {
			return true
		}
		if e.Host != "" && host != "" && e.Host == host {
			return true
		}
	}
	return false
}

func matchRepoProvider(repoURL, tenantID string, traceHeaders ...map[string]string) providerMatch {
	var yamlMatch providerMatch
	if providerResolver != nil {
		yamlMatch = providerResolver.matchProvider(repoURL)
	}
	trace := outboundTrace(traceHeaders...)
	if conn := lookupTenantGitLabConn(tenantID, trace); conn != nil && conn.Configured && conn.Active && strings.TrimSpace(conn.BaseURL) != "" {
		if repoHostMatchesOrigin(repoURL, conn.BaseURL) {
			if !yamlExactProviderMatch(repoURL, yamlMatch) {
				gitoauth := yamlMatch.GitoauthBase
				if strings.TrimSpace(gitoauth) == "" {
					gitoauth = cfg.GitoauthBaseURL
				}
				logInfo("path-a gitlab provider selected key="+conn.ProviderKey, traceHeaderID(trace))
				return providerMatch{ProviderKey: conn.ProviderKey, GitoauthBase: gitoauth}
			}
		}
	}
	return yamlMatch
}

func defaultLookupTenantGitLabConn(tenantID string, trace map[string]string) *tenantGitLabConn {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return nil
	}
	now := time.Now()
	tenantGitLabCacheMu.Lock()
	if ent, ok := tenantGitLabCache[tid]; ok && now.Before(ent.expiresAt) {
		conn := ent.conn
		tenantGitLabCacheMu.Unlock()
		return conn
	}
	tenantGitLabCacheMu.Unlock()

	base := strings.TrimRight(strings.TrimSpace(cfg.GitoauthBaseURL), "/")
	if base == "" {
		return nil
	}
	u := base + "/api/internal/git-oauth/gitlab-tenant-connection/?company_id=" + url.QueryEscape(tid)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	for k, v := range gitoauthBridgeHeaders() {
		req.Header.Set(k, v)
	}
	for k, v := range outboundTrace(trace) {
		req.Header.Set(k, v)
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		logWarn("tenant-gitlab-connection lookup failed: "+err.Error(), traceHeaderID(trace))
		return nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logWarn(fmt.Sprintf("tenant-gitlab-connection http %d", resp.StatusCode), traceHeaderID(trace))
		return nil
	}
	var conn tenantGitLabConn
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &conn)
	}
	if !conn.Configured || !conn.Active {
		conn = tenantGitLabConn{}
	}
	tenantGitLabCacheMu.Lock()
	tenantGitLabCache[tid] = tenantGitLabCacheEntry{conn: &conn, expiresAt: now.Add(5 * time.Second)}
	tenantGitLabCacheMu.Unlock()
	return &conn
}

func traceHeaderID(trace map[string]string) string {
	if trace == nil {
		return ""
	}
	return strings.TrimSpace(trace["X-Trace-Id"])
}

// invalidateTenantGitLabConnCache drops the cached Path A connection for a tenant
// (or the whole cache when tenantID is empty), so the next lookup re-fetches from
// taskGitOauth. Called by the internal invalidation endpoint after a
// tenant-connection PUT/DELETE; the 30s→5s TTL below is only a backstop.
func invalidateTenantGitLabConnCache(tenantID string) {
	tenantGitLabCacheMu.Lock()
	defer tenantGitLabCacheMu.Unlock()
	if strings.TrimSpace(tenantID) == "" {
		tenantGitLabCache = map[string]tenantGitLabCacheEntry{}
		return
	}
	delete(tenantGitLabCache, strings.TrimSpace(tenantID))
}

// handleInternalTenantGitLabCacheInvalidate serves
// POST /api/internal/taskproject/tenant-gitlab-cache/invalidate?company_id=...
// so taskGitOauth can drop the Path A connection cache right after a write.
func handleInternalTenantGitLabCacheInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireProjectInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	invalidateTenantGitLabConnCache(tenantID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"invalidated": tenantID,
	})
}
