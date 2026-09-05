package main

import (
	"snowflake"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func fpNowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.999999")
}

func featureParamsConn() (*sql.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("task cloud db not open")
	}
	return db, nil
}

func fpParseJSONArray(raw string) []any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []any{}
	}
	var out []any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []any{}
	}
	return out
}

func fpBoolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func coalesceJSONString(v any) string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return "[]"
		}
		return t
	default:
		if t == nil {
			return "[]"
		}
		b, err := json.Marshal(t)
		if err != nil {
			return "[]"
		}
		return string(b)
	}
}

func queryTenantFeatureParamsRow(companyID string) (*tenantFeatureParamsRow, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, err
	}
	row := &tenantFeatureParamsRow{}
	var llmBudget int
	err = conn.QueryRow(`
		SELECT id, company_id, providers, agent_model, agent_model_provider, agent_max_steps,
			summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars
		FROM cloud_tenant_feature_params WHERE company_id = ?`, companyID,
	).Scan(
		&row.ID, &row.CompanyID, &row.ProvidersJSON, &row.AgentModel, &row.AgentProvider,
		&row.AgentMaxSteps, &row.SummaryModel, &row.SummaryProvider, &llmBudget, &row.ExtraEnvJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.LLMBudgetEnabled = llmBudget != 0
	return row, nil
}

func tenantParamsMap(row *tenantFeatureParamsRow) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":                     row.ID,
		"company_id":             row.CompanyID,
		"providers":              fpParseJSONArray(row.ProvidersJSON),
		"llm_budget_enabled":     row.LLMBudgetEnabled,
		"agent_model":            row.AgentModel,
		"agent_model_provider":   row.AgentProvider,
		"agent_max_steps":        row.AgentMaxSteps,
		"summary_model":          row.SummaryModel,
		"summary_model_provider": row.SummaryProvider,
		"extra_env_vars":         fpParseJSONArray(row.ExtraEnvJSON),
	}
}

func upsertTenantFeatureParamsLocal(body map[string]interface{}) (map[string]any, int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	companyID := strings.TrimSpace(strField(body, "company_id"))
	if companyID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("company_id required")
	}
	maxSteps := intFromAny(body["agent_max_steps"], 200)
	providersJSON := coalesceJSONString(body["providers"])
	extraJSON := coalesceJSONString(body["extra_env_vars"])
	llmBudget := fpBoolInt(parseBoolish(body["llm_budget_enabled"], false))
	now := fpNowUTC()

	var existingID string
	_ = conn.QueryRow(`SELECT id FROM cloud_tenant_feature_params WHERE company_id = ?`, companyID).Scan(&existingID)
	id := existingID
	if id == "" {
		id = snowflake.GenerateIDString()
		_, err = conn.Exec(`
			INSERT INTO cloud_tenant_feature_params (
				id, company_id, providers, agent_model, agent_model_provider, agent_max_steps,
				summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, companyID, providersJSON,
			strField(body, "agent_model"), strField(body, "agent_model_provider"), maxSteps,
			strField(body, "summary_model"), strField(body, "summary_model_provider"),
			llmBudget, extraJSON, now, now,
		)
	} else {
		_, err = conn.Exec(`
			UPDATE cloud_tenant_feature_params SET
				providers = ?, agent_model = ?, agent_model_provider = ?, agent_max_steps = ?,
				summary_model = ?, summary_model_provider = ?, llm_budget_enabled = ?,
				extra_env_vars = ?, updated_at = ?
			WHERE company_id = ?`,
			providersJSON,
			strField(body, "agent_model"), strField(body, "agent_model_provider"), maxSteps,
			strField(body, "summary_model"), strField(body, "summary_model_provider"),
			llmBudget, extraJSON, now, companyID,
		)
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	row, err := queryTenantFeatureParamsRow(companyID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return tenantParamsMap(row), http.StatusOK, nil
}

func queryWorkspaceFeatureParamsLoad(workspaceID string) (*workspaceFeatureParamsLoad, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, err
	}
	var (
		id, companyID, providers, agentModel, agentProvider string
		summaryModel, summaryProvider, extraJSON            string
		useCompany, llmBudget, maxSteps                     int
	)
	err = conn.QueryRow(`
		SELECT id, company_id, use_company_default, providers, agent_model, agent_model_provider,
			agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars
		FROM cloud_workspace_feature_params WHERE workspace_id = ?`, workspaceID,
	).Scan(
		&id, &companyID, &useCompany, &providers, &agentModel, &agentProvider,
		&maxSteps, &summaryModel, &summaryProvider, &llmBudget, &extraJSON,
	)
	if err == sql.ErrNoRows {
		return &workspaceFeatureParamsLoad{}, nil
	}
	if err != nil {
		return nil, err
	}
	load := &workspaceFeatureParamsLoad{
		Found:             true,
		UseCompanyDefault: useCompany != 0,
		Params: map[string]any{
			"id":                     id,
			"providers":              fpParseJSONArray(providers),
			"agent_model":            agentModel,
			"agent_model_provider":   agentProvider,
			"agent_max_steps":        maxSteps,
			"summary_model":          summaryModel,
			"summary_model_provider": summaryProvider,
			"extra_env_vars":         fpParseJSONArray(extraJSON),
		},
	}
	return load, nil
}

func upsertWorkspaceFeatureParamsLocal(body map[string]interface{}) (map[string]any, int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	companyID := strings.TrimSpace(strField(body, "company_id"))
	if workspaceID == "" || companyID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("workspace_id and company_id required")
	}
	useDefault := parseBoolish(body["use_company_default"], true)
	now := fpNowUTC()

	var existingID string
	_ = conn.QueryRow(`SELECT id FROM cloud_workspace_feature_params WHERE workspace_id = ?`, workspaceID).Scan(&existingID)
	id := existingID
	if id == "" {
		id = snowflake.GenerateIDString()
	}

	providersJSON := "[]"
	extraJSON := coalesceJSONString(body["extra_env_vars"])
	agentModel := ""
	agentProvider := ""
	summaryModel := ""
	summaryProvider := ""
	maxSteps := 200
	llmBudget := 0
	if !useDefault {
		providersJSON = coalesceJSONString(body["providers"])
		agentModel = strField(body, "agent_model")
		agentProvider = strField(body, "agent_model_provider")
		summaryModel = strField(body, "summary_model")
		summaryProvider = strField(body, "summary_model_provider")
		maxSteps = intFromAny(body["agent_max_steps"], 200)
		llmBudget = fpBoolInt(parseBoolish(body["llm_budget_enabled"], false))
	}

	if existingID == "" {
		_, err = conn.Exec(`
			INSERT INTO cloud_workspace_feature_params (
				id, workspace_id, company_id, use_company_default, providers, agent_model,
				agent_model_provider, agent_max_steps, summary_model, summary_model_provider,
				llm_budget_enabled, extra_env_vars, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, workspaceID, companyID, fpBoolInt(useDefault), providersJSON, agentModel, agentProvider,
			maxSteps, summaryModel, summaryProvider, llmBudget, extraJSON, now, now,
		)
	} else {
		_, err = conn.Exec(`
			UPDATE cloud_workspace_feature_params SET
				company_id = ?, use_company_default = ?, providers = ?, agent_model = ?,
				agent_model_provider = ?, agent_max_steps = ?, summary_model = ?,
				summary_model_provider = ?, llm_budget_enabled = ?, extra_env_vars = ?, updated_at = ?
			WHERE workspace_id = ?`,
			companyID, fpBoolInt(useDefault), providersJSON, agentModel, agentProvider,
			maxSteps, summaryModel, summaryProvider, llmBudget, extraJSON, now, workspaceID,
		)
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	load, err := queryWorkspaceFeatureParamsLoad(workspaceID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return load.Params, http.StatusOK, nil
}
