package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleModelBudgetUsageWritesLocalLedger(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"openai","api_key":"sk","base_url":"https://api.openai.com/v1","supported_models":["gpt"],"use_sub_token":true,"budget_enabled":true}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "t1", ProvidersJSON: providers, LLMBudgetEnabled: true,
		AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})

	rec := httptest.NewRecorder()
	handleModelBudgetUsage(
		rec,
		nil,
		map[string]any{
			"items": []any{
				map[string]any{
					"provider":            "openai",
					"base_url":            "https://api.openai.com/v1",
					"model_name":          "gpt",
					"idempotency_key":     "k1",
					"input_tokens_delta":  float64(10),
					"output_tokens_delta": float64(0),
				},
			},
		},
		"t1", "w1", "task1",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"currency":"CNY"`) {
		t.Fatalf("response=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"created":true`) {
		t.Fatalf("response=%s", rec.Body.String())
	}
}

func TestHandleModelBudgetUsageRejectsBadItems(t *testing.T) {
	rec := httptest.NewRecorder()
	handleModelBudgetUsage(rec, nil, map[string]any{"items": "nope"}, "t", "w", "x")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandleFeatureParamsEnvPrefersGoResolve(t *testing.T) {
	setupBudgetTestDB(t)
	seedCompanyFP(t, "t1", "m1")
	prevBind := fetchTaskFeatureParamsBindingFn
	fetchTaskFeatureParamsBindingFn = func(tenantID, taskID string) (*taskFeatureParamsBinding, error) {
		return &taskFeatureParamsBinding{Source: "company", WorkspaceID: "w1", TenantID: tenantID}, nil
	}
	t.Cleanup(func() { fetchTaskFeatureParamsBindingFn = prevBind })
	prevAllow := fetchWorkspaceAllowPersonalFn
	fetchWorkspaceAllowPersonalFn = func(ctx context.Context, workspaceID string) (bool, error) { return false, nil }
	t.Cleanup(func() { fetchWorkspaceAllowPersonalFn = prevAllow })

	rec := httptest.NewRecorder()
	handleFeatureParamsEnv(
		rec, nil,
		&CloudServerConfig{CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		map[string]any{},
		"t1", "w1", "task1", "tok-proxy",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"TASK_AGENT_MODEL":"m1"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleCloudTaskRoutesModelBudgetAndRelay(t *testing.T) {
	prevCred := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = ""
	defer func() { cfg.CredentialServiceURL = prevCred }()

	for _, path := range []string{
		"/api/tenant/t1/workspace/w1/task/task1/comment/cmt1/cloud/model-budget-usage/",
		"/api/tenant/t1/workspace/w1/task/task1/comment/cmt1/cloud/relay-to-trae/status-push/",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"access_token":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		sub := strings.TrimPrefix(path, "/api/tenant/t1/workspace/w1/task/task1/comment/cmt1/cloud/")
		handleCloudTaskRoutes(rec, req, sub)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("path %s not routed", path)
		}
	}
}
