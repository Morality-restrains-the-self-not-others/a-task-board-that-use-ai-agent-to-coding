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

type pathAConnection struct {
	Configured bool   `json:"configured"`
	BaseURL    string `json:"base_url"`
	Active     bool   `json:"active"`
}

func resolveTenantMember(ctx context.Context, companyID, userID string) (memberID string, isAdmin bool, err error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TenantServiceURL), "/")
	if base == "" {
		return "", false, fmt.Errorf("tenant service url empty")
	}
	u := fmt.Sprintf("%s/api/internal/tenant/members/resolve?user_id=%s&company_id=%s",
		base, url.QueryEscape(userID), url.QueryEscape(companyID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("members/resolve status %d", resp.StatusCode)
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

func fetchPathAConnection(ctx context.Context, companyID string) (pathAConnection, error) {
	var empty pathAConnection
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOauthServiceURL), "/")
	if base == "" {
		return empty, fmt.Errorf("git oauth service url empty")
	}
	u := fmt.Sprintf("%s/api/internal/git-oauth/gitlab-tenant-connection/?company_id=%s",
		base, url.QueryEscape(companyID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return empty, err
	}
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(cfg.GitOauthBridgeSecret); sec != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("gitlab-tenant-connection status %d", resp.StatusCode)
	}
	var out pathAConnection
	if err := json.Unmarshal(raw, &out); err != nil {
		return empty, err
	}
	return out, nil
}
