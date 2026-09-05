package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type cloudServerConfig struct {
	CompanyID           string `json:"company_id"`
	WorkspaceID         string `json:"workspace_id"`
	TaskID              string `json:"task_id"`
	CommentID           string `json:"comment_id"`
	ServerURL           string `json:"server_url"`
	BusinessAPIEndpoint string `json:"business_api_endpoint"`
}

func lookupCloudServerConfig(ctx context.Context, tenantID, workspaceID, taskID string) (*cloudServerConfig, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	q := url.Values{}
	q.Set("tenant_id", tenantID)
	q.Set("workspace_id", workspaceID)
	q.Set("task_id", taskID)
	reqURL := cloudServiceBase() + "/api/internal/cloud-server-config/lookup/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud lookup status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out cloudServerConfig
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
