package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

var tenantHTTP = &http.Client{Timeout: 10 * time.Second}

func tenantBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskTenantURL), "/")
	if base == "" {
		return "http://127.0.0.1:8020"
	}
	return base
}

func tenantGET(ctx context.Context, path string, query url.Values) ([]byte, int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u := tenantBaseURL() + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	start := time.Now()
	resp, err := tenantHTTP.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		tracelog.LogForwardStage(ctx, "tenant_internal", map[string]any{
			"tenant_status": 502,
			"duration_ms":   duration,
			"path":          path,
			"detail":        "taskTenantService unreachable",
		})
		return nil, 502, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	tracelog.LogForwardStage(ctx, "tenant_internal", map[string]any{
		"tenant_status": resp.StatusCode,
		"duration_ms":   duration,
		"path":          path,
	})
	return raw, resp.StatusCode, nil
}

// verifyCompanyExists checks tenant_company SSOT in taskTenantService.
// Django /api/internal/companies/{id}/exists/ was removed when company rows moved to taskTenantService.
func verifyCompanyExists(companyID string) error {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return fmt.Errorf("company %s not found", companyID)
	}
	if strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return nil
	}
	q := url.Values{}
	q.Set("company_id", companyID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/companies/creator", q)
	if err != nil {
		return fmt.Errorf("company verification failed: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("company %s not found (status %d)", companyID, status)
	}
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if found, ok := out["found"].(bool); ok && !found {
		return fmt.Errorf("company %s not found", companyID)
	}
	return nil
}

// tenantUserInGroup checks group membership via taskTenantService.
// Replaces Django /api/internal/taskproject/user-in-group/ (OPT-20260729-024 #10).
func tenantUserInGroup(groupID, userID string) (bool, error) {
	q := url.Values{}
	q.Set("user_id", userID)
	q.Set("group_id", groupID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/groups/user-in-group", q)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("user-in-group: status %d", status)
	}
	var result struct {
		InGroup bool `json:"in_group"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, nil
	}
	return result.InGroup, nil
}

// tenantIsCompanyCreator checks if userID is the creator of tenantID.
// Replaces Django /api/internal/taskproject/resolve-user-member/ is_creator field (OPT-20260729-024 #8).
func tenantIsCompanyCreator(tenantID, userID string) (bool, error) {
	q := url.Values{}
	q.Set("company_id", tenantID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/companies/creator", q)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("company-creator: status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, err
	}
	creatorID := strings.TrimSpace(strField(out, "creator_id"))
	return creatorID != "" && creatorID == userID, nil
}
