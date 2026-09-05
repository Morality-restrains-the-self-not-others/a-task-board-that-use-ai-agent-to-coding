package main

import (
	"fmt"
	"net/http"
	"strings"
)

func handleCompanyFeatureParams(w http.ResponseWriter, r *http.Request, tenantID string) {
	userID, ok := ensureFeatureParamsAuthenticated(w, r)
	if !ok {
		return
	}
	exists, err := verifyTenantCompanyExistsFP(tenantID)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "租户校验失败")
		return
	}
	if !exists {
		writeFeatureParamsMessage(w, r, http.StatusNotFound, "租户不存在")
		return
	}
	_, isAdmin, memberOK := resolveTenantMemberFP(w, r, tenantID, userID)
	if !memberOK {
		return
	}

	decision := resolveFeatureParamsAccess(r, fpResourceCompany)
	if !decision.Allowed {
		recordFeatureParamsAccessAudit(r, userID, tenantID, "", fpResourceCompany, decision, http.StatusForbidden)
		writeFeatureParamsMessage(w, r, http.StatusForbidden, decision.Message)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleCompanyFeatureParamsGet(w, r, tenantID, userID, isAdmin, decision)
	case http.MethodPost:
		handleCompanyFeatureParamsPost(w, r, tenantID, userID, isAdmin, decision)
	default:
		writeFeatureParamsMessage(w, r, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func handleCompanyFeatureParamsGet(
	w http.ResponseWriter, r *http.Request,
	tenantID, userID string, isAdmin bool, decision featureParamsAccessDecision,
) {
	params, err := loadTenantFeatureParams(tenantID)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "读取功能参数失败")
		return
	}
	data := serializeCompanyFeatureParamsData(params)
	if decision.ViewMode == fpViewSummary {
		data = redactFeatureParamsPayload(data)
	}
	recordFeatureParamsAccessAudit(r, userID, tenantID, "", fpResourceCompany, decision, http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":            data,
		"is_tenant_admin": isAdmin,
	})
}

func handleCompanyFeatureParamsPost(
	w http.ResponseWriter, r *http.Request,
	tenantID, userID string, isAdmin bool, decision featureParamsAccessDecision,
) {
	if !isAdmin {
		recordFeatureParamsAccessAudit(r, userID, tenantID, "", fpResourceCompany, decision, http.StatusForbidden)
		writeFeatureParamsMessage(w, r, http.StatusForbidden, "仅租户管理员可修改功能参数配置")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, "无效 JSON")
		return
	}
	if msg := rejectLegacyFeatureParamsFields(body); msg != "" {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, msg)
		return
	}
	if _, ok := body["llm_budget_enabled"]; ok {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest,
			"llm_budget_enabled 已废弃，请在各 API 端点勾选 budget_enabled（启用 LLM 预算）")
		return
	}
	providers := normalizeProvidersList(body["providers"])
	forceSubTokenDisabled(providers) // OPT-20260816-016: 派生子 Key 功能下线，写路径强制关闭
	maxSteps, stepErr := parseAgentMaxSteps(body, 200)
	if stepErr != "" {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, stepErr)
		return
	}
	llmBudgetEnabled := computeLLMBudgetEnabled(providers)
	payload := map[string]interface{}{
		"company_id":             tenantID,
		"providers":              providers,
		"agent_model":            strings.TrimSpace(strField(body, "agent_model")),
		"agent_model_provider":   strings.ToLower(strings.TrimSpace(strField(body, "agent_model_provider"))),
		"agent_max_steps":        maxSteps,
		"summary_model":          strings.TrimSpace(strField(body, "summary_model")),
		"summary_model_provider": strings.ToLower(strings.TrimSpace(strField(body, "summary_model_provider"))),
		"llm_budget_enabled":     llmBudgetEnabled,
		"extra_env_vars":         extraEnvVarsFromBody(body),
	}
	paramsMap, status, err := upsertTenantFeatureParamsHTTP(payload)
	if err != nil || status == http.StatusNotFound {
		writeFeatureParamsMessage(w, r, http.StatusNotFound, "租户不存在")
		return
	}
	if status < 200 || status >= 300 {
		writeFeatureParamsMessage(w, r, status, "保存失败")
		return
	}
	row := tenantFeatureParamsFromMap(paramsMap)
	recordFeatureParamsAccessAudit(r, userID, tenantID, "", fpResourceCompany, decision, http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "保存成功",
		"data":    serializeCompanyFeatureParamsData(row),
	})
}

func serializeCompanyFeatureParamsData(params *tenantFeatureParamsRow) map[string]any {
	providers := []map[string]any{}
	extra := []map[string]any{}
	agentModel, agentProvider := "", ""
	summaryModel, summaryProvider := "", ""
	maxSteps := 200
	var id string

	if params != nil {
		providers = providersFromJSONString(params.ProvidersJSON)
		extra = parseExtraEnvVarsJSON(params.ExtraEnvJSON)
		agentModel = params.AgentModel
		agentProvider = params.AgentProvider
		summaryModel = params.SummaryModel
		summaryProvider = params.SummaryProvider
		maxSteps = params.AgentMaxSteps
		if maxSteps <= 0 {
			maxSteps = 200
		}
		id = params.ID
	}

	base := map[string]any{
		"providers":              providers,
		"agent_model":            agentModel,
		"agent_model_provider":   agentProvider,
		"agent_max_steps":        fmt.Sprintf("%d", maxSteps),
		"summary_model":          summaryModel,
		"summary_model_provider": summaryProvider,
		"llm_budget_enabled":     computeLLMBudgetEnabled(providers),
		"extra_env_vars":         extra,
	}
	// OPT-20260809-019: 来源可用性标志随 data 下发，summary 脱敏时保留。
	// 公司级 GET 无工作空间上下文，workspace 恒为 false。
	base["env_var_sources_available"] = featureParamsSourcesAvailable(
		featureParamsSourceConfigured(extra, providers, agentModel, agentProvider),
		false,
	)
	if id != "" {
		base["id"] = id
	}
	cfgRow := companyConfigFromTenantRow(params, "公司默认")
	base["env_preview"] = envPreviewFromConfig(cfgRow)
	return base
}
