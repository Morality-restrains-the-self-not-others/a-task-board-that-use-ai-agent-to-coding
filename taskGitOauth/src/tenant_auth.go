package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gatewayauth"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func (a *App) effectiveUserID(r *http.Request) string {
	if uid := strings.TrimSpace(r.Header.Get("X-Auth-User-Id")); uid != "" {
		return uid
	}
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func (a *App) ensureTenantMember(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	userID := a.effectiveUserID(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return false
	}
	if a.Cfg.TenantAuthBypass {
		return true
	}
	// Fast path: gateway-embedded membership JWT
	if claims, ok := gatewayauth.ParseTenantClaims(r, a.Cfg.GatewayInternalSecret); ok {
		if claims.Subject == userID && gatewayauth.IsTenantMember(claims, tenantID) {
			return true
		}
	}
	memberID, _, err := a.resolveUserMember(tenantID, userID)
	if err != nil || memberID == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "无权访问该租户资源"})
		return false
	}
	return true
}

func (a *App) ensureTenantAdmin(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	if !a.ensureTenantMember(w, r, tenantID) {
		return false
	}
	if a.Cfg.TenantAuthBypass {
		if strings.TrimSpace(r.Header.Get("X-Is-Admin")) == "1" {
			return true
		}
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "需要租户管理员权限"})
		return false
	}
	// Fast path: gateway-embedded membership JWT
	uid := a.effectiveUserID(r)
	if claims, ok := gatewayauth.ParseTenantClaims(r, a.Cfg.GatewayInternalSecret); ok {
		if claims.Subject == uid && gatewayauth.IsTenantAdmin(claims, tenantID) {
			return true
		}
	}
	_, isAdmin, err := a.resolveUserMember(tenantID, uid)
	if err != nil || !isAdmin {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "需要租户管理员权限"})
		return false
	}
	return true
}

// resolveUserMember maps user_id → company_member_id + is_admin via taskTenantService.
// Migrated from Django /api/internal/taskproject/resolve-user-member/ (OPT-20260729-024 #8).
func (a *App) resolveUserMember(tenantID, userID string) (memberID string, isAdmin bool, err error) {
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskTenantServiceURL), "/")
	if base == "" {
		return "", false, fmt.Errorf("TaskTenantServiceURL not configured")
	}

	url := fmt.Sprintf("%s/api/internal/tenant/members/resolve?user_id=%s&company_id=%s",
		base, userID, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(a.Cfg.DjangoInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return "", false, fmt.Errorf("member not found")
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("resolve-user-member status %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false, err
	}
	memberID = strings.TrimSpace(fmt.Sprint(out["id"]))
	if memberID == "<nil>" {
		memberID = ""
	}
	if v, ok := out["is_admin"].(bool); ok {
		isAdmin = v
	}
	return memberID, isAdmin, nil
}

func (a *App) resolveProviderByServiceProvider(sp string) (*infrastructure.ProviderConfig, error) {
	return a.Cfg.ResolveByServiceProviderWithDB(a.DB, a.Fernet, sp)
}

func (a *App) resolveGitLabAuthorizeContext(sp, allowedHost string) (map[string]string, string) {
	var tenantPC *infrastructure.ProviderConfig
	if companyID, ok := domain.ParseTenantCompanyID(sp); ok {
		row, err := a.DB.GetTenantGitLabConnection(companyID)
		if err == nil && row != nil && row.Active {
			tenantPC, _ = infrastructure.TenantRowToProviderConfig(row, a.Fernet)
		}
	}
	return infrastructure.ResolveGitLabAuthorizeContext(a.Cfg, sp, allowedHost, tenantPC)
}
