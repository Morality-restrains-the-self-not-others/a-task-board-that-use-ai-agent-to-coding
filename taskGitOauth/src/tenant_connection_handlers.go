package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func (a *App) handleTenantGitlabOAuthConnection(w http.ResponseWriter, r *http.Request) {
	tid := extractTenantIDFromPath(r.URL.Path)
	if tid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "missing tenant id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		a.handleTenantGitlabConnectionGet(w, r, tid)
	case http.MethodPut:
		a.handleTenantGitlabConnectionPut(w, r, tid)
	case http.MethodDelete:
		a.handleTenantGitlabConnectionDelete(w, r, tid)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// isTenantGitlabConnectionPath reports whether path is a tenant GitLab OAuth
// connection public API (canonical key/value or legacy positional form).
func isTenantGitlabConnectionPath(path string) bool {
	const prefix = "/api/git-oauth/tenant-connection/"
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, prefix), "/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		return false
	}
	// Canonical: tenant_id/{tid}[/optional...]
	if parts[0] == "tenant_id" {
		return strings.TrimSpace(parts[1]) != ""
	}
	// Legacy: {tid}/gitlab-oauth-connection[/...]
	return parts[1] == "gitlab-oauth-connection" && strings.TrimSpace(parts[0]) != ""
}

func isTenantGitlabReachabilityPath(path string) bool {
	if !isTenantGitlabConnectionPath(path) {
		return false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/api/git-oauth/tenant-connection/"), "/"), "/")
	return len(parts) >= 3 && parts[len(parts)-1] == "reachability"
}

// extractTenantIDFromPath parses tenant id from:
//   - /api/git-oauth/tenant-connection/tenant_id/{tid}/  (canonical, api-url-path-design)
//   - /api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/  (legacy)
func extractTenantIDFromPath(path string) string {
	const prefix = "/api/git-oauth/tenant-connection/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, prefix), "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	if parts[0] == "tenant_id" {
		return strings.TrimSpace(parts[1])
	}
	if parts[1] == "gitlab-oauth-connection" {
		return strings.TrimSpace(parts[0])
	}
	return ""
}

func (a *App) handleTenantGitlabConnectionGet(w http.ResponseWriter, r *http.Request, tenantID string) {
	if !a.ensureTenantMember(w, r, tenantID) {
		return
	}
	writeJSON(w, http.StatusOK, a.tenantConnectionPublicJSON(tenantID))
}

func (a *App) handleTenantGitlabConnectionPut(w http.ResponseWriter, r *http.Request, tenantID string) {
	if !a.ensureTenantAdmin(w, r, tenantID) {
		return
	}
	body, err := readJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	baseURL, err := domain.NormalizeBaseURL(asString(body["base_url"]))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	clientID := strings.TrimSpace(asString(body["client_id"]))
	clientSecret := strings.TrimSpace(asString(body["client_secret"]))
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "client_id required"})
		return
	}
	remark := strings.TrimSpace(asString(body["remark"]))
	existing, _ := a.DB.GetTenantGitLabConnection(tenantID)
	intranet := false
	if existing != nil {
		intranet = existing.Intranet
	}
	if _, ok := body["intranet"]; ok {
		intranet = asBool(body["intranet"])
	}
	cipher := ""
	if clientSecret != "" {
		var encErr error
		cipher, encErr = a.Fernet.Encrypt(clientSecret)
		if encErr != nil || cipher == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "encrypt failed"})
			return
		}
	} else if existing != nil {
		cipher = existing.ClientSecretEnc
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "client_secret required"})
		return
	}
	redirectURI := domain.DefaultRedirectURI(a.Cfg.PublicBaseURL, tenantID)
	scope := domain.DefaultTenantGitLabScope
	row, err := a.DB.UpsertTenantGitLabConnection(&infrastructure.TenantGitLabOAuthConnectionRow{
		CompanyID:       tenantID,
		BaseURL:         baseURL,
		ClientID:        clientID,
		ClientSecretEnc: cipher,
		Remark:          remark,
		RedirectURI:     redirectURI,
		Scope:           scope,
		Active:          true,
		Intranet:        intranet,
	})
	if err != nil {
		logWarn("tenant gitlab connection upsert: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "upsert failed"})
		return
	}
	publishTenantGitLabOAuthConnectionUpserted(a.Cfg, map[string]any{
		"company_id":       tenantID,
		"provider_key":     domain.ProviderKeyForCompany(tenantID),
		"service_provider": domain.TenantServiceProvider(tenantID),
		"base_url":         row.BaseURL,
		"client_id":        row.ClientID,
		"remark":           row.Remark,
		"redirect_uri":     row.RedirectURI,
		"active":           row.Active,
		"intranet":         row.Intranet,
	}, tenantID)
	// Drop taskProjectService's Path A conn cache so branch preview uses the new
	// base_url/provider immediately (OPT-20260827-006).
	go a.invalidateTaskProjectGitLabConnCache(tenantID)
	writeJSON(w, http.StatusOK, a.tenantConnectionPublicFromRow(row))
}

func (a *App) handleTenantGitlabConnectionDelete(w http.ResponseWriter, r *http.Request, tenantID string) {
	if !a.ensureTenantAdmin(w, r, tenantID) {
		return
	}
	providerKey := domain.ProviderKeyForCompany(tenantID)
	if _, err := a.DB.DeleteCredentialsByProviderKey(providerKey); err != nil {
		logWarn("cascade delete credentials: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "cascade delete failed"})
		return
	}
	deleted, err := a.DB.DeleteTenantGitLabConnection(tenantID)
	if err != nil {
		logWarn("tenant gitlab connection delete: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "delete failed"})
		return
	}
	if deleted {
		publishTenantGitLabOAuthConnectionDeleted(a.Cfg, map[string]any{
			"company_id":       tenantID,
			"provider_key":     providerKey,
			"service_provider": domain.TenantServiceProvider(tenantID),
		}, tenantID)
		// Drop taskProjectService's Path A conn cache (OPT-20260827-006).
		go a.invalidateTaskProjectGitLabConnCache(tenantID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleInternalTenantGitlabOAuthConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if companyID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "company_id required"})
		return
	}
	writeJSON(w, http.StatusOK, a.tenantConnectionPublicJSON(companyID))
}

func (a *App) tenantConnectionPublicJSON(companyID string) map[string]any {
	row, err := a.DB.GetTenantGitLabConnection(companyID)
	if err != nil {
		logWarn("get tenant connection: %v", err)
		return a.emptyTenantConnectionPublic(companyID)
	}
	if row == nil {
		return a.emptyTenantConnectionPublic(companyID)
	}
	return a.tenantConnectionPublicFromRow(row)
}

func (a *App) emptyTenantConnectionPublic(companyID string) map[string]any {
	return map[string]any{
		"configured":       false,
		"company_id":       companyID,
		"provider_key":     domain.ProviderKeyForCompany(companyID),
		"service_provider": domain.TenantServiceProvider(companyID),
		"redirect_uri":     domain.DefaultRedirectURI(a.Cfg.PublicBaseURL, companyID),
		"scope":            domain.DefaultTenantGitLabScope,
		"base_url":         "",
		"client_id":        "",
		"remark":           "",
		"active":           false,
		"intranet":         false,
	}
}

func (a *App) tenantConnectionPublicFromRow(row *infrastructure.TenantGitLabOAuthConnectionRow) map[string]any {
	// Always expose the canonical tenant-scoped redirect URI so the settings
	// page copy target stays correct even when DB still holds the legacy shared URI.
	redirectURI := domain.DefaultRedirectURI(a.Cfg.PublicBaseURL, row.CompanyID)
	return map[string]any{
		"configured":       true,
		"id":               row.ID,
		"company_id":       row.CompanyID,
		"base_url":         row.BaseURL,
		"client_id":        row.ClientID,
		"remark":           row.Remark,
		"redirect_uri":     redirectURI,
		"scope":            row.Scope,
		"active":           row.Active,
		"intranet":         row.Intranet,
		"provider_key":     domain.ProviderKeyForCompany(row.CompanyID),
		"service_provider": domain.TenantServiceProvider(row.CompanyID),
		"created_at":       formatUTC(row.CreatedAt),
		"updated_at":       formatUTC(row.UpdatedAt),
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	default:
		return false
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	s := fmt.Sprint(v)
	if s == "<nil>" {
		return ""
	}
	return s
}

func formatUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
