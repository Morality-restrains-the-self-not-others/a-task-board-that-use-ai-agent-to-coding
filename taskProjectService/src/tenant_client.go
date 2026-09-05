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
)

var tenantHTTP = &http.Client{Timeout: 10 * time.Second}

func tenantBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskTenantURL), "/")
	if base == "" {
		return "http://127.0.0.1:8020"
	}
	return base
}

func tenantHeaders() map[string]string {
	h := map[string]string{}
	if cfg.InternalSecret != "" {
		h["X-Internal-Secret"] = cfg.InternalSecret
	}
	return h
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
	for k, v := range tenantHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := tenantHTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func tenantResolveMember(tenantID, userID string) (map[string]interface{}, error) {
	q := url.Values{}
	q.Set("user_id", userID)
	q.Set("company_id", tenantID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/members/resolve", q)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("tenant resolve member status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func tenantListMembers(tenantID string) ([]map[string]interface{}, error) {
	q := url.Values{}
	q.Set("company_id", tenantID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/members", q)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("tenant list members status %d", status)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	return list, nil
}

func tenantGroupMemberUserIDs(groupID string) ([]string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("group_id", groupID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/groups/members", q)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("tenant group members status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	rawIDs, _ := out["user_ids"].([]interface{})
	ids := make([]string, 0, len(rawIDs))
	for _, v := range rawIDs {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" && s != "<nil>" {
			ids = append(ids, s)
		}
	}
	return ids, nil
}

func tenantGetGroup(groupID, tenantID string) (map[string]interface{}, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("group_id", groupID)
	if tenantID != "" {
		q.Set("company_id", tenantID)
	}
	raw, status, err := tenantGET(nil, "/api/internal/tenant/groups/by-id", q)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("tenant get group status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// tenantCompanyCreatorID reads creator_id from taskTenantService (local tenant_company SSOT).
func tenantCompanyCreatorID(tenantID string) (string, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "", nil
	}
	q := url.Values{}
	q.Set("company_id", tenantID)
	raw, status, err := tenantGET(nil, "/api/internal/tenant/companies/creator", q)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		return "", nil
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("tenant company-creator status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if found, ok := out["found"].(bool); ok && !found {
		return "", nil
	}
	return strings.TrimSpace(strField(out, "creator_id")), nil
}

// tenantUpdateMemberWorkspace calls PATCH /api/internal/tenant/members/workspace.
// Replaces Django /api/internal/taskproject/persist-workspace-selection/ (OPT-20260729-024 #1).
func tenantUpdateMemberWorkspace(userID, tenantID, workspaceID string) error {
	body, _ := json.Marshal(map[string]string{
		"user_id":      userID,
		"company_id":   tenantID,
		"workspace_id": workspaceID,
	})
	req, err := http.NewRequest(http.MethodPatch, tenantBaseURL()+"/api/internal/tenant/members/workspace", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range tenantHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := tenantHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tenant update workspace status %d", resp.StatusCode)
	}
	return nil
}

func tenantGetMemberByID(memberID, tenantID string) (map[string]interface{}, error) {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("member_id", memberID)
	if tenantID != "" {
		q.Set("company_id", tenantID)
	}
	raw, status, err := tenantGET(nil, "/api/internal/tenant/members/by-id", q)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("tenant member by-id status %d", status)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
