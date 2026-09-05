package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"authz"
	"gatewayauth"
)

func effectiveUserID(r *http.Request) string {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)
	if uid := getAuthUser(r); uid != "" {
		return uid
	}
	return strings.TrimSpace(r.Header.Get(gatewayauth.HeaderUserID))
}

func ensureTenantMember(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	userID := effectiveUserID(r)
	if userID == "" {
		writeErrorMapJSON(w, r, 401, map[string]interface{}{"status": "error", "message": "未认证"})
		return false
	}
	// 平台超管（super_admin，网关 forward-auth 注入 X-User-Roles）可访问任意租户资源：
	// 与 taskProjectService 项目 CRUD / 系统管理页的既有授权模型一致，
	// 避免 bootstrap-admin 在租户项目页（运行模版/地域列表等）被 403 阻断。
	if authz.AuthContextFromHeaders(r).HasPlatformRole("super_admin") {
		return true
	}
	if cfg.TaskTenantURL == "" {
		return true
	}
	memberID, _, err := resolveUserMember(r, tenantID, userID)
	if err != nil || memberID == "" {
		writeErrorMapJSON(w, r, 403, map[string]interface{}{"status": "error", "message": "无权访问该租户资源"})
		return false
	}
	return true
}

// resolveUserMember maps user_id → company_member_id + is_admin.
// Fast path: reads the X-Auth-Tenant-Claims JWT (set by gateway forward-auth),
// avoiding the HTTP round-trip to taskTenantService.
// Fallback: calls resolveMemberViaTenantService when the header is absent/invalid.
// Django fallback removed — taskTenantService is the SSOT (OPT-20260729-024 #8).
func resolveUserMember(r *http.Request, tenantID, userID string) (memberID string, isAdmin bool, err error) {
	// Fast path: gateway-embedded membership JWT
	if claims, ok := gatewayauth.ParseTenantClaims(r, cfg.GatewayInternalSecret); ok {
		if claims.Subject == userID {
			for _, m := range claims.Members {
				if m.TenantID == tenantID {
					return m.MemberID, m.IsAdmin, nil
				}
			}
			// User is known to have NO membership in this tenant
			return "", false, errMemberNotFound
		}
	}
	// Fallback: direct HTTP call to taskTenantService
	return resolveMemberViaTenantService(tenantID, userID)
}

// resolveMemberViaTenantService calls taskTenantService /api/internal/tenant/members/resolve
// as the SSOT for company membership (OPT-20260720-009).
func resolveMemberViaTenantService(tenantID, userID string) (string, bool, error) {
	if cfg.TaskTenantURL == "" {
		return "", false, errMemberNotFound
	}
	q := url.Values{}
	q.Set("company_id", tenantID)
	q.Set("user_id", userID)
	body, status, err := tenantGET(nil, "/api/internal/tenant/members/resolve", q)
	if err != nil || status != 200 {
		return "", false, errMemberNotFound
	}
	var out map[string]interface{}
	if json.Unmarshal(body, &out) != nil {
		return "", false, errMemberNotFound
	}
	memberID := strField(out, "id")
	if memberID == "" {
		memberID = strField(out, "company_member_id")
	}
	isAdmin := false
	if v, ok := out["is_admin"].(bool); ok {
		isAdmin = v
	}
	return memberID, isAdmin, nil
}

// handleInternalTenantMember resolves membership via taskTenantService.
// GET /api/internal/tenant-member/?tenant_id=&user_id=
func handleInternalTenantMember(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if tenantID == "" || userID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id and user_id required")
		return
	}
	memberID, isAdmin, err := resolveUserMember(r, tenantID, userID)
	if err != nil || memberID == "" {
		writeErrorJSON(w, r, http.StatusNotFound, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"company_member_id": memberID,
		"user_id":           userID,
		"tenant_id":         tenantID,
		"is_admin":          isAdmin,
	})
}

var errMemberNotFound = &memberNotFoundError{}

type memberNotFoundError struct{}

func (e *memberNotFoundError) Error() string { return "member not found" }

var tenantPathFromReferer = regexp.MustCompile(`/tenant/([^/]+)/`)

// resolveGlobalCloudTenantID 解析无租户段的全局 cloud API 所属租户（Referer / query / 网关头）。
func resolveGlobalCloudTenantID(r *http.Request) string {
	if tid := strings.TrimSpace(getAuthTenant(r)); tid != "" {
		return tid
	}
	if tid := strings.TrimSpace(r.URL.Query().Get("tenant_id")); tid != "" {
		return tid
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		if m := tenantPathFromReferer.FindStringSubmatch(referer); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
