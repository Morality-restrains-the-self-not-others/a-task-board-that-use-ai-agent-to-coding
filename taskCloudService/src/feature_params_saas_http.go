package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Local SQL accessors for feature-params tables (user resolution stays on saas HTTP).

func loadTenantFeatureParams(companyID string) (*tenantFeatureParamsRow, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, nil
	}
	return queryTenantFeatureParamsRow(companyID)
}

func tenantFeatureParamsFromMap(params map[string]any) *tenantFeatureParamsRow {
	steps := 200
	switch v := params["agent_max_steps"].(type) {
	case float64:
		steps = int(v)
	case int:
		steps = v
	}
	if steps <= 0 {
		steps = 200
	}
	enabled := false
	switch v := params["llm_budget_enabled"].(type) {
	case bool:
		enabled = v
	case float64:
		enabled = v != 0
	}
	return &tenantFeatureParamsRow{
		ID:               strField(params, "id"),
		CompanyID:        strField(params, "company_id"),
		ProvidersJSON:    coalesceJSONString(params["providers"]),
		LLMBudgetEnabled: enabled,
		AgentModel:       strField(params, "agent_model"),
		AgentProvider:    strField(params, "agent_model_provider"),
		AgentMaxSteps:    steps,
		SummaryModel:     strField(params, "summary_model"),
		SummaryProvider:  strField(params, "summary_model_provider"),
		ExtraEnvJSON:     coalesceJSONString(params["extra_env_vars"]),
	}
}

func loadWorkspaceFeatureParams(workspaceID string) (*featureParamsConfig, bool, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, false, nil
	}
	load, err := queryWorkspaceFeatureParamsLoad(workspaceID)
	if err != nil {
		return nil, false, err
	}
	if !load.Found {
		return nil, false, nil
	}
	if load.UseCompanyDefault {
		return nil, false, nil
	}
	params := load.Params
	if params == nil {
		return nil, false, nil
	}
	steps := intFromAny(params["agent_max_steps"], 200)
	cfgRow := &featureParamsConfig{
		ID:              strField(params, "id"),
		ProvidersJSON:   coalesceJSONString(params["providers"]),
		AgentModel:      strField(params, "agent_model"),
		AgentProvider:   strField(params, "agent_model_provider"),
		AgentMaxSteps:   steps,
		SummaryModel:    strField(params, "summary_model"),
		SummaryProvider: strField(params, "summary_model_provider"),
		ExtraEnvJSON:    coalesceJSONString(params["extra_env_vars"]),
		Scope:           "workspace",
		DisplayName:     fmt.Sprintf("工作空间配置:%s", workspaceID),
	}
	return cfgRow, true, nil
}

func loadPersonalFeatureParams(configID, userID string) (*featureParamsConfig, error) {
	configID = strings.TrimSpace(configID)
	userID = strings.TrimSpace(userID)
	if configID == "" || userID == "" {
		return nil, nil
	}
	params, err := queryPersonalFeatureParamsRow(configID, userID)
	if err != nil {
		return nil, err
	}
	if params == nil {
		return nil, nil
	}
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
		Scope:           "personal",
		DisplayName:     fmt.Sprintf("个人配置:%s", name),
	}, nil
}

func resolveOwnerUserID(tenantID, ownerKey string) string {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return ""
	}
	// 2026-07-29 (OPT-026): call taskTenantService directly instead of
	// Django resolve-owner-user → tenant_client.get_member_by_id → taskTenantService.
	// Eliminates the Django HTTP hop.
	if strings.TrimSpace(cfg.TaskTenantURL) != "" {
		q := url.Values{}
		q.Set("member_id", ownerKey)
		cid := strings.TrimSpace(tenantID)
		if cid != "" {
			q.Set("company_id", cid)
		}
		raw, status, err := tenantGET(nil, "/api/internal/tenant/members/by-id", q)
		if err == nil && status == 200 && len(raw) > 0 {
			var out map[string]interface{}
			if json.Unmarshal(raw, &out) == nil {
				if uid := strField(out, "user_id"); uid != "" {
					return uid
				}
			}
		}
	}
	// Django fallback removed — taskTenantService is the SSOT (2026-07-30).
	return ownerKey
}

func writeFeatureParamsSnapshot(
	taskID, workspaceID, tenantID string,
	meta featureParamsSnapshotMeta,
	env map[string]string,
	cfgRow *featureParamsConfig,
) error {
	return insertFeatureParamsSnapshotLocal(taskID, workspaceID, tenantID, meta, env, cfgRow)
}

func writeFeatureParamsSnapshotSafe(
	taskID, workspaceID, tenantID string,
	meta featureParamsSnapshotMeta,
	env map[string]string,
	cfgRow *featureParamsConfig,
) {
	if err := writeFeatureParamsSnapshot(taskID, workspaceID, tenantID, meta, env, cfgRow); err != nil {
		logWarn(fmt.Sprintf(
			"Failed to write feature params snapshot for task=%s — continuing: %v",
			taskID, err,
		), "")
	}
}

func recordTaskApiKeyUsage(
	tenantID, workspaceID, taskID string,
	providers []map[string]any,
) error {
	return insertTaskApiKeyUsageLocal(tenantID, workspaceID, taskID, providers)
}

func recordTaskApiKeyUsageSafe(
	tenantID, workspaceID, taskID string,
	providers []map[string]any,
) {
	if err := recordTaskApiKeyUsage(tenantID, workspaceID, taskID, providers); err != nil {
		logWarn(fmt.Sprintf(
			"record_task_api_key_usage skipped task=%s: %v", taskID, err,
		), "")
	}
}
