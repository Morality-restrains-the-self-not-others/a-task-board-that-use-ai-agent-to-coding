package main

import (
	"authz"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"tracelog"
)

var tenantHTTP = &http.Client{Timeout: 10 * time.Second}

// resolveUserMemberFn is replaceable in tests.
var resolveUserMemberFn = resolveUserMember

// resolveUserMember resolves user membership via taskTenantService.
// Migrated from Django /api/internal/taskproject/resolve-user-member/ (OPT-20260729-024 #8).
func resolveUserMember(tenantID, userID string) (memberID string, isAdmin bool, err error) {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return "", false, fmt.Errorf("TaskTenantServiceURL not configured")
	}

	url := fmt.Sprintf("%s/api/internal/tenant/members/resolve?user_id=%s&company_id=%s",
		base, userID, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := tenantHTTP.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", false, fmt.Errorf("member not found")
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("resolve-user-member returned status %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false, err
	}
	memberID, _ = out["id"].(string)
	if v, ok := out["is_admin"].(bool); ok {
		isAdmin = v
	}
	return memberID, isAdmin, nil
}

// requireTenantAdmin checks that the authenticated user is a tenant admin.
// Returns true if admin; false and writes 403 response if not.
// v63: 判定迁移至 authz 权限码（billing:manage），网关注入 X-Tenant-Perms。
func requireTenantAdmin(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	return authz.RequirePerm(w, r, authz.PermBillingManage, tenantID)
}
