package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type routeInfo struct {
	Provider      string `json:"provider"`
	UpstreamBase  string `json:"upstream_base_url"`
	BudgetEnabled bool   `json:"budget_enabled"`
	UseSubToken   bool   `json:"use_sub_token"`
}

type upstreamCredential struct {
	Provider     string `json:"provider"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	DeriveNeeded bool   `json:"derive_needed"`
}

type budgetDecision struct {
	Allowed     bool   `json:"allowed"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
	SpentAmount string `json:"spent_amount,omitempty"`
	BudgetLimit string `json:"budget_limit,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

type commitUsageRequest struct {
	TenantID       string `json:"tenant_id"`
	WorkspaceID    string `json:"workspace_id"`
	TaskID         string `json:"task_id"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	ModelName      string `json:"model_name"`
	InputTokens    int    `json:"input_tokens"`
	OutputTokens   int    `json:"output_tokens"`
	IdempotencyKey string `json:"idempotency_key"`
	RequestID      string `json:"request_id"`
}

type credentialValidateResponse struct {
	Valid       bool   `json:"valid"`
	CompanyID   string `json:"company_id"`
	WorkspaceID string `json:"workspace_id"`
	TaskID      string `json:"task_id"`
	Detail      string `json:"detail,omitempty"`
}

func cloudPost(ctx context.Context, path string, payload any) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.CloudServiceURL), "/")
	if base == "" {
		return http.StatusBadGateway, []byte(`{"detail":"taskCloudService url unset"}`), nil
	}
	url := base + path
	timeout := time.Duration(cfg.UpstreamTimeoutSec * float64(time.Second))
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cfg.CloudInternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.CloudInternalSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"taskCloudService unreachable"}`), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"failed to read taskCloudService response"}`), nil
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	return resp.StatusCode, body, nil
}

func credentialPost(ctx context.Context, path string, payload any) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.CredentialServiceURL), "/")
	if base == "" {
		return http.StatusBadGateway, []byte(`{"detail":"taskCredentialService url unset"}`), nil
	}
	url := base + path
	timeout := time.Duration(cfg.UpstreamTimeoutSec * float64(time.Second))
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"taskCredentialService unreachable"}`), nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"failed to read credential response"}`), nil
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	return resp.StatusCode, body, nil
}

// validateProxyToken calls taskCredentialService /v1/token/validate then compares scope.
func validateProxyToken(ctx context.Context, tenantID, workspaceID, taskID, token string) (bool, string) {
	status, body, _ := credentialPost(ctx, "/v1/token/validate", map[string]string{
		"access_token": token,
	})
	if status != http.StatusOK {
		return false, "invalid proxy token"
	}
	var out credentialValidateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return false, "invalid proxy token response"
	}
	if !out.Valid {
		if out.Detail != "" {
			return false, out.Detail
		}
		return false, "invalid proxy token"
	}
	if strings.TrimSpace(out.CompanyID) != strings.TrimSpace(tenantID) ||
		strings.TrimSpace(out.WorkspaceID) != strings.TrimSpace(workspaceID) ||
		strings.TrimSpace(out.TaskID) != strings.TrimSpace(taskID) {
		return false, "token scope mismatch"
	}
	return true, ""
}

func cloudResolveRoute(ctx context.Context, tenantID, workspaceID, taskID, provider string) (*routeInfo, int, []byte) {
	status, body, _ := cloudPost(ctx, "/api/internal/ai-endpoint/resolve-route/", map[string]string{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
		"provider":     provider,
	})
	if status != http.StatusOK {
		return nil, status, body
	}
	var out routeInfo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, http.StatusBadGateway, []byte(`{"detail":"invalid route response"}`)
	}
	return &out, status, body
}

func cloudUpstreamCredential(ctx context.Context, tenantID, provider, baseURL string) (*upstreamCredential, int, []byte) {
	status, body, _ := cloudPost(ctx, "/api/internal/ai-endpoint/upstream-credentials/", map[string]string{
		"tenant_id": tenantID,
		"provider":  provider,
		"base_url":  baseURL,
	})
	if status != http.StatusOK {
		return nil, status, body
	}
	var out upstreamCredential
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, http.StatusBadGateway, []byte(`{"detail":"invalid credential response"}`)
	}
	return &out, status, body
}

func cloudBudgetGate(ctx context.Context, tenantID, workspaceID, taskID, provider, baseURL, modelName string) (*budgetDecision, int, []byte) {
	status, body, _ := cloudPost(ctx, "/api/internal/budget/reserve-or-deny/", map[string]string{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
		"provider":     provider,
		"base_url":     baseURL,
		"model_name":   modelName,
	})
	if status != http.StatusOK {
		var dec budgetDecision
		_ = json.Unmarshal(body, &dec)
		if dec.Code != "" {
			return &dec, status, body
		}
		return nil, status, body
	}
	var dec budgetDecision
	if err := json.Unmarshal(body, &dec); err != nil {
		return nil, http.StatusBadGateway, body
	}
	return &dec, status, body
}

func cloudCommitUsage(ctx context.Context, req commitUsageRequest) {
	_, _, _ = cloudPost(ctx, "/api/internal/budget/record-usage/", req)
}

func extractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if strings.HasPrefix(authHeader, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	}
	return authHeader
}

func budgetExhaustedPayload(dec *budgetDecision) []byte {
	payload := map[string]any{
		"error": map[string]any{
			"code":    "budget_exhausted",
			"message": dec.Message,
		},
	}
	if dec.SpentAmount != "" {
		payload["error"].(map[string]any)["spent_amount"] = dec.SpentAmount
	}
	if dec.BudgetLimit != "" {
		payload["error"].(map[string]any)["budget_limit"] = dec.BudgetLimit
	}
	if dec.Currency != "" {
		payload["error"].(map[string]any)["currency"] = dec.Currency
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func fmtUpstreamURL(baseURL, subPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if subPath == "" {
		return base
	}
	return fmt.Sprintf("%s/%s", base, strings.TrimLeft(subPath, "/"))
}

// ensureUpstreamV1Base appends /v1 only when missing. Feature-params often store
// upstream as https://api.deepseek.com/v1; naively appending /v1 again yields
// .../v1/v1/chat/completions → permanent upstream 404.
func ensureUpstreamV1Base(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return base
	}
	if strings.HasSuffix(strings.ToLower(base), "/v1") {
		return base
	}
	return base + "/v1"
}
