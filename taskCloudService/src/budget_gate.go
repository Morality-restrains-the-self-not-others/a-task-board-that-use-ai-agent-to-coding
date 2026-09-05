package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type tenantFeatureParamsRow struct {
	ID               string
	CompanyID        string
	ProvidersJSON    string
	LLMBudgetEnabled bool
	AgentModel       string
	AgentProvider    string
	AgentMaxSteps    int
	SummaryModel     string
	SummaryProvider  string
	ExtraEnvJSON     string
}

func parseProvidersJSON(raw string) []map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func isBudgetEligibleProvider(entry map[string]any, llmBudgetEnabled bool) bool {
	if !llmBudgetEnabled {
		return false
	}
	provider := strings.TrimSpace(fmt.Sprintf("%v", entry["provider"]))
	useSub, _ := entry["use_sub_token"].(bool)
	if !useSub {
		switch v := entry["use_sub_token"].(type) {
		case float64:
			useSub = v != 0
		case string:
			useSub = strings.EqualFold(v, "true") || v == "1"
		}
	}
	budgetEnabled := false
	if useSub {
		switch v := entry["budget_enabled"].(type) {
		case bool:
			budgetEnabled = v
		case float64:
			budgetEnabled = v != 0
		case string:
			budgetEnabled = strings.EqualFold(v, "true") || v == "1"
		}
	}
	return provider != "" && provider != "<nil>" && useSub && budgetEnabled
}

func requireLLMBudgetEnabled(companyID string) error {
	params, err := loadTenantFeatureParams(companyID)
	if err != nil {
		return err
	}
	if params == nil || !params.LLMBudgetEnabled {
		return errBudgetDisabled
	}
	return nil
}

func requireBudgetEligibleProvider(companyID, provider, baseURL string) error {
	if err := requireLLMBudgetEnabled(companyID); err != nil {
		return err
	}
	params, err := loadTenantFeatureParams(companyID)
	if err != nil {
		return err
	}
	providers := parseProvidersJSON(params.ProvidersJSON)
	norm := normalizeBaseURL(baseURL)
	for _, item := range providers {
		if strings.TrimSpace(fmt.Sprintf("%v", item["provider"])) != strings.TrimSpace(provider) {
			continue
		}
		itemBase := normalizeBaseURL(fmt.Sprintf("%v", item["base_url"]))
		if itemBase != norm {
			continue
		}
		if isBudgetEligibleProvider(item, true) {
			return nil
		}
	}
	return errBudgetProviderInelig
}

func resolveEffectiveBudgetLimit(budgetConn *sql.DB, todoID, workspaceID, companyID, provider, baseURL, modelName string) (string, error) {
	norm := normalizeBaseURL(baseURL)
	rows, err := budgetConn.Query(`
		SELECT budget_limit, base_url FROM cloud_task_model_budget
		WHERE todo_id = ? AND provider = ? AND model_name = ?
	`, todoID, provider, modelName)
	if err != nil {
		return "0", err
	}
	defer rows.Close()
	var taskLimit sql.NullString
	for rows.Next() {
		var lim sql.NullString
		var b string
		if err := rows.Scan(&lim, &b); err != nil {
			return "0", err
		}
		if normalizeBaseURL(b) == norm || b == baseURL {
			taskLimit = lim
			break
		}
	}
	if taskLimit.Valid && strings.TrimSpace(taskLimit.String) != "" {
		return strings.TrimSpace(taskLimit.String), nil
	}

	drows, err := budgetConn.Query(`
		SELECT budget_limit, base_url FROM cloud_workspace_model_budget_default
		WHERE workspace_id = ? AND company_id = ? AND provider = ? AND model_name = ?
	`, workspaceID, companyID, provider, modelName)
	if err != nil {
		return "0", err
	}
	defer drows.Close()
	for drows.Next() {
		var lim string
		var b string
		if err := drows.Scan(&lim, &b); err != nil {
			return "0", err
		}
		if normalizeBaseURL(b) == norm || b == baseURL {
			return strings.TrimSpace(lim), nil
		}
	}
	return "0", nil
}

func checkBudgetGate(
	budgetConn *sql.DB,
	todoID, workspaceID, companyID, provider, baseURL, modelName string,
) (map[string]any, error) {
	params, err := loadTenantFeatureParams(companyID)
	if err != nil {
		return nil, err
	}
	providers := []map[string]any{}
	if params != nil {
		providers = parseProvidersJSON(params.ProvidersJSON)
	}
	eligible := false
	norm := normalizeBaseURL(baseURL)
	for _, item := range providers {
		if strings.TrimSpace(fmt.Sprintf("%v", item["provider"])) != strings.TrimSpace(provider) {
			continue
		}
		if normalizeBaseURL(fmt.Sprintf("%v", item["base_url"])) != norm {
			continue
		}
		eligible = isBudgetEligibleProvider(item, true)
		break
	}
	if !eligible {
		return map[string]any{"allowed": true, "currency": "CNY"}, nil
	}
	limit, err := resolveEffectiveBudgetLimit(budgetConn, todoID, workspaceID, companyID, provider, baseURL, modelName)
	if err != nil {
		return nil, err
	}
	usage, err := loadUsage(budgetConn, todoID, provider, baseURL, modelName)
	if err != nil {
		return nil, err
	}
	spent := "0"
	if usage != nil {
		spent = usage.SpentAmount
	}
	if decimalCmp(limit, "0") > 0 && decimalCmp(spent, limit) >= 0 {
		return map[string]any{
			"allowed":      false,
			"code":         "budget_exhausted",
			"message":      fmt.Sprintf("模型 %s 预算已用尽", modelName),
			"spent_amount": spent,
			"budget_limit": limit,
			"currency":     "CNY",
		}, nil
	}
	return map[string]any{
		"allowed":      true,
		"spent_amount": spent,
		"budget_limit": limit,
		"currency":     "CNY",
	}, nil
}
