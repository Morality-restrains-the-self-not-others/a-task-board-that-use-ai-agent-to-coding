package main

import (
	"snowflake"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

func listPersonalFeatureParamsLocal(userID string) ([]map[string]any, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(`
		SELECT id, user_id, company_id, name, providers, agent_model, agent_model_provider,
			agent_max_steps, summary_model, summary_model_provider, extra_env_vars, created_at, updated_at
		FROM cloud_personal_feature_params_config
		WHERE user_id = ? ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		item, scanErr := scanPersonalParamsRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPersonalParamsRow(rows *sql.Rows) (map[string]any, error) {
	var (
		id, uid, companyID, name, providers, agentModel, agentProvider string
		summaryModel, summaryProvider, extraJSON, createdAt, updatedAt string
		maxSteps                                                         int
	)
	if err := rows.Scan(
		&id, &uid, &companyID, &name, &providers, &agentModel, &agentProvider,
		&maxSteps, &summaryModel, &summaryProvider, &extraJSON, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":                     id,
		"name":                   name,
		"company_id":             companyID,
		"providers":              fpParseJSONArray(providers),
		"agent_model":            agentModel,
		"agent_model_provider":   agentProvider,
		"agent_max_steps":        maxSteps,
		"summary_model":          summaryModel,
		"summary_model_provider": summaryProvider,
		"extra_env_vars":         fpParseJSONArray(extraJSON),
		"created_at":             createdAt,
		"updated_at":             updatedAt,
	}, nil
}

func queryPersonalFeatureParamsRow(configID, userID string) (map[string]any, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, err
	}
	row := conn.QueryRow(`
		SELECT id, user_id, company_id, name, providers, agent_model, agent_model_provider,
			agent_max_steps, summary_model, summary_model_provider, extra_env_vars, created_at, updated_at
		FROM cloud_personal_feature_params_config WHERE id = ? AND user_id = ?`, configID, userID,
	)
	var (
		id, uid, companyID, name, providers, agentModel, agentProvider string
		summaryModel, summaryProvider, extraJSON, createdAt, updatedAt string
		maxSteps                                                         int
	)
	err = row.Scan(
		&id, &uid, &companyID, &name, &providers, &agentModel, &agentProvider,
		&maxSteps, &summaryModel, &summaryProvider, &extraJSON, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": id, "name": name, "company_id": companyID,
		"providers": fpParseJSONArray(providers), "agent_model": agentModel,
		"agent_model_provider": agentProvider, "agent_max_steps": maxSteps,
		"summary_model": summaryModel, "summary_model_provider": summaryProvider,
		"extra_env_vars": fpParseJSONArray(extraJSON),
		"created_at": createdAt, "updated_at": updatedAt,
	}, nil
}

func upsertPersonalFeatureParamsLocal(body map[string]interface{}) (map[string]any, int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	userID := strings.TrimSpace(strField(body, "user_id"))
	if userID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("user_id required")
	}
	configID := strings.TrimSpace(strField(body, "id"))
	if configID == "" {
		configID = strings.TrimSpace(strField(body, "config_id"))
	}
	now := fpNowUTC()
	status := http.StatusOK

	if configID != "" {
		existing, err := queryPersonalFeatureParamsRow(configID, userID)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		if existing == nil {
			return nil, http.StatusNotFound, fmt.Errorf("not found")
		}
		name := strField(existing, "name")
		if v, ok := body["name"]; ok {
			name = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		providersJSON := coalesceJSONString(existing["providers"])
		if v, ok := body["providers"]; ok {
			providersJSON = coalesceJSONString(v)
		}
		extraJSON := coalesceJSONString(existing["extra_env_vars"])
		if v, ok := body["extra_env_vars"]; ok {
			extraJSON = coalesceJSONString(v)
		}
		agentModel := strField(existing, "agent_model")
		if v, ok := body["agent_model"]; ok {
			agentModel = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		agentProvider := strField(existing, "agent_model_provider")
		if v, ok := body["agent_model_provider"]; ok {
			agentProvider = strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
		}
		summaryModel := strField(existing, "summary_model")
		if v, ok := body["summary_model"]; ok {
			summaryModel = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		summaryProvider := strField(existing, "summary_model_provider")
		if v, ok := body["summary_model_provider"]; ok {
			summaryProvider = strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
		}
		maxSteps := intFromAny(existing["agent_max_steps"], 200)
		if v, ok := body["agent_max_steps"]; ok {
			maxSteps = intFromAny(v, maxSteps)
		}
		llmBudget := fpBoolInt(parseBoolish(body["llm_budget_enabled"], false))
		_, err = conn.Exec(`
			UPDATE cloud_personal_feature_params_config SET
				name = ?, providers = ?, agent_model = ?, agent_model_provider = ?,
				agent_max_steps = ?, summary_model = ?, summary_model_provider = ?,
				llm_budget_enabled = ?, extra_env_vars = ?, updated_at = ?
			WHERE id = ? AND user_id = ?`,
			name, providersJSON, agentModel, agentProvider, maxSteps, summaryModel, summaryProvider,
			llmBudget, extraJSON, now, configID, userID,
		)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	} else {
		companyID := strings.TrimSpace(strField(body, "company_id"))
		name := strings.TrimSpace(strField(body, "name"))
		if companyID == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("company_id required")
		}
		if name == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("name required")
		}
		configID = snowflake.GenerateIDString()
		maxSteps := intFromAny(body["agent_max_steps"], 200)
		llmBudget := fpBoolInt(parseBoolish(body["llm_budget_enabled"], false))
		_, err = conn.Exec(`
			INSERT INTO cloud_personal_feature_params_config (
				id, user_id, company_id, name, providers, agent_model, agent_model_provider,
				agent_max_steps, summary_model, summary_model_provider, llm_budget_enabled,
				extra_env_vars, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			configID, userID, companyID, name, coalesceJSONString(body["providers"]),
			strField(body, "agent_model"), strField(body, "agent_model_provider"), maxSteps,
			strField(body, "summary_model"), strField(body, "summary_model_provider"),
			llmBudget, coalesceJSONString(body["extra_env_vars"]), now, now,
		)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		status = http.StatusCreated
	}
	params, err := queryPersonalFeatureParamsRow(configID, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return params, status, nil
}

func deletePersonalFeatureParamsLocal(configID, userID string) (int, error) {
	conn, err := featureParamsConn()
	if err != nil {
		return http.StatusBadGateway, err
	}
	res, err := conn.Exec(
		`DELETE FROM cloud_personal_feature_params_config WHERE id = ? AND user_id = ?`,
		configID, userID,
	)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return http.StatusNotFound, fmt.Errorf("not found")
	}
	return http.StatusOK, nil
}
