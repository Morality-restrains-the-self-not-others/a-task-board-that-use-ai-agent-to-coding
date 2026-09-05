package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gatewayauth"
)

// isInternalCall returns true when the request originates from a trusted
// internal service (X-Auth-User-Id == "internal").
func isInternalCall(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get("X-Auth-User-Id")) == "internal"
}

// requireTenantMember verifies that the user is a member of the given tenant
// by calling taskTenantService's internal resolve endpoint. Internal service-
// to-service calls bypass this check. When TaskTenantServiceURL is not configured
// (e.g. test/dev environments), the check is skipped.
// Returns true if the user is a member; writes 401/403/502 and returns false otherwise.
func requireTenantMember(w http.ResponseWriter, r *http.Request, userID, tenantID string) bool {
	if isInternalCall(r) {
		return true
	}
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "请先登录")
		return false
	}
	// Fast path: gateway-embedded membership JWT (avoids HTTP round-trip)
	if claims, ok := gatewayauth.ParseTenantClaims(r, cfg.TaskGatewayInternalSecret); ok {
		if claims.Subject == userID && gatewayauth.IsTenantMember(claims, tenantID) {
			return true
		}
	}

	// Skip membership check when tenant service URL is not configured (tests / dev)
	if strings.TrimSpace(cfg.TaskTenantServiceURL) == "" {
		return true
	}
	m, err := tenantResolveMember(tenantID, userID)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "租户成员校验失败")
		return false
	}
	if m == nil {
		writeError(w, r, http.StatusForbidden, "您不是该公司的成员")
		return false
	}
	return true
}

// tenantResolveMember calls taskTenantService to check if userID is a member
// of the given company (tenantID). Returns the member object or nil if not found.
func tenantResolveMember(tenantID, userID string) (map[string]interface{}, error) {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return nil, fmt.Errorf("task tenant service URL not configured")
	}
	url := fmt.Sprintf("%s/api/internal/tenant/members/resolve?user_id=%s&company_id=%s",
		base, userID, tenantID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tenant resolve member status %d: %s", resp.StatusCode, string(raw))
	}

	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// tenantListMembers fetches all company members from taskTenantService.
// Used by batchResolveTaskOwners (OPT-20260729-024 #9).
func tenantListMembers(tenantID string) ([]map[string]interface{}, error) {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return nil, fmt.Errorf("task tenant service URL not configured")
	}
	url := fmt.Sprintf("%s/api/internal/tenant/members?company_id=%s", base, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenant list members status %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// verifyCompanyViaTenantService checks company existence via taskTenantService.
// Replaces Django /api/internal/companies/{id}/exists/.
func verifyCompanyViaTenantService(companyID string) error {
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		return nil // permissive in dev
	}
	url := fmt.Sprintf("%s/api/internal/tenant/companies/creator?company_id=%s", base, companyID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("company verification failed: %w", err)
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("company verification failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("company %s not found (status %d)", companyID, resp.StatusCode)
	}
	return nil
}
