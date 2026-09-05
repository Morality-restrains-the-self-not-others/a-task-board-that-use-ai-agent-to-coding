package main

import (
	"fmt"
	"net/http"
	"strings"
)

func handlePersonalFeatureParamsList(w http.ResponseWriter, r *http.Request) {
	userID, ok := ensureFeatureParamsAuthenticated(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, status, err := listPersonalFeatureParamsHTTP(userID)
		if err != nil || status != http.StatusOK {
			writeFeatureParamsMessage(w, r, http.StatusBadGateway, "读取个人配置失败")
			return
		}
		configs := make([]map[string]any, 0, len(items))
		for _, item := range items {
			configs = append(configs, serializePersonalFeatureParamsItem(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"configs": configs})
	case http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, "无效 JSON")
			return
		}
		name := strings.TrimSpace(strField(body, "name"))
		if name == "" {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, "配置名不能为空")
			return
		}
		companyID := strings.TrimSpace(strField(body, "company_id"))
		if companyID == "" {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, "company_id 不能为空")
			return
		}
		if _, _, memberOK := resolveTenantMemberFP(w, r, companyID, userID); !memberOK {
			writeFeatureParamsMessage(w, r, http.StatusForbidden, "不属于该公司")
			return
		}
		providers := normalizeProvidersList(body["providers"])
		forceSubTokenDisabled(providers) // OPT-20260816-016: 派生子 Key 功能下线，写路径强制关闭
		maxSteps, stepErr := parseAgentMaxSteps(body, 200)
		if stepErr != "" {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, stepErr)
			return
		}
		payload := map[string]interface{}{
			"user_id":                userID,
			"company_id":             companyID,
			"name":                   name,
			"providers":              providers,
			"agent_model":            strings.TrimSpace(strField(body, "agent_model")),
			"agent_model_provider":   strings.ToLower(strings.TrimSpace(strField(body, "agent_model_provider"))),
			"agent_max_steps":        maxSteps,
			"summary_model":          strings.TrimSpace(strField(body, "summary_model")),
			"summary_model_provider": strings.ToLower(strings.TrimSpace(strField(body, "summary_model_provider"))),
			"llm_budget_enabled":     computeLLMBudgetEnabled(providers),
			"extra_env_vars":         extraEnvVarsFromBody(body),
		}
		params, status, err := upsertPersonalFeatureParamsHTTP(payload)
		if err != nil || status >= 400 {
			writeFeatureParamsMessage(w, r, status, "创建失败")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"message": "创建成功",
			"config":  serializePersonalFeatureParamsItem(params),
		})
	default:
		writeFeatureParamsMessage(w, r, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func handlePersonalFeatureParamsDetail(w http.ResponseWriter, r *http.Request, configID string) {
	userID, ok := ensureFeatureParamsAuthenticated(w, r)
	if !ok {
		return
	}
	configID = strings.TrimSpace(configID)
	if configID == "" {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, "缺少配置ID")
		return
	}
	cfgRow, err := loadPersonalFeatureParams(configID, userID)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "读取配置失败")
		return
	}
	if cfgRow == nil {
		writeFeatureParamsMessage(w, r, http.StatusNotFound, "配置不存在")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"config": serializePersonalFeatureParamsConfig(cfgRow),
		})
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, "无效 JSON")
			return
		}
		payload := map[string]interface{}{
			"id":      configID,
			"user_id": userID,
		}
		if name, ok := body["name"]; ok {
			payload["name"] = strings.TrimSpace(fmt.Sprintf("%v", name))
		}
		if providers, ok := body["providers"]; ok {
			providersList := normalizeProvidersList(providers)
			forceSubTokenDisabled(providersList) // OPT-20260816-016: 派生子 Key 功能下线，写路径强制关闭
			payload["providers"] = providersList
		}
		if extra, ok := body["extra_env_vars"]; ok {
			if items, ok2 := extra.([]any); ok2 {
				out := make([]map[string]any, 0, len(items))
				for _, item := range items {
					if m, ok3 := item.(map[string]any); ok3 {
						out = append(out, m)
					}
				}
				payload["extra_env_vars"] = out
			}
		}
		for _, field := range []string{"agent_model", "agent_model_provider", "summary_model", "summary_model_provider"} {
			if v, ok := body[field]; ok {
				val := strings.TrimSpace(fmt.Sprintf("%v", v))
				if strings.Contains(field, "provider") {
					val = strings.ToLower(val)
				}
				payload[field] = val
			}
		}
		if _, ok := body["agent_max_steps"]; ok {
			maxSteps, stepErr := parseAgentMaxSteps(body, cfgRow.AgentMaxSteps)
			if stepErr != "" {
				writeFeatureParamsMessage(w, r, http.StatusBadRequest, stepErr)
				return
			}
			payload["agent_max_steps"] = maxSteps
		}
		params, status, err := upsertPersonalFeatureParamsHTTP(payload)
		if err != nil || status >= 400 {
			writeFeatureParamsMessage(w, r, http.StatusBadGateway, "更新失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "更新成功",
			"config":  serializePersonalFeatureParamsItem(params),
		})
	case http.MethodDelete:
		status, err := deletePersonalFeatureParamsHTTP(configID, userID)
		if err != nil || status >= 400 {
			writeFeatureParamsMessage(w, r, http.StatusBadGateway, "删除失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "删除成功"})
	default:
		writeFeatureParamsMessage(w, r, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func serializePersonalFeatureParamsItem(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	cfg := personalConfigFromParamsMap(params)
	out := serializePersonalFeatureParamsConfig(cfg)
	if v, ok := params["created_at"]; ok {
		out["created_at"] = v
	}
	if v, ok := params["updated_at"]; ok {
		out["updated_at"] = v
	}
	return out
}

func personalConfigFromParamsMap(params map[string]any) *featureParamsConfig {
	steps := intFromAny(params["agent_max_steps"], 200)
	name := strField(params, "name")
	return &featureParamsConfig{
		ID:              strField(params, "id"),
		ProvidersJSON:   coalesceJSONString(params["providers"]),
		AgentModel:      strField(params, "agent_model"),
		AgentProvider:   strField(params, "agent_model_provider"),
		AgentMaxSteps:   steps,
		SummaryModel:    strField(params, "summary_model"),
		SummaryProvider: strField(params, "summary_model_provider"),
		ExtraEnvJSON:    coalesceJSONString(params["extra_env_vars"]),
		DisplayName:     name,
		Scope:           "personal",
	}
}

func serializePersonalFeatureParamsConfig(cfg *featureParamsConfig) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	providers := providersFromJSONString(cfg.ProvidersJSON)
	extra := parseExtraEnvVarsJSON(cfg.ExtraEnvJSON)
	name := cfg.DisplayName
	if strings.HasPrefix(name, "个人配置:") {
		name = strings.TrimPrefix(name, "个人配置:")
	}
	out := map[string]any{
		"id":                     cfg.ID,
		"name":                   name,
		"providers":              providers,
		"agent_model":            cfg.AgentModel,
		"agent_model_provider":   cfg.AgentProvider,
		"agent_max_steps":        fmt.Sprintf("%d", cfg.AgentMaxSteps),
		"summary_model":          cfg.SummaryModel,
		"summary_model_provider": cfg.SummaryProvider,
		"extra_env_vars":         extra,
		"env_preview":            envPreviewFromConfig(cfg),
	}
	return out
}
