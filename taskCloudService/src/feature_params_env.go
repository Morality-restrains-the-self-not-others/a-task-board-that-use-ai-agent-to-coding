package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FeatureParams env resolve (full hierarchy):
// 1. Read task binding (feature_params_source / personal_feature_params_config_id) via taskTaskService
// 2. Resolve personal → workspace → company (FeatureParamsResolver parity)
// 3. Serialize env (FeatureParamsEnvSerializer core fields)
// 4. Inject budget policy; rewrite use_sub_token providers for AI endpoint proxy; record TaskApiKeyUsage
// 5. Append-only snapshot write (failure logged, does not block response)
//
// Note: Unlike the old MVP, we do NOT short-circuit on an existing snapshot —
// each pull re-resolves and appends a new audit row (Django ApplicationService parity).

const (
	envProvidersJSON   = "TASK_LLM_PROVIDERS_JSON"
	envAgentModel      = "TASK_AGENT_MODEL"
	envAgentProvider   = "TASK_AGENT_MODEL_PROVIDER"
	envAgentMaxSteps   = "TASK_AGENT_MAX_STEPS"
	envSummaryModel    = "TASK_SUMMARY_MODEL"
	envSummaryProvider = "TASK_SUMMARY_MODEL_PROVIDER"
	envScope           = "TASK_FEATURE_PARAMS_SCOPE"
	envConfigID        = "TASK_FEATURE_PARAMS_CONFIG_ID"
	envConfigName      = "TASK_FEATURE_PARAMS_CONFIG_NAME"
	envBudgetPolicy    = "TASK_LLM_BUDGET_POLICY"
)

type featureParamsResolveResult struct {
	CompanyID   string            `json:"company_id"`
	WorkspaceID string            `json:"workspace_id"`
	TaskID      string            `json:"task_id"`
	Env         map[string]string `json:"env"`
	Source      string            `json:"resolve_source,omitempty"`
}

func buildCompanyFeatureParamsEnv(params *tenantFeatureParamsRow, companyID string) map[string]string {
	cfgRow := companyConfigFromTenantRow(params, "公司默认")
	_ = companyID
	return serializeFeatureParamsEnv(cfgRow)
}

func buildTaskLLMBudgetPolicyEnv(budgetConn *sql.DB, todoID, workspaceID, companyID string) (string, error) {
	params, err := loadTenantFeatureParams(companyID)
	if err != nil {
		return "", err
	}
	if params == nil || !params.LLMBudgetEnabled {
		return "", nil
	}
	providers := parseProvidersJSON(params.ProvidersJSON)
	rows, err := budgetConn.Query(`
		SELECT provider, base_url, model_name, budget_limit
		FROM cloud_task_model_budget WHERE todo_id = ?
	`, todoID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	type modelRow struct {
		Provider string `json:"provider"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
		InPrice  string `json:"input_price_per_1m"`
		OutPrice string `json:"output_price_per_1m"`
		Limit    string `json:"budget_limit"`
		Spent    string `json:"spent_amount"`
	}
	models := []modelRow{}
	for rows.Next() {
		var provider, baseURL, modelName string
		var limit sql.NullString
		if err := rows.Scan(&provider, &baseURL, &modelName, &limit); err != nil {
			return "", err
		}
		eligible := false
		norm := normalizeBaseURL(baseURL)
		for _, item := range providers {
			if strings.TrimSpace(fmt.Sprintf("%v", item["provider"])) != provider {
				continue
			}
			if normalizeBaseURL(fmt.Sprintf("%v", item["base_url"])) != norm {
				continue
			}
			eligible = isBudgetEligibleProvider(item, true)
			break
		}
		if !eligible {
			continue
		}
		eff, err := resolveEffectiveBudgetLimit(budgetConn, todoID, workspaceID, companyID, provider, baseURL, modelName)
		if err != nil {
			return "", err
		}
		if decimalCmp(eff, "0") <= 0 {
			continue
		}
		inP, outP, err := loadUnitPrices(budgetConn, workspaceID, companyID, provider, baseURL, modelName)
		if err != nil {
			return "", err
		}
		spent := "0"
		usage, err := loadUsage(budgetConn, todoID, provider, baseURL, modelName)
		if err != nil {
			return "", err
		}
		if usage != nil {
			spent = usage.SpentAmount
		}
		models = append(models, modelRow{
			Provider: provider, BaseURL: baseURL, Model: modelName,
			InPrice: inP, OutPrice: outP, Limit: eff, Spent: spent,
		})
	}
	if len(models) == 0 {
		return "", nil
	}
	payload := map[string]any{"currency": "CNY", "models": models}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func resolveFeatureParamsEnvLocal(
	ctx context.Context,
	budgetConn *sql.DB,
	tenantID, workspaceID, taskID, companyID, proxyToken string,
) (*featureParamsResolveResult, error) {
	if companyID == "" {
		companyID = tenantID
	}

	source := "company"
	personalConfigID := ""
	userID := ""
	binding, bindErr := fetchTaskFeatureParamsBindingFn(tenantID, taskID)
	if bindErr != nil {
		logWarn("feature_params: task binding lookup failed, defaulting company: "+bindErr.Error(), "")
	} else if binding != nil {
		if binding.Source != "" {
			source = binding.Source
		}
		personalConfigID = binding.PersonalConfigID
		if binding.WorkspaceID != "" && workspaceID == "" {
			workspaceID = binding.WorkspaceID
		}
		userID = resolveOwnerUserID(companyID, binding.OwnerID)
	}

	cfgRow, meta, err := resolveFeatureParamsHierarchy(
		ctx, companyID, workspaceID, source, personalConfigID, userID,
	)
	if err != nil {
		return nil, err
	}
	cfgRow.Scope = meta.Source
	cfgRow.DisplayName = meta.SourceDisplayName
	cfgRow.ID = meta.SourceConfigID

	env := serializeFeatureParamsEnv(cfgRow)

	if budgetConn != nil {
		policy, err := buildTaskLLMBudgetPolicyEnv(budgetConn, taskID, workspaceID, companyID)
		if err != nil {
			return nil, err
		}
		if policy != "" {
			env[envBudgetPolicy] = policy
		}
	}

	token := strings.TrimSpace(proxyToken)
	providersForUsage := applyProxyRewriteToEnv(env, tenantID, workspaceID, taskID, token)
	if token != "" && taskAIEndpointEnabled() {
		base := taskAIEndpointPublicBase()
		if base != "" {
			env["TASK_AI_ENDPOINT_BASE_URL"] = base
			env["TASK_LLM_PROXY_TOKEN"] = token
		}
	}
	recordTaskApiKeyUsageSafe(tenantID, workspaceID, taskID, providersForUsage)

	writeFeatureParamsSnapshotSafe(taskID, workspaceID, tenantID, meta, env, cfgRow)

	return &featureParamsResolveResult{
		CompanyID: companyID, WorkspaceID: workspaceID, TaskID: taskID,
		Env: env, Source: meta.Source,
	}, nil
}

func taskAIEndpointEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_ENABLED")); v != "" {
		return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	return cfg.TaskAIEndpointEnabled
}

func taskAIEndpointPublicBase() string {
	if env := strings.TrimSpace(os.Getenv("TASK_AI_ENDPOINT_PUBLIC_BASE")); env != "" {
		return strings.TrimRight(env, "/")
	}
	return cfg.TaskAIEndpointPublicBase
}
