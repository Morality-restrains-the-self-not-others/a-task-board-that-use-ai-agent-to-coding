package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"authz"
	"taskAuth/domain"
	"tracelog"
)

const tenantGitLabOidcRegion = "settings.gitlab.main"

// handleInternalOidcSsoIdempotencyCleanup 内部维护端点：
// POST /api/internal/taskauth/oidc-sso-idempotency/cleanup/?max_age_days=N&limit=N
// 由 taskEvents timer（oidc_sso_idempotency_cleanup）周期调用，分批删除
// auth_oidc_sso_idempotency 中超过热窗口的行（OPT-20260825-030）。
func handleInternalOidcSsoIdempotencyCleanup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	maxAgeDays := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("max_age_days")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			maxAgeDays = v
		}
	}
	limit := int64(500)
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}
	n, err := cleanupOidcSsoIdempotency(maxAgeDays, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "oidc_sso_idempotency_cleanup",
			"error", err.Error(), "trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusInternalServerError, "cleanup failed")
		return
	}
	slog.InfoContext(r.Context(), "oidc_sso_idempotency_cleanup_ok",
		"deleted", n, "max_age_days", maxAgeDays, "trace_id", tracelog.TraceIDFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": n})
}

func handleTenantGitLabOidcSsoGet(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := gateTenantGitLabOidcSso(w, r, false)
	if !ok {
		return
	}
	_ = userID
	if _, err := loadTenantOidcClient(tenantID); err != nil {
		slog.ErrorContext(r.Context(), "tenant_gitlab_oidc_sso_get_load",
			"company_id", tenantID, "error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, "", false))
}

func handleTenantGitLabOidcSsoPut(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := gateTenantGitLabOidcSso(w, r, true)
	if !ok {
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "Idempotency-Key required")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	reqBase, err := domain.NormalizeBaseURL(strField(body, "base_url"))
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
		return
	}
	pathA, err := fetchPathAConnection(r.Context(), tenantID)
	if err != nil {
		slog.WarnContext(r.Context(), "tenant_gitlab_oidc_sso_path_a_lookup_failed",
			"company_id", tenantID, "trace_id", tracelog.TraceIDFromContext(r.Context()), "error", err.Error())
		writeErrorDetail(w, r, http.StatusBadGateway, "无法校验已保存的 GitLab 连接")
		return
	}
	saved, nerr := domain.NormalizeBaseURL(pathA.BaseURL)
	if !pathA.Configured || nerr != nil || saved != reqBase {
		writeErrorDetail(w, r, http.StatusBadRequest, domain.ErrBaseURLMismatch.Error())
		return
	}
	redirectURI, err := domain.RedirectURIFromBaseURL(reqBase)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := domain.ValidateProductionRedirectURI(redirectURI); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if domain.RedirectURIIsHTTP(redirectURI) {
		host := ""
		if u, perr := url.Parse(redirectURI); perr == nil {
			host = u.Hostname()
		}
		slog.WarnContext(r.Context(), "tenant_gitlab_oidc_sso_http_redirect",
			"company_id", tenantID, "host", host,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
	}
	existing, err := loadTenantOidcClient(tenantID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if existing != nil {
		slog.WarnContext(r.Context(), "idempotency skip",
			"operation", "enable", "company_id", tenantID, "reason", "already_configured",
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, "", false))
		return
	}
	secret, err := generateOidcClientSecret()
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "secret generation failed")
		return
	}
	uris, err := tenantOidcRedirectURIsJSON(redirectURI)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "redirect encode failed")
		return
	}
	clientID, _ := domain.TenantGitLabOidcClientID(tenantID)
	if err := insertTenantOidcClient(tenantID, "Tenant GitLab SSO "+tenantID, uris, secret); err != nil {
		if isDuplicateKey(err) {
			writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, "", false))
			return
		}
		slog.ErrorContext(r.Context(), "tenant_gitlab_oidc_sso_insert", "error", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "insert failed")
		return
	}
	slog.InfoContext(r.Context(), "tenant_gitlab_oidc_sso_enabled",
		"company_id", tenantID, "client_id", clientID, "actor_user_id", userID,
		"trace_id", tracelog.TraceIDFromContext(r.Context()))
	_, _ = claimOidcSsoIdempotency(tenantID, "enable", idem)
	publishTenantGitLabOidcSsoEvent(r.Context(), "TenantGitLabOidcSsoEnabled", tenantID, clientID, userID, redirectURI)
	writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, secret, true))
}

func handleTenantGitLabOidcSsoRotate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := gateTenantGitLabOidcSso(w, r, true)
	if !ok {
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "Idempotency-Key required")
		return
	}
	existing, err := loadTenantOidcClient(tenantID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if existing == nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "sso not configured")
		return
	}
	already, err := claimOidcSsoIdempotency(tenantID, "rotate", idem)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "idempotency failed")
		return
	}
	if already {
		slog.WarnContext(r.Context(), "idempotency skip",
			"operation", "rotate", "company_id", tenantID, "client_id", existing.ClientID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, "", false))
		return
	}
	secret, err := generateOidcClientSecret()
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "secret generation failed")
		return
	}
	if err := updateTenantOidcClientSecretAndRedirect(existing.ClientID, secret, existing.RedirectURIs); err != nil {
		slog.ErrorContext(r.Context(), "tenant_gitlab_oidc_sso_rotate",
			"client_id", existing.ClientID, "error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusInternalServerError, "rotate failed")
		return
	}
	slog.InfoContext(r.Context(), "tenant_gitlab_oidc_sso_rotated",
		"company_id", tenantID, "client_id", existing.ClientID, "actor_user_id", userID,
		"trace_id", tracelog.TraceIDFromContext(r.Context()))
	publishTenantGitLabOidcSsoEvent(r.Context(), "TenantGitLabOidcSsoSecretRotated", tenantID, existing.ClientID, userID, "")
	writeJSON(w, http.StatusOK, tenantGitLabOidcSsoPublic(r, tenantID, secret, true))
}

func handleTenantGitLabOidcSsoDelete(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := gateTenantGitLabOidcSso(w, r, true)
	if !ok {
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "Idempotency-Key required")
		return
	}
	already, err := claimOidcSsoIdempotency(tenantID, "disable", idem)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "idempotency failed")
		return
	}
	clientID, _ := domain.TenantGitLabOidcClientID(tenantID)
	if already {
		slog.WarnContext(r.Context(), "idempotency skip",
			"operation", "disable", "company_id", tenantID, "client_id", clientID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
	}
	if !already {
		if err := deleteTenantOidcClient(clientID); err != nil {
			slog.ErrorContext(r.Context(), "tenant_gitlab_oidc_sso_delete",
				"client_id", clientID, "error", err.Error(),
				"trace_id", tracelog.TraceIDFromContext(r.Context()))
			writeErrorDetail(w, r, http.StatusInternalServerError, "delete failed")
			return
		}
		slog.InfoContext(r.Context(), "tenant_gitlab_oidc_sso_disabled",
			"company_id", tenantID, "client_id", clientID, "actor_user_id", userID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		publishTenantGitLabOidcSsoEvent(r.Context(), "TenantGitLabOidcSsoDisabled", tenantID, clientID, userID, "")
	}
	w.WriteHeader(http.StatusNoContent)
}

func gateTenantGitLabOidcSso(w http.ResponseWriter, r *http.Request, write bool) (tenantID, userID string, ok bool) {
	userID, authed := requireAuthenticatedUser(w, r)
	if !authed {
		return "", "", false
	}
	tenantID = strings.TrimSpace(r.PathValue("tenant_id"))
	if tenantID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "tenant_id required")
		return "", "", false
	}
	if write {
		if !authz.RequireRegionOperate(w, r, tenantGitLabOidcRegion, tenantID) {
			return "", "", false
		}
	} else if !authz.RequireRegionView(w, r, tenantGitLabOidcRegion, tenantID) {
		return "", "", false
	}
	memberID, isAdmin, err := resolveTenantMember(r.Context(), tenantID, userID)
	if err != nil {
		slog.WarnContext(r.Context(), "tenant_gitlab_oidc_sso_member_lookup_failed",
			"company_id", tenantID, "user_id", userID, "error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusForbidden, "无权访问该租户资源")
		return "", "", false
	}
	if memberID == "" {
		slog.WarnContext(r.Context(), "tenant_gitlab_oidc_sso_forbidden",
			"reason", "not_member", "company_id", tenantID, "user_id", userID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusForbidden, "无权访问该租户资源")
		return "", "", false
	}
	if write && !isAdmin {
		slog.WarnContext(r.Context(), "tenant_gitlab_oidc_sso_forbidden",
			"reason", "not_admin", "company_id", tenantID, "user_id", userID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusForbidden, "需要租户管理员权限")
		return "", "", false
	}
	return tenantID, userID, true
}

func tenantGitLabOidcSsoPublic(r *http.Request, tenantID, secret string, includeSecret bool) map[string]any {
	clientID, _ := domain.TenantGitLabOidcClientID(tenantID)
	iss := strings.TrimRight(issuerURL(), "/")
	row, _ := loadTenantOidcClient(tenantID)
	configured := row != nil
	redirectURI := firstRedirectURI(row)
	if redirectURI == "" {
		if pathA, err := fetchPathAConnection(r.Context(), tenantID); err == nil && pathA.Configured {
			if u, err := domain.RedirectURIFromBaseURL(pathA.BaseURL); err == nil {
				redirectURI = u
			}
		}
	}
	out := map[string]any{
		"configured":       configured,
		"company_id":       tenantID,
		"client_id":        clientID,
		"issuer":           iss,
		"discovery_url":    iss + "/.well-known/openid-configuration",
		"redirect_uri":     redirectURI,
		"omniauth_snippet": domain.OmniAuthSnippet(iss, clientID, redirectURI, tenantID),
	}
	if includeSecret && secret != "" {
		out["client_secret"] = secret
	}
	return out
}

func enforceTenantGitLabOidcAuthorize(w http.ResponseWriter, r *http.Request, sendError func(code, desc string), client *oidcClientRow, userID string) bool {
	if client == nil || !domain.IsTenantGitLabOidcClient(client.ClientID, client.ManagedBy) {
		return true
	}
	owner := ownerCompanyIDOfClient(client)
	memberID, _, err := resolveTenantMember(r.Context(), owner, userID)
	decideErr := domain.DecideAuthorizeMembership(domain.AuthorizeMembershipInput{
		UserID:          userID,
		OwnerCompanyID:  owner,
		IsMember:        memberID != "",
		MembershipError: err,
	})
	if decideErr != nil {
		slog.WarnContext(r.Context(), "oidc_tenant_sso_access_denied",
			"client_id", client.ClientID, "user_id", userID, "company_id", owner,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
			"member_lookup_error", err != nil)
		sendError("access_denied", "not a member of this tenant")
		return false
	}
	return true
}
