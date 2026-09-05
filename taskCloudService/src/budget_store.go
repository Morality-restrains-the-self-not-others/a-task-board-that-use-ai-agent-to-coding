package main

import (
	"snowflake"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func loadUnitPrices(conn *sql.DB, workspaceID, companyID, provider, baseURL, modelName string) (inPrice, outPrice string, err error) {
	norm := normalizeBaseURL(baseURL)
	rows, err := conn.Query(`
		SELECT COALESCE(input_price_per_1m,'0'), COALESCE(output_price_per_1m,'0'), base_url
		FROM cloud_workspace_model_budget_default
		WHERE workspace_id = ? AND company_id = ? AND provider = ? AND model_name = ?
	`, workspaceID, companyID, provider, modelName)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	for rows.Next() {
		var inRaw, outRaw, storedBase string
		if err := rows.Scan(&inRaw, &outRaw, &storedBase); err != nil {
			return "", "", err
		}
		if normalizeBaseURL(storedBase) == norm || storedBase == baseURL {
			return strings.TrimSpace(inRaw), strings.TrimSpace(outRaw), nil
		}
	}
	return "0", "0", nil
}

func loadUsageByID(conn *sql.DB, usageID string) (*budgetUsageRow, error) {
	row := conn.QueryRow(`
		SELECT id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			COALESCE(input_tokens,0), COALESCE(output_tokens,0), COALESCE(spent_amount,'0'), last_reported_at
		FROM cloud_task_model_budget_usage WHERE id = ? LIMIT 1
	`, usageID)
	var u budgetUsageRow
	var last sql.NullString
	if err := row.Scan(
		&u.ID, &u.TodoID, &u.WorkspaceID, &u.CompanyID, &u.Provider, &u.BaseURL, &u.ModelName,
		&u.InputTokens, &u.OutputTokens, &u.SpentAmount, &last,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if last.Valid {
		s := last.String
		u.LastReportedAt = &s
	}
	return &u, nil
}

func loadUsage(conn *sql.DB, todoID, provider, baseURL, modelName string) (*budgetUsageRow, error) {
	rows, err := conn.Query(`
		SELECT id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			COALESCE(input_tokens,0), COALESCE(output_tokens,0), COALESCE(spent_amount,'0'), last_reported_at
		FROM cloud_task_model_budget_usage
		WHERE todo_id = ? AND provider = ? AND model_name = ?
	`, todoID, provider, modelName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	norm := normalizeBaseURL(baseURL)
	for rows.Next() {
		var u budgetUsageRow
		var last sql.NullString
		if err := rows.Scan(
			&u.ID, &u.TodoID, &u.WorkspaceID, &u.CompanyID, &u.Provider, &u.BaseURL, &u.ModelName,
			&u.InputTokens, &u.OutputTokens, &u.SpentAmount, &last,
		); err != nil {
			return nil, err
		}
		if normalizeBaseURL(u.BaseURL) == norm || u.BaseURL == baseURL {
			if last.Valid {
				s := last.String
				u.LastReportedAt = &s
			}
			return &u, nil
		}
	}
	return nil, nil
}

func recordModelBudgetUsageDelta(
	conn *sql.DB,
	todoID, workspaceID, companyID string,
	item budgetRecordItem,
) (*budgetUsageRow, bool, error) {
	if conn == nil {
		return nil, false, fmt.Errorf("budget db not open")
	}
	idem := strings.TrimSpace(item.IdempotencyKey)
	if idem == "" {
		return nil, false, fmt.Errorf("idempotency_key required")
	}
	provider := strings.TrimSpace(item.Provider)
	modelName := strings.TrimSpace(item.ModelName)
	baseURL := strings.TrimSpace(item.BaseURL)
	if provider == "" || modelName == "" {
		return nil, false, fmt.Errorf("provider and model_name required")
	}
	if companyID == "" {
		return nil, false, errBudgetCompanyMissing
	}

	// 幂等与读价在事务外完成，避免单连接/WAL 下 tx 内再查同一 DB 死锁
	var existingUsageID string
	err := conn.QueryRow(
		`SELECT usage_id FROM cloud_task_model_budget_usage_idempotency WHERE idempotency_key = ?`,
		idem,
	).Scan(&existingUsageID)
	if err == nil && existingUsageID != "" {
		usage, loadErr := loadUsageByID(conn, existingUsageID)
		if loadErr != nil {
			return nil, false, loadErr
		}
		if usage != nil {
			return usage, false, nil
		}
		return nil, false, fmt.Errorf("idempotent usage missing: %s", existingUsageID)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	inPrice, outPrice, err := loadUnitPrices(conn, workspaceID, companyID, provider, baseURL, modelName)
	if err != nil {
		return nil, false, err
	}

	usage, err := loadUsage(conn, todoID, provider, baseURL, modelName)
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	inDelta := item.InputTokensDelta
	if inDelta < 0 {
		inDelta = 0
	}
	outDelta := item.OutputTokensDelta
	if outDelta < 0 {
		outDelta = 0
	}

	tx, err := conn.Begin()
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	// 事务内再查一次幂等，防并发双写
	existingUsageID = ""
	err = tx.QueryRow(
		`SELECT usage_id FROM cloud_task_model_budget_usage_idempotency WHERE idempotency_key = ?`,
		idem,
	).Scan(&existingUsageID)
	if err == nil && existingUsageID != "" {
		_ = tx.Rollback()
		usage, loadErr := loadUsageByID(conn, existingUsageID)
		if loadErr != nil {
			return nil, false, loadErr
		}
		if usage != nil {
			return usage, false, nil
		}
		return nil, false, fmt.Errorf("idempotent usage missing: %s", existingUsageID)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	if usage == nil {
		id := snowflake.GenerateIDString()
		spent, cerr := calculateSpentCNY(inDelta, outDelta, inPrice, outPrice)
		if cerr != nil {
			return nil, false, cerr
		}
		_, err = tx.Exec(`
			INSERT INTO cloud_task_model_budget_usage
			(id, todo_id, workspace_id, company_id, provider, base_url, model_name,
			 input_tokens, output_tokens, spent_amount, last_reported_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, todoID, workspaceID, companyID, provider, baseURL, modelName,
			inDelta, outDelta, spent, now, now, now)
		if err != nil {
			return nil, false, err
		}
		usage = &budgetUsageRow{
			ID: id, TodoID: todoID, WorkspaceID: workspaceID, CompanyID: companyID,
			Provider: provider, BaseURL: baseURL, ModelName: modelName,
			InputTokens: inDelta, OutputTokens: outDelta, SpentAmount: spent, LastReportedAt: &now,
		}
	} else {
		newIn := usage.InputTokens
		if newIn < 0 {
			newIn = 0
		}
		newOut := usage.OutputTokens
		if newOut < 0 {
			newOut = 0
		}
		newIn += inDelta
		newOut += outDelta
		spent, cerr := calculateSpentCNY(newIn, newOut, inPrice, outPrice)
		if cerr != nil {
			return nil, false, cerr
		}
		_, err = tx.Exec(`
			UPDATE cloud_task_model_budget_usage
			SET input_tokens=?, output_tokens=?, spent_amount=?, last_reported_at=?, updated_at=?
			WHERE id=?
		`, newIn, newOut, spent, now, now, usage.ID)
		if err != nil {
			return nil, false, err
		}
		usage.InputTokens = newIn
		usage.OutputTokens = newOut
		usage.SpentAmount = spent
		usage.LastReportedAt = &now
	}

	idemID := snowflake.GenerateIDString()
	_, err = tx.Exec(`
		INSERT INTO cloud_task_model_budget_usage_idempotency
		(id, idempotency_key, usage_id, created_at) VALUES (?, ?, ?, ?)
	`, idemID, idem, usage.ID, now)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return usage, true, nil
}


