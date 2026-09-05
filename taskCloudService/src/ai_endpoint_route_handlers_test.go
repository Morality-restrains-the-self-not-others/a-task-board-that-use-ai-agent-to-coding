package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveProviderRouteAndUpstreamCredentials(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"deepseek","api_key":"sk-master","base_url":"https://api.deepseek.com/v1","supported_models":["m1"],"use_sub_token":true,"budget_enabled":true},{"provider":"openai","api_key":"sk-o","base_url":"https://api.openai.com/v1","use_sub_token":false}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "c1", ProvidersJSON: providers, LLMBudgetEnabled: true,
		AgentModel: "m1", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})

	route, err := resolveProviderRoute("c1", "deepseek")
	if err != nil {
		t.Fatal(err)
	}
	if route == nil || route["upstream_base_url"] != "https://api.deepseek.com/v1" {
		t.Fatalf("route=%v", route)
	}
	if route["budget_enabled"] != true || route["use_sub_token"] != true {
		t.Fatalf("flags=%v", route)
	}
	noRoute, err := resolveProviderRoute("c1", "openai")
	if err != nil {
		t.Fatal(err)
	}
	if noRoute != nil {
		t.Fatalf("openai without use_sub_token should be nil, got %v", noRoute)
	}

	cred, err := upstreamCredentials("c1", "deepseek", "https://api.deepseek.com/v1/")
	if err != nil {
		t.Fatal(err)
	}
	if cred == nil || cred["api_key"] != "sk-master" {
		t.Fatalf("cred=%v", cred)
	}
	if cred["derive_needed"] != true {
		t.Fatalf("derive=%v", cred["derive_needed"])
	}
}

func TestInternalAIEndpointResolveRouteHTTP(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"deepseek","api_key":"sk","base_url":"https://api.deepseek.com/v1","use_sub_token":true,"budget_enabled":false}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "c1", ProvidersJSON: providers,
		AgentModel: "m1", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	prev := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = prev })

	body := `{"tenant_id":"c1","workspace_id":"w","task_id":"t","provider":"deepseek"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/ai-endpoint/resolve-route/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalAIEndpointRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["provider"] != "deepseek" {
		t.Fatalf("out=%v", out)
	}
}
