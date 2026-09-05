package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func handleInternalBudgetRoutes(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/budget/")
	path = strings.Trim(path, "/")
	switch {
	case path == "record-usage" || path == "record-usage-batch":
		if r.Method != http.MethodPost {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalBudgetRecordUsage(w, r)
	case path == "reserve-or-deny":
		if r.Method != http.MethodPost {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalBudgetReserveOrDeny(w, r)
	case path == "workspace-defaults":
		handleInternalWorkspaceBudgetDefaults(w, r)
	case path == "workspace-defaults/upsert":
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalWorkspaceBudgetDefaultsUpsert(w, r)
	case path == "workspace-defaults/delete":
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalWorkspaceBudgetDefaultsDelete(w, r)
	case path == "task-budgets":
		handleInternalTaskModelBudgets(w, r)
	case path == "task-budgets/upsert":
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalTaskModelBudgetsUpsert(w, r)
	case path == "task-usage":
		if r.Method != http.MethodGet {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalTaskModelBudgetUsage(w, r)
	case path == "tenant-permissions":
		handleInternalTenantBudgetPermissions(w, r)
	case path == "tenant-permissions/upsert":
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleInternalTenantBudgetPermissionsUpsert(w, r)
	case path == "tenant-permissions/evaluate-raise":
		handleInternalTenantBudgetPermissionsEvaluateRaise(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func requireBudgetDB(w http.ResponseWriter) *sql.DB {
	budgetConn := getBudgetDB()
	if budgetConn == nil {
		writeErrorJSON(w, nil, http.StatusServiceUnavailable, "budget db unavailable")
		return nil
	}
	return budgetConn
}

func handleInternalWorkspaceBudgetDefaults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	q := r.URL.Query()
	workspaceID := strings.TrimSpace(q.Get("workspace_id"))
	companyID := strings.TrimSpace(q.Get("company_id"))
	if workspaceID == "" || companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "workspace_id, company_id required")
		return
	}
	rows, err := listWorkspaceBudgetDefaults(budgetConn, workspaceID, companyID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceDefaultToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
}

func handleInternalWorkspaceBudgetDefaultsUpsert(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	workspaceID := strField(body, "workspace_id")
	companyID := strField(body, "company_id")
	if workspaceID == "" || companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "workspace_id, company_id required")
		return
	}
	rawItems, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
		return
	}
	out := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, isObj := raw.(map[string]any)
		if !isObj {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		row, err := upsertWorkspaceBudgetDefault(budgetConn, workspaceBudgetDefaultRow{
			WorkspaceID:      workspaceID,
			CompanyID:        companyID,
			Provider:         strField(item, "provider"),
			BaseURL:          strField(item, "base_url"),
			ModelName:        strField(item, "model_name"),
			InputPricePer1M:  strField(item, "input_price_per_1m"),
			OutputPricePer1M: strField(item, "output_price_per_1m"),
			BudgetLimit:      strField(item, "budget_limit"),
		})
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
			return
		}
		out = append(out, workspaceDefaultToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": "CNY"})
}

func handleInternalWorkspaceBudgetDefaultsDelete(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	workspaceID := strField(body, "workspace_id")
	companyID := strField(body, "company_id")
	if workspaceID == "" || companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "workspace_id, company_id required")
		return
	}
	rawItems, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
		return
	}
	deleted := 0
	for _, raw := range rawItems {
		item, isObj := raw.(map[string]any)
		if !isObj {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		ok, err := deleteWorkspaceBudgetDefault(
			budgetConn, workspaceID, companyID,
			strField(item, "provider"), strField(item, "base_url"), strField(item, "model_name"),
		)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if ok {
			deleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

func handleInternalTaskModelBudgets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	todoID := strings.TrimSpace(r.URL.Query().Get("todo_id"))
	if todoID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "todo_id required")
		return
	}
	rows, err := listTaskModelBudgets(budgetConn, todoID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, taskBudgetToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
}

func handleInternalTaskModelBudgetsUpsert(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	todoID := strField(body, "todo_id")
	workspaceID := strField(body, "workspace_id")
	companyID := strField(body, "company_id")
	if todoID == "" || workspaceID == "" || companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "todo_id, workspace_id, company_id required")
		return
	}
	rawItems, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
		return
	}
	out := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, isObj := raw.(map[string]any)
		if !isObj {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		var limPtr *string
		if v, exists := item["budget_limit"]; exists && v != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				limPtr = &s
			}
		}
		source := strField(item, "budget_limit_source")
		row, err := upsertTaskModelBudget(budgetConn, taskModelBudgetRow{
			TodoID:            todoID,
			WorkspaceID:       workspaceID,
			CompanyID:         companyID,
			Provider:          strField(item, "provider"),
			BaseURL:           strField(item, "base_url"),
			ModelName:         strField(item, "model_name"),
			BudgetLimit:       limPtr,
			BudgetLimitSource: source,
		})
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
			return
		}
		out = append(out, taskBudgetToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": "CNY"})
}

func handleInternalTaskModelBudgetUsage(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	todoID := strings.TrimSpace(r.URL.Query().Get("todo_id"))
	if todoID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "todo_id required")
		return
	}
	rows, err := listTaskModelBudgetUsage(budgetConn, todoID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, usageToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
}

func parseTokenDelta(v any) (int64, error) {
	switch t := v.(type) {
	case nil:
		return 0, nil
	case float64:
		return int64(t), nil
	case int:
		return int64(t), nil
	case int64:
		return t, nil
	case string:
		if strings.TrimSpace(t) == "" {
			return 0, nil
		}
		return strconv.ParseInt(strings.TrimSpace(t), 10, 64)
	default:
		return strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", t)), 10, 64)
	}
}

func handleInternalBudgetRecordUsage(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	if tenantID == "" || workspaceID == "" || taskID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id, workspace_id, task_id required")
		return
	}

	budgetConn := getBudgetDB()
	if budgetConn == nil {
		writeErrorJSON(w, r, http.StatusServiceUnavailable, "budget db unavailable")
		return
	}

	// Single-item commit-usage shape (taskAIEndPoint)
	if _, hasItems := body["items"]; !hasItems {
		provider := strField(body, "provider")
		baseURL := strField(body, "base_url")
		modelName := strField(body, "model_name")
		idem := strField(body, "idempotency_key")
		if idem == "" {
			idem = strField(body, "request_id")
		}
		inDelta, err := parseTokenDelta(body["input_tokens"])
		if err != nil {
			inDelta, err = parseTokenDelta(body["input_tokens_delta"])
		}
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid token counts")
			return
		}
		outDelta, err := parseTokenDelta(body["output_tokens"])
		if err != nil {
			outDelta, err = parseTokenDelta(body["output_tokens_delta"])
		}
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid token counts")
			return
		}
		if provider == "" || modelName == "" || idem == "" {
			writeErrorJSON(w, r, http.StatusBadRequest, "provider、model_name、idempotency_key 必填")
			return
		}
		// commit-usage（taskAIEndPoint）不重复做 provider gate；与 Django commit_budget_usage 一致
		usage, created, err := recordModelBudgetUsageDelta(budgetConn, taskID, workspaceID, tenantID, budgetRecordItem{
			Provider: provider, BaseURL: baseURL, ModelName: modelName,
			InputTokensDelta: inDelta, OutputTokensDelta: outDelta, IdempotencyKey: idem,
		})
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"usage_id":     usage.ID,
			"created":      created,
			"spent_amount": usage.SpentAmount,
			"currency":     "CNY",
		})
		return
	}

	rawItems, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
		return
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		writeBudgetGateError(w, err)
		return
	}

	results := make([]budgetRecordResult, 0, len(rawItems))
	for _, raw := range rawItems {
		item, isObj := raw.(map[string]any)
		if !isObj {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		baseURL := strings.TrimSpace(fmt.Sprintf("%v", item["base_url"]))
		modelName := strings.TrimSpace(fmt.Sprintf("%v", item["model_name"]))
		idem := strings.TrimSpace(fmt.Sprintf("%v", item["idempotency_key"]))
		if provider == "" || provider == "<nil>" || modelName == "" || modelName == "<nil>" || idem == "" || idem == "<nil>" {
			writeErrorJSON(w, r, http.StatusBadRequest, "provider、model_name、idempotency_key 必填")
			return
		}
		if baseURL == "<nil>" {
			baseURL = ""
		}
		if err := requireBudgetEligibleProvider(tenantID, provider, baseURL); err != nil {
			writeBudgetGateError(w, err)
			return
		}
		inDelta, err := parseTokenDelta(item["input_tokens_delta"])
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid token counts")
			return
		}
		outDelta, err := parseTokenDelta(item["output_tokens_delta"])
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid token counts")
			return
		}
		usage, created, err := recordModelBudgetUsageDelta(budgetConn, taskID, workspaceID, tenantID, budgetRecordItem{
			Provider: provider, BaseURL: baseURL, ModelName: modelName,
			InputTokensDelta: inDelta, OutputTokensDelta: outDelta, IdempotencyKey: idem,
		})
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		spent := usage.SpentAmount
		if spent == "" {
			spent = "0"
		}
		results = append(results, budgetRecordResult{
			Provider: provider, ModelName: modelName,
			SpentAmount: spent, Created: created, UsageID: usage.ID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": results, "currency": "CNY"})
}

func handleInternalBudgetReserveOrDeny(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	provider := strField(body, "provider")
	baseURL := strField(body, "base_url")
	modelName := strField(body, "model_name")
	if tenantID == "" || workspaceID == "" || taskID == "" || provider == "" || baseURL == "" || modelName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "missing fields"})
		return
	}
	budgetConn := getBudgetDB()
	if budgetConn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"message": "budget db unavailable"})
		return
	}
	decision, err := checkBudgetGate(budgetConn, taskID, workspaceID, tenantID, provider, baseURL, modelName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	if allowed, _ := decision["allowed"].(bool); !allowed {
		writeJSON(w, http.StatusPaymentRequired, decision)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func writeBudgetGateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBudgetDisabled):
		writeJSON(w, http.StatusForbidden, map[string]any{
			"message": "LLM 预算限额未启用",
			"code":    "llm_budget_disabled",
		})
	case errors.Is(err, errBudgetProviderInelig):
		writeJSON(w, http.StatusForbidden, map[string]any{
			"message": "该 provider 未启用派生子 Key，不支持预算配置",
			"code":    "llm_budget_provider_not_sub_token",
		})
	default:
		writeErrorJSON(w, nil, http.StatusInternalServerError, err.Error())
	}
}

// recordBudgetUsageBatchLocal is used by container inbound (no Django hop).
func recordBudgetUsageBatchLocal(tenantID, workspaceID, taskID string, items []any) (map[string]any, int, error) {
	budgetConn := getBudgetDB()
	if budgetConn == nil {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("budget db unavailable")
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		if errors.Is(err, errBudgetDisabled) {
			return map[string]any{
				"message": "LLM 预算限额未启用",
				"code":    "llm_budget_disabled",
			}, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}
	results := make([]budgetRecordResult, 0, len(items))
	for _, raw := range items {
		item, isObj := raw.(map[string]any)
		if !isObj {
			return map[string]any{"detail": "items 元素必须是对象"}, http.StatusBadRequest, fmt.Errorf("bad item")
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		baseURL := strings.TrimSpace(fmt.Sprintf("%v", item["base_url"]))
		modelName := strings.TrimSpace(fmt.Sprintf("%v", item["model_name"]))
		idem := strings.TrimSpace(fmt.Sprintf("%v", item["idempotency_key"]))
		if provider == "" || provider == "<nil>" || modelName == "" || modelName == "<nil>" || idem == "" || idem == "<nil>" {
			return map[string]any{"detail": "provider、model_name、idempotency_key 必填"}, http.StatusBadRequest, fmt.Errorf("missing fields")
		}
		if baseURL == "<nil>" {
			baseURL = ""
		}
		if err := requireBudgetEligibleProvider(tenantID, provider, baseURL); err != nil {
			if errors.Is(err, errBudgetDisabled) {
				return map[string]any{"message": "LLM 预算限额未启用", "code": "llm_budget_disabled"}, http.StatusForbidden, err
			}
			if errors.Is(err, errBudgetProviderInelig) {
				return map[string]any{
					"message": "该 provider 未启用派生子 Key，不支持预算配置",
					"code":    "llm_budget_provider_not_sub_token",
				}, http.StatusForbidden, err
			}
			return nil, http.StatusInternalServerError, err
		}
		inDelta, err := parseTokenDelta(item["input_tokens_delta"])
		if err != nil {
			return map[string]any{"detail": "invalid token counts"}, http.StatusBadRequest, err
		}
		outDelta, err := parseTokenDelta(item["output_tokens_delta"])
		if err != nil {
			return map[string]any{"detail": "invalid token counts"}, http.StatusBadRequest, err
		}
		usage, created, err := recordModelBudgetUsageDelta(budgetConn, taskID, workspaceID, tenantID, budgetRecordItem{
			Provider: provider, BaseURL: baseURL, ModelName: modelName,
			InputTokensDelta: inDelta, OutputTokensDelta: outDelta, IdempotencyKey: idem,
		})
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		results = append(results, budgetRecordResult{
			Provider: provider, ModelName: modelName,
			SpentAmount: usage.SpentAmount, Created: created, UsageID: usage.ID,
		})
	}
	return map[string]any{"items": results, "currency": "CNY"}, http.StatusOK, nil
}
