package main

import (
	"snowflake"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type workspaceBudgetDefaultRow struct {
	ID               string
	WorkspaceID      string
	CompanyID        string
	Provider         string
	BaseURL          string
	ModelName        string
	InputPricePer1M  string
	OutputPricePer1M string
	BudgetLimit      string
}

type taskModelBudgetRow struct {
	ID                string
	TodoID            string
	WorkspaceID       string
	CompanyID         string
	Provider          string
	BaseURL           string
	ModelName         string
	BudgetLimit       *string
	BudgetLimitSource string
}

func listWorkspaceBudgetDefaults(conn *sql.DB, workspaceID, companyID string) ([]workspaceBudgetDefaultRow, error) {
	rows, err := conn.Query(`
		SELECT id, workspace_id, company_id, provider, base_url, model_name,
			COALESCE(input_price_per_1m,'0'), COALESCE(output_price_per_1m,'0'), COALESCE(budget_limit,'0')
		FROM cloud_workspace_model_budget_default
		WHERE workspace_id = ? AND company_id = ?
		ORDER BY provider, base_url, model_name
	`, workspaceID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]workspaceBudgetDefaultRow, 0)
	for rows.Next() {
		var r workspaceBudgetDefaultRow
		if err := rows.Scan(
			&r.ID, &r.WorkspaceID, &r.CompanyID, &r.Provider, &r.BaseURL, &r.ModelName,
			&r.InputPricePer1M, &r.OutputPricePer1M, &r.BudgetLimit,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func upsertWorkspaceBudgetDefault(conn *sql.DB, row workspaceBudgetDefaultRow) (workspaceBudgetDefaultRow, error) {
	if conn == nil {
		return row, fmt.Errorf("budget db not open")
	}
	workspaceID := strings.TrimSpace(row.WorkspaceID)
	companyID := strings.TrimSpace(row.CompanyID)
	provider := strings.TrimSpace(row.Provider)
	baseURL := strings.TrimSpace(row.BaseURL)
	modelName := strings.TrimSpace(row.ModelName)
	if workspaceID == "" || companyID == "" || provider == "" || modelName == "" {
		return row, fmt.Errorf("workspace_id, company_id, provider, model_name required")
	}
	inPrice := strings.TrimSpace(row.InputPricePer1M)
	if inPrice == "" {
		inPrice = "0"
	}
	outPrice := strings.TrimSpace(row.OutputPricePer1M)
	if outPrice == "" {
		outPrice = "0"
	}
	limit := strings.TrimSpace(row.BudgetLimit)
	if limit == "" {
		limit = "0"
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	var existingID string
	err := conn.QueryRow(`
		SELECT id FROM cloud_workspace_model_budget_default
		WHERE workspace_id = ? AND provider = ? AND base_url = ? AND model_name = ?
		LIMIT 1
	`, workspaceID, provider, baseURL, modelName).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err = conn.Exec(`
			UPDATE cloud_workspace_model_budget_default
			SET company_id=?, input_price_per_1m=?, output_price_per_1m=?, budget_limit=?, updated_at=?
			WHERE id=?
		`, companyID, inPrice, outPrice, limit, now, existingID)
		if err != nil {
			return row, err
		}
		row.ID = existingID
	} else if err != nil && err != sql.ErrNoRows {
		return row, err
	} else {
		id := snowflake.GenerateIDString()
		_, err = conn.Exec(`
			INSERT INTO cloud_workspace_model_budget_default
			(id, workspace_id, company_id, provider, base_url, model_name,
			 input_price_per_1m, output_price_per_1m, budget_limit, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, workspaceID, companyID, provider, baseURL, modelName, inPrice, outPrice, limit, now, now)
		if err != nil {
			return row, err
		}
		row.ID = id
	}
	row.WorkspaceID = workspaceID
	row.CompanyID = companyID
	row.Provider = provider
	row.BaseURL = baseURL
	row.ModelName = modelName
	row.InputPricePer1M = inPrice
	row.OutputPricePer1M = outPrice
	row.BudgetLimit = limit
	return row, nil
}

func deleteWorkspaceBudgetDefault(conn *sql.DB, workspaceID, companyID, provider, baseURL, modelName string) (bool, error) {
	if conn == nil {
		return false, fmt.Errorf("budget db not open")
	}
	res, err := conn.Exec(`
		DELETE FROM cloud_workspace_model_budget_default
		WHERE workspace_id = ? AND company_id = ? AND provider = ? AND base_url = ? AND model_name = ?
	`, workspaceID, companyID, provider, baseURL, modelName)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func listTaskModelBudgets(conn *sql.DB, todoID string) ([]taskModelBudgetRow, error) {
	rows, err := conn.Query(`
		SELECT id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			budget_limit, COALESCE(budget_limit_source,'inherited')
		FROM cloud_task_model_budget
		WHERE todo_id = ?
		ORDER BY provider, base_url, model_name
	`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]taskModelBudgetRow, 0)
	for rows.Next() {
		var r taskModelBudgetRow
		var lim sql.NullString
		if err := rows.Scan(
			&r.ID, &r.TodoID, &r.WorkspaceID, &r.CompanyID, &r.Provider, &r.BaseURL, &r.ModelName,
			&lim, &r.BudgetLimitSource,
		); err != nil {
			return nil, err
		}
		if lim.Valid {
			s := lim.String
			r.BudgetLimit = &s
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func upsertTaskModelBudget(conn *sql.DB, row taskModelBudgetRow) (taskModelBudgetRow, error) {
	if conn == nil {
		return row, fmt.Errorf("budget db not open")
	}
	todoID := strings.TrimSpace(row.TodoID)
	workspaceID := strings.TrimSpace(row.WorkspaceID)
	companyID := strings.TrimSpace(row.CompanyID)
	provider := strings.TrimSpace(row.Provider)
	baseURL := strings.TrimSpace(row.BaseURL)
	modelName := strings.TrimSpace(row.ModelName)
	source := strings.TrimSpace(row.BudgetLimitSource)
	if source == "" {
		source = "inherited"
	}
	if todoID == "" || workspaceID == "" || companyID == "" || provider == "" || modelName == "" {
		return row, fmt.Errorf("todo_id, workspace_id, company_id, provider, model_name required")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var lim any
	if row.BudgetLimit != nil {
		lim = strings.TrimSpace(*row.BudgetLimit)
	} else {
		lim = nil
	}

	var existingID string
	err := conn.QueryRow(`
		SELECT id FROM cloud_task_model_budget
		WHERE todo_id = ? AND provider = ? AND base_url = ? AND model_name = ?
		LIMIT 1
	`, todoID, provider, baseURL, modelName).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err = conn.Exec(`
			UPDATE cloud_task_model_budget
			SET workspace_id=?, company_id=?, budget_limit=?, budget_limit_source=?, updated_at=?
			WHERE id=?
		`, workspaceID, companyID, lim, source, now, existingID)
		if err != nil {
			return row, err
		}
		row.ID = existingID
	} else if err != nil && err != sql.ErrNoRows {
		return row, err
	} else {
		id := snowflake.GenerateIDString()
		_, err = conn.Exec(`
			INSERT INTO cloud_task_model_budget
			(id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			 budget_limit, budget_limit_source, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, todoID, workspaceID, companyID, provider, baseURL, modelName, lim, source, now, now)
		if err != nil {
			return row, err
		}
		row.ID = id
	}
	row.TodoID = todoID
	row.WorkspaceID = workspaceID
	row.CompanyID = companyID
	row.Provider = provider
	row.BaseURL = baseURL
	row.ModelName = modelName
	row.BudgetLimitSource = source
	return row, nil
}

func listTaskModelBudgetUsage(conn *sql.DB, todoID string) ([]budgetUsageRow, error) {
	rows, err := conn.Query(`
		SELECT id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			COALESCE(input_tokens,0), COALESCE(output_tokens,0), COALESCE(spent_amount,'0'), last_reported_at
		FROM cloud_task_model_budget_usage
		WHERE todo_id = ?
		ORDER BY provider, base_url, model_name
	`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]budgetUsageRow, 0)
	for rows.Next() {
		var u budgetUsageRow
		var last sql.NullString
		if err := rows.Scan(
			&u.ID, &u.TodoID, &u.WorkspaceID, &u.CompanyID, &u.Provider, &u.BaseURL, &u.ModelName,
			&u.InputTokens, &u.OutputTokens, &u.SpentAmount, &last,
		); err != nil {
			return nil, err
		}
		if last.Valid {
			s := last.String
			u.LastReportedAt = &s
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func workspaceDefaultToMap(r workspaceBudgetDefaultRow) map[string]any {
	return map[string]any{
		"id":                  r.ID,
		"workspace_id":        r.WorkspaceID,
		"company_id":          r.CompanyID,
		"provider":            r.Provider,
		"base_url":            r.BaseURL,
		"model_name":          r.ModelName,
		"input_price_per_1m":  r.InputPricePer1M,
		"output_price_per_1m": r.OutputPricePer1M,
		"budget_limit":        r.BudgetLimit,
	}
}

func taskBudgetToMap(r taskModelBudgetRow) map[string]any {
	m := map[string]any{
		"id":                  r.ID,
		"todo_id":             r.TodoID,
		"workspace_id":        r.WorkspaceID,
		"company_id":          r.CompanyID,
		"provider":            r.Provider,
		"base_url":            r.BaseURL,
		"model_name":          r.ModelName,
		"budget_limit_source": r.BudgetLimitSource,
	}
	if r.BudgetLimit != nil {
		m["budget_limit"] = *r.BudgetLimit
	} else {
		m["budget_limit"] = nil
	}
	return m
}

func usageToMap(u budgetUsageRow) map[string]any {
	m := map[string]any{
		"id":            u.ID,
		"todo_id":       u.TodoID,
		"workspace_id":  u.WorkspaceID,
		"company_id":    u.CompanyID,
		"provider":      u.Provider,
		"base_url":      u.BaseURL,
		"model_name":    u.ModelName,
		"input_tokens":  u.InputTokens,
		"output_tokens": u.OutputTokens,
		"spent_amount":  u.SpentAmount,
	}
	if u.LastReportedAt != nil {
		m["last_reported_at"] = *u.LastReportedAt
	} else {
		m["last_reported_at"] = nil
	}
	return m
}
