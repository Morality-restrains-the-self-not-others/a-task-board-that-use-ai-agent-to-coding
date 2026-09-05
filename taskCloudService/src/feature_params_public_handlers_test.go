package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gatewayauth"
)

func setupFeatureParamsPublicTest(t *testing.T) *saasHTTPStore {
	t.Helper()
	store := setupBudgetTestDB(t)
	cfg.GatewayInternalSecret = "gw-secret"
	store.putMember("c1", "u1", "m1", false)
	store.putMember("c1", "admin-u", "m-admin", true)
	store.putCreator("c1", "admin-u")
	providers := `[{"provider":"openai","api_key":"sk-secret","base_url":"https://api.openai.com/v1","supported_models":["gpt-4.1"],"use_sub_token":false,"budget_enabled":false}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "tp1", CompanyID: "c1", ProvidersJSON: providers, LLMBudgetEnabled: false,
		AgentModel: "gpt-4.1", AgentProvider: "openai", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	return store
}

func TestParseSupportedModelsFieldChineseComma(t *testing.T) {
	got := parseSupportedModelsField("gpt-4.1，gpt-4.1-mini")
	if len(got) != 2 || got[0] != "gpt-4.1" || got[1] != "gpt-4.1-mini" {
		t.Fatalf("chinese comma: got %#v", got)
	}
	got = parseSupportedModelsField("gpt-4.1, gpt-4.1-mini；o3-mini")
	if len(got) != 3 || got[2] != "o3-mini" {
		t.Fatalf("mixed separators: got %#v", got)
	}
	entry := normalizeProviderEntry(map[string]any{
		"provider":         "openai",
		"api_key":          "sk",
		"base_url":         "https://api.openai.com/v1",
		"supported_models": "gpt-4.1，gpt-4.1-mini",
	})
	models, _ := entry["supported_models"].([]string)
	if len(models) != 2 || models[0] != "gpt-4.1" || models[1] != "gpt-4.1-mini" {
		t.Fatalf("normalizeProviderEntry: %#v", entry["supported_models"])
	}
}

func TestNormalizeProviderBaseURLKeepsOperatorTypedPaths(t *testing.T) {
	got := normalizeProviderBaseURL("deepseek", "https://api.deepseek.com/anthropic")
	if got != "https://api.deepseek.com/anthropic" {
		t.Fatalf("got=%q", got)
	}
	got = normalizeProviderBaseURL("deepSeek", "https://api.deepseek.com/anthropic/")
	if got != "https://api.deepseek.com/anthropic" {
		t.Fatalf("trailing slash got=%q", got)
	}
	got = normalizeProviderBaseURL("anthropic", "https://api.deepseek.com/anthropic")
	if got != "https://api.deepseek.com/anthropic" {
		t.Fatalf("anthropic provider mutated: %q", got)
	}
	got = normalizeProviderBaseURL("deepseek", "https://api.deepseek.com/v1")
	if got != "https://api.deepseek.com/v1" {
		t.Fatalf("valid deepseek url mutated: %q", got)
	}
	got = normalizeProviderBaseURL("deepseek", "https://gateway.example.com/anthropic")
	if got != "https://gateway.example.com/anthropic" {
		t.Fatalf("custom host mutated: %q", got)
	}
}

func TestNormalizeProviderBaseURLRepairsConcatenatedSchemes(t *testing.T) {
	// Auto-fill https://api.deepseek.com/v1 + paste https://api.deepseek.com
	got := normalizeProviderBaseURL("deepseek", "https://api.deepseek.com/v1https://api.deepseek.com")
	if got != "https://api.deepseek.com" {
		t.Fatalf("concat repaired=%q", got)
	}
	got = normalizeProviderBaseURL("openai", "https://api.openai.com/v1https://proxy.example.com/v1")
	if got != "https://proxy.example.com/v1" {
		t.Fatalf("openai concat repaired=%q", got)
	}
	got = normalizeProviderBaseURL("openai", "https://api.openai.com/v1")
	if got != "https://api.openai.com/v1" {
		t.Fatalf("single url mutated: %q", got)
	}
	// Query-embedded URL must not be split.
	got = normalizeProviderBaseURL("openai", "https://example.com/p?next=https://other.example/v1")
	if got != "https://example.com/p?next=https://other.example/v1" {
		t.Fatalf("query url mutated: %q", got)
	}
}

func TestNormalizeProviderEntryKeepsOperatorTypedBaseURL(t *testing.T) {
	entry := normalizeProviderEntry(map[string]any{
		"provider": "deepseek",
		"api_key":  "sk",
		"base_url": "https://api.deepseek.com/anthropic",
	})
	if entry["base_url"] != "https://api.deepseek.com/anthropic" {
		t.Fatalf("base_url=%v", entry["base_url"])
	}
}

func TestCompanyFeatureParamsGetWithoutContextForbidden(t *testing.T) {
	setupFeatureParamsPublicTest(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/feature-params/tenant_id/c1/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompanyFeatureParamsGetSummaryRedactsApiKey(t *testing.T) {
	setupFeatureParamsPublicTest(t)
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
	providers, _ := data["providers"].([]any)
	if len(providers) == 0 {
		t.Fatal("expected providers")
	}
	p0, _ := providers[0].(map[string]any)
	if p0["api_key"] != "" {
		t.Fatalf("expected empty api_key, got %v", p0["api_key"])
	}
	if _, ok := data["env_preview"]; ok {
		t.Fatal("env_preview should be redacted")
	}
}

func TestCompanyFeatureParamsGetCompanySettingsWithKey(t *testing.T) {
	setupFeatureParamsPublicTest(t)
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
	if p0["api_key"] != "sk-secret" {
		t.Fatalf("expected api key preserved, got %v", p0["api_key"])
	}
	if _, ok := data["env_preview"]; !ok {
		t.Fatal("expected env_preview in full view")
	}
}

func TestCompanyFeatureParamsGetNonMemberForbidden(t *testing.T) {
	store := setupFeatureParamsPublicTest(t)
	store.putCreator("c2", "other-u")
	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/feature-params/tenant_id/c2/?view=summary", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPersonalFeatureParamsList(t *testing.T) {
	store := setupFeatureParamsPublicTest(t)
	store.putPersonal(&featureParamsConfig{
		ID: "pc1", ProvidersJSON: "[]", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
		DisplayName: "个人配置:mine", Scope: "personal",
	}, "u1")
	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/personal/feature-params-configs/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "pc1") {
		t.Fatalf("expected config in list: %s", rec.Body.String())
	}
}

// TestPersonalFeatureParamsErrorCarriesTraceID — 元规则：错误响应 body 必须携带
// trace_id（请求 X-Trace-Id），支撑前端 data-traceId 绑定（OPT-20260806-027）。
func TestPersonalFeatureParamsErrorCarriesTraceID(t *testing.T) {
	setupFeatureParamsPublicTest(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	// 未认证 → 401，且 body.trace_id 回显请求 X-Trace-Id
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/personal/feature-params-configs/", nil)
	req.Header.Set("X-Trace-Id", "trace-for-personal-001")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["message"] != "未认证" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
	if body["trace_id"] != "trace-for-personal-001" {
		t.Fatalf("expected trace_id echoed in error body, got %q (body=%s)", body["trace_id"], rec.Body.String())
	}

	// 无 X-Trace-Id 请求头 → 不输出空 trace_id 噪音
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/personal/feature-params-configs/", nil)
	mux.ServeHTTP(rec2, req2)
	var body2 map[string]string
	if err := json.Unmarshal(rec2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, has := body2["trace_id"]; has {
		t.Fatalf("trace_id should be omitted when request has none: %s", rec2.Body.String())
	}
}

// OPT-20260809-019: feature-params GET 响应须携带 env_var_sources_available 标志，
// summary 视图脱敏 extra_env_vars 后仍保留，前端据此判定来源可用性而无需拉取 full payload。

func TestCompanyFeatureParamsGetSummaryExposesEnvVarSourcesAvailable(t *testing.T) {
	store := setupFeatureParamsPublicTest(t)
	// 公司级 extra_env_vars 含 key 条目 → company 可用；空 key 条目不计数
	store.putTenant(&tenantFeatureParamsRow{
		ID: "tp1", CompanyID: "c1", ProvidersJSON: "[]", LLMBudgetEnabled: false,
		AgentMaxSteps: 200, ExtraEnvJSON: `[{"key":"COMPANY_KEY","value":"v"},{"key":"","value":"skip"}]`,
	})
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
	// summary 模式下 extra_env_vars 已被脱敏为 []，但标志须保留
	if _, ok := data["extra_env_vars"]; !ok {
		t.Fatal("extra_env_vars should still be present (redacted)")
	}
	flag, _ := data["env_var_sources_available"].(map[string]any)
	if flag["company"] != true {
		t.Fatalf("expected company=true, got %#v (body=%s)", flag, rec.Body.String())
	}
	// 公司级 GET 无工作空间上下文 → workspace 恒 false
	if flag["workspace"] != false {
		t.Fatalf("expected workspace=false, got %#v", flag)
	}
}

func TestCompanyFeatureParamsGetSummaryEnvVarsEmptyFlagFalse(t *testing.T) {
	// 无自定义变量、无 LLM provider/模型时两级均 false（默认夹具含 openai，须覆盖为空）
	store := setupFeatureParamsPublicTest(t)
	store.putTenant(&tenantFeatureParamsRow{
		ID: "tp1", CompanyID: "c1", ProvidersJSON: "[]", LLMBudgetEnabled: false,
		AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
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
	if flag["company"] != false || flag["workspace"] != false {
		t.Fatalf("expected company=false workspace=false, got %#v", flag)
	}
}

func TestSerializeWorkspaceFeatureParamsDataEnvVarSourcesAvailable(t *testing.T) {
	// 工作空间自有 extra_env_vars 含 key → workspace=true；公司为空 → company=false
	wsLoad := &workspaceFeatureParamsLoad{
		Found: true, UseCompanyDefault: false,
		Params: map[string]any{
			"providers":       `[]`,
			"extra_env_vars":  `[{"key":"WS_KEY","value":"v"},{"key":"","value":"skip"}]`,
			"agent_max_steps": "200",
		},
	}
	company := &tenantFeatureParamsRow{ID: "tp1", ProvidersJSON: "[]", AgentMaxSteps: 200, ExtraEnvJSON: "[]"}
	data := serializeWorkspaceFeatureParamsData(wsLoad, company, "ws1")
	flag, _ := data["env_var_sources_available"].(map[string]any)
	if flag["company"] != false {
		t.Fatalf("expected company=false, got %#v", flag)
	}
	if flag["workspace"] != true {
		t.Fatalf("expected workspace=true, got %#v", flag)
	}

	// 反向：公司含 key、工作空间 use_company_default（无自有变量）→ company=true workspace=false
	company2 := &tenantFeatureParamsRow{ID: "tp1", ProvidersJSON: "[]", AgentMaxSteps: 200, ExtraEnvJSON: `[{"key":"COMPANY_KEY","value":"v"}]`}
	wsLoad2 := &workspaceFeatureParamsLoad{Found: true, UseCompanyDefault: true, Params: map[string]any{
		"providers": `[]`, "extra_env_vars": `[]`, "agent_max_steps": "200",
	}}
	data2 := serializeWorkspaceFeatureParamsData(wsLoad2, company2, "ws1")
	flag2, _ := data2["env_var_sources_available"].(map[string]any)
	if flag2["company"] != true {
		t.Fatalf("expected company=true, got %#v", flag2)
	}
	if flag2["workspace"] != false {
		t.Fatalf("expected workspace=false, got %#v", flag2)
	}
}

// TestCompanyFeatureParamsPostForceSubTokenDisabled — OPT-20260816-016 回归：
// 派生子 Key 开关已从租户页移除，直接 POST API 携带 use_sub_token=true 时，
// 写路径必须强制规范为 false（落库与响应均不应再开启）。
func TestCompanyFeatureParamsPostForceSubTokenDisabled(t *testing.T) {
	setupFeatureParamsPublicTest(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{
		"providers": [{
			"provider": "openai",
			"api_key": "sk-secret",
			"base_url": "https://api.openai.com/v1",
			"supported_models": ["gpt-4.1"],
			"use_sub_token": true,
			"budget_enabled": true
		}],
		"agent_model": "gpt-4.1",
		"agent_model_provider": "openai",
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
		t.Fatalf("unmarshal providers: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("providers=%s", row.ProvidersJSON)
	}
	if providerBool(providers[0]["use_sub_token"]) {
		t.Fatalf("use_sub_token should be forced false on write, got %s", row.ProvidersJSON)
	}
	if providerBool(providers[0]["budget_enabled"]) {
		t.Fatalf("budget_enabled should be forced false on write, got %s", row.ProvidersJSON)
	}

	// 响应 data.providers 同样应为 false
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid resp json: %v", err)
	}
	data, _ := resp["data"].(map[string]any)
	pArr, _ := data["providers"].([]any)
	if len(pArr) == 0 {
		t.Fatal("expected providers in response")
	}
	if providerBool(pArr[0].(map[string]any)["use_sub_token"]) {
		t.Fatalf("response use_sub_token should be false: %s", rec.Body.String())
	}
}

// TestPersonalFeatureParamsCreateForceSubTokenDisabled — OPT-20260816-016 个人配置创建路径同样强制关闭。
func TestPersonalFeatureParamsCreateForceSubTokenDisabled(t *testing.T) {
	setupFeatureParamsPublicTest(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{
		"name": "my-config",
		"company_id": "c1",
		"providers": [{
			"provider": "openai",
			"api_key": "sk-secret",
			"base_url": "https://api.openai.com/v1",
			"supported_models": ["gpt-4.1"],
			"use_sub_token": true,
			"budget_enabled": true
		}],
		"agent_model": "gpt-4.1",
		"agent_model_provider": "openai",
		"agent_max_steps": 200
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/personal/feature-params-configs/", strings.NewReader(body))
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid resp json: %v", err)
	}
	cfg, _ := resp["config"].(map[string]any)
	pArr, _ := cfg["providers"].([]any)
	if len(pArr) == 0 {
		t.Fatal("expected providers in personal config")
	}
	p0, _ := pArr[0].(map[string]any)
	if providerBool(p0["use_sub_token"]) {
		t.Fatalf("personal config use_sub_token should be false: %s", rec.Body.String())
	}
	if providerBool(p0["budget_enabled"]) {
		t.Fatalf("personal config budget_enabled should be false: %s", rec.Body.String())
	}
}
