package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gatewayauth"
)

func TestFeatureParamsSourceConfigured(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		extra         []map[string]any
		providers     []map[string]any
		agentModel    string
		agentProvider string
		want          bool
	}{
		{name: "all empty", want: false},
		{name: "empty slices", extra: []map[string]any{}, providers: []map[string]any{}, want: false},
		{name: "placeholder provider only", providers: []map[string]any{{"provider": "", "api_key": ""}}, want: false},
		{name: "blank extra keys", extra: []map[string]any{{"key": "", "value": "v"}, {"key": "  ", "value": "x"}}, want: false},
		{name: "custom extra env key", extra: []map[string]any{{"key": "COMPANY_KEY", "value": "v"}}, want: true},
		{name: "named llm provider", providers: []map[string]any{{"provider": "deepseek"}}, want: true},
		{name: "agent model only", agentModel: "deepseek-v4-flash", want: true},
		{name: "agent provider only", agentProvider: "deepseek", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := featureParamsSourceConfigured(tc.extra, tc.providers, tc.agentModel, tc.agentProvider)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// 复现：公司设置页已配 LLM provider/模型，extra_env_vars=[] 时旧逻辑把 company 标为 false，
// 任务详情误报「暂无可用环境变量」。
func TestCompanyFeatureParamsGetSummaryLLMConfigCountsAsAvailable(t *testing.T) {
	setupFeatureParamsPublicTest(t) // openai + gpt-4.1, extra=[]
	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/feature-params/tenant_id/c1/?view=summary", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	flag, _ := data["env_var_sources_available"].(map[string]any)
	if flag["company"] != true {
		t.Fatalf("expected company=true when LLM provider/model is set, got %#v (body=%s)", flag, rec.Body.String())
	}
	if flag["workspace"] != false {
		t.Fatalf("expected workspace=false on company GET, got %#v", flag)
	}
}

func TestSerializeWorkspaceFeatureParamsDataCompanyLLMConfigAvailable(t *testing.T) {
	t.Parallel()
	company := &tenantFeatureParamsRow{
		ID: "tp1", ProvidersJSON: `[{"provider":"deepseek"}]`,
		AgentModel: "deepseek-v4-flash", AgentProvider: "deepseek", AgentMaxSteps: 200,
		ExtraEnvJSON: "[]",
	}
	wsLoad := &workspaceFeatureParamsLoad{Found: true, UseCompanyDefault: true, Params: map[string]any{
		"providers": `[]`, "extra_env_vars": `[]`, "agent_max_steps": "200",
	}}
	data := serializeWorkspaceFeatureParamsData(wsLoad, company, "ws1")
	flag, _ := data["env_var_sources_available"].(map[string]any)
	if flag["company"] != true {
		t.Fatalf("expected company=true from LLM config, got %#v", flag)
	}
	if flag["workspace"] != false {
		t.Fatalf("expected workspace=false when inheriting company default, got %#v", flag)
	}
}

func TestSerializeWorkspaceFeatureParamsDataWorkspaceLLMConfigAvailable(t *testing.T) {
	t.Parallel()
	company := &tenantFeatureParamsRow{ID: "tp1", ProvidersJSON: "[]", AgentMaxSteps: 200, ExtraEnvJSON: "[]"}
	wsLoad := &workspaceFeatureParamsLoad{Found: true, UseCompanyDefault: false, Params: map[string]any{
		"providers":            `[{"provider":"openai"}]`,
		"agent_model":          "gpt-4.1",
		"agent_model_provider": "openai",
		"extra_env_vars":       `[]`,
		"agent_max_steps":      "200",
	}}
	data := serializeWorkspaceFeatureParamsData(wsLoad, company, "ws1")
	flag, _ := data["env_var_sources_available"].(map[string]any)
	if flag["company"] != false {
		t.Fatalf("expected company=false, got %#v", flag)
	}
	if flag["workspace"] != true {
		t.Fatalf("expected workspace=true from workspace LLM config, got %#v", flag)
	}
}

func TestParseExtraEnvVarsAnyTypedMaps(t *testing.T) {
	t.Parallel()
	got := parseExtraEnvVarsAny([]map[string]any{{"key": "WS_KEY", "value": "v"}})
	if !hasKeyedEnvVarEntry(got) {
		t.Fatalf("typed []map[string]any extra_env_vars should count as keyed, got %#v", got)
	}
}
