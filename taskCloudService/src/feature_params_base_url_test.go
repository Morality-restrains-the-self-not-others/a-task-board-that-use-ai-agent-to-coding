package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gatewayauth"
)

func TestProvidersFromJSONStringRepairsConcatenatedBaseURL(t *testing.T) {
	got := providersFromJSONString(
		`[{"provider":"deepseek","api_key":"sk","base_url":"https://api.deepseek.com/v1https://api.deepseek.com"}]`,
	)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0]["base_url"] != "https://api.deepseek.com" {
		t.Fatalf("base_url=%v", got[0]["base_url"])
	}
}

func TestCompanyFeatureParamsGetRepairsConcatenatedBaseURL(t *testing.T) {
	store := setupFeatureParamsPublicTest(t)
	store.putTenant(&tenantFeatureParamsRow{
		ID: "tp1", CompanyID: "c1",
		ProvidersJSON:    `[{"provider":"deepseek","api_key":"sk","base_url":"https://api.deepseek.com/v1https://api.deepseek.com","supported_models":["deepseek-chat"],"use_sub_token":false,"budget_enabled":false}]`,
		LLMBudgetEnabled: false,
		AgentModel:       "deepseek-chat", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/feature-params/tenant_id/c1/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	req.Header.Set(headerFeatureParamsAccessContext, fpContextCompanySettings)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	providers, _ := data["providers"].([]any)
	p0, _ := providers[0].(map[string]any)
	if p0["base_url"] != "https://api.deepseek.com" {
		t.Fatalf("GET base_url=%v", p0["base_url"])
	}
}

func TestCompanyFeatureParamsPostKeepsOperatorTypedBaseURL(t *testing.T) {
	setupFeatureParamsPublicTest(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{
		"providers": [{
			"provider": "deepseek",
			"api_key": "sk",
			"base_url": "https://gateway.example.com/anthropic",
			"supported_models": ["deepseek-chat"]
		}],
		"agent_model": "deepseek-chat",
		"agent_model_provider": "deepseek",
		"agent_max_steps": 200
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/feature-params/tenant_id/c1/", strings.NewReader(body))
	req.Header.Set(gatewayauth.HeaderUserID, "admin-u")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	req.Header.Set(headerFeatureParamsAccessContext, fpContextCompanySettings)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	row, err := loadTenantFeatureParams("c1")
	if err != nil || row == nil {
		t.Fatalf("loadTenantFeatureParams: %v", err)
	}
	var providers []map[string]any
	if err := json.Unmarshal([]byte(row.ProvidersJSON), &providers); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if providers[0]["base_url"] != "https://gateway.example.com/anthropic" {
		t.Fatalf("persisted base_url=%v json=%s", providers[0]["base_url"], row.ProvidersJSON)
	}
}
