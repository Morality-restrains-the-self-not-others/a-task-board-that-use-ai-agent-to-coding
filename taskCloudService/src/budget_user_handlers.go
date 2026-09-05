package main

import (
	"authz"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"tracelog"
)

type budgetTaskSnap struct {
	ID          string
	TenantID    string
	WorkspaceID string
	OwnerID     string
}

func handleUserWorkspaceModelBudgetDefaults(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		writeBudgetGateError(w, err)
		return
	}
	ws, err := fetchWorkspaceHTTP(r.Context(), workspaceID)
	if err != nil || ws == nil {
		writeErrorJSON(w, r, http.StatusNotFound, "工作空间不存在")
		return
	}
	if strField(ws, "company_id") != tenantID && strField(ws, "company") != tenantID {
		writeErrorJSON(w, r, http.StatusNotFound, "工作空间不存在")
		return
	}

	switch r.Method {
	case http.MethodGet:
		savedRows, err := listWorkspaceBudgetDefaults(budgetConn, workspaceID, tenantID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		saved := map[string]workspaceBudgetDefaultRow{}
		for _, row := range savedRows {
			key := budgetModelKey(row.Provider, row.BaseURL, row.ModelName)
			saved[key] = row
		}
		items := make([]map[string]any, 0)
		for _, spec := range eligibleBudgetModels(tenantID) {
			key := budgetModelKey(spec.Provider, spec.BaseURL, spec.ModelName)
			row, ok := saved[key]
			item := map[string]any{
				"provider":            spec.Provider,
				"base_url":            spec.BaseURL,
				"model_name":          spec.ModelName,
				"input_price_per_1m":  "0",
				"output_price_per_1m": "0",
				"budget_limit":        "0",
				"configured":          ok,
			}
			if ok {
				item["input_price_per_1m"] = row.InputPricePer1M
				item["output_price_per_1m"] = row.OutputPricePer1M
				item["budget_limit"] = row.BudgetLimit
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
	case http.MethodPatch, http.MethodPut, http.MethodPost:
		if !ensureCanEditWorkspaceBudgetDefaults(w, r, tenantID, workspaceID) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "无效 JSON")
			return
		}
		rawItems, ok := body["items"].([]any)
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
			return
		}
		eligible := map[string]bool{}
		for _, spec := range eligibleBudgetModels(tenantID) {
			eligible[budgetModelKey(spec.Provider, spec.BaseURL, spec.ModelName)] = true
		}
		upserts := make([]workspaceBudgetDefaultRow, 0, len(rawItems))
		for _, raw := range rawItems {
			m, ok := raw.(map[string]any)
			if !ok {
				writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
				return
			}
			provider := strings.TrimSpace(strField(m, "provider"))
			baseURL := strings.TrimSpace(strField(m, "base_url"))
			modelName := strings.TrimSpace(strField(m, "model_name"))
			if provider == "" || modelName == "" {
				writeErrorJSON(w, r, http.StatusBadRequest, "provider 与 model_name 必填")
				return
			}
			if err := requireBudgetEligibleProvider(tenantID, provider, baseURL); err != nil {
				writeBudgetGateError(w, err)
				return
			}
			if !eligible[budgetModelKey(provider, baseURL, modelName)] {
				writeErrorJSON(w, r, http.StatusBadRequest, "模型不在 sub-token provider 支持列表中")
				return
			}
			inPrice := strField(m, "input_price_per_1m")
			outPrice := strField(m, "output_price_per_1m")
			limit := strField(m, "budget_limit")
			if !isNonNegativeDecimal(inPrice) || !isNonNegativeDecimal(outPrice) || !isNonNegativeDecimal(limit) {
				writeErrorJSON(w, r, http.StatusBadRequest, "单价格式无效或金额不能为负")
				return
			}
			upserts = append(upserts, workspaceBudgetDefaultRow{
				WorkspaceID: workspaceID, CompanyID: tenantID,
				Provider: provider, BaseURL: baseURL, ModelName: modelName,
				InputPricePer1M: inPrice, OutputPricePer1M: outPrice, BudgetLimit: limit,
			})
		}
		out := make([]map[string]any, 0, len(upserts))
		for _, row := range upserts {
			saved, err := upsertWorkspaceBudgetDefault(budgetConn, row)
			if err != nil {
				writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			out = append(out, workspaceDefaultToMap(saved))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out, "currency": "CNY"})
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleUserTaskModelBudgets(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		writeBudgetGateError(w, err)
		return
	}
	snap, err := fetchTaskInWorkspaceHTTP(tenantID, workspaceID, taskID)
	if err != nil || snap == nil {
		writeErrorJSON(w, r, http.StatusNotFound, "任务不存在")
		return
	}
	if snap.TenantID != "" && snap.TenantID != tenantID {
		writeErrorJSON(w, r, http.StatusNotFound, "任务不属于该租户")
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := buildTaskBudgetItemsLocal(snap)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
	case http.MethodPatch, http.MethodPut, http.MethodPost:
		if !ensureCanModifyTaskBudget(w, r, tenantID, snap) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "无效 JSON")
			return
		}
		rawItems, ok := body["items"].([]any)
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
			return
		}
		for _, raw := range rawItems {
			m, ok := raw.(map[string]any)
			if !ok {
				writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
				return
			}
			provider := strings.TrimSpace(strField(m, "provider"))
			baseURL := strings.TrimSpace(strField(m, "base_url"))
			modelName := strings.TrimSpace(strField(m, "model_name"))
			if provider == "" || modelName == "" {
				writeErrorJSON(w, r, http.StatusBadRequest, "provider 与 model_name 必填")
				return
			}
			if err := requireBudgetEligibleProvider(tenantID, provider, baseURL); err != nil {
				writeBudgetGateError(w, err)
				return
			}
			limit := strField(m, "budget_limit")
			if limit == "" {
				limit = "0"
			}
			if !isNonNegativeDecimal(limit) {
				writeErrorJSON(w, r, http.StatusBadRequest, "budget_limit 格式无效或预算不能为负")
				return
			}
			limCopy := limit
			_, err := upsertTaskModelBudget(budgetConn, taskModelBudgetRow{
				TodoID: snap.ID, WorkspaceID: snap.WorkspaceID, CompanyID: tenantID,
				Provider: provider, BaseURL: baseURL, ModelName: modelName,
				BudgetLimit: &limCopy, BudgetLimitSource: "override",
			})
			if err != nil {
				writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		}
		items, err := buildTaskBudgetItemsLocal(snap)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleUserRaiseTaskModelBudget(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		writeBudgetGateError(w, err)
		return
	}
	snap, err := fetchTaskInWorkspaceHTTP(tenantID, workspaceID, taskID)
	if err != nil || snap == nil {
		writeErrorJSON(w, r, http.StatusNotFound, "任务不存在")
		return
	}
	userID := effectiveUserID(r)
	memberID, _, _ := resolveUserMember(r, tenantID, userID)
	isWSAdmin := isWorkspaceAdminOrEditor(r.Context(), tenantID, workspaceID, userID)
	// v63: 管理员判定迁移至 authz 权限码
	allowed, err := evaluateUserCanRaiseTaskBudget(
		budgetConn, tenantID, userID, memberID, snap.OwnerID, authz.HasPerm(r, tenantID, authz.PermCloudManage), isWSAdmin,
	)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !allowed {
		writeErrorJSON(w, r, http.StatusForbidden, "无权临时上调任务预算")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "无效 JSON")
		return
	}
	if !boolField(body, "confirm") {
		writeErrorJSON(w, r, http.StatusBadRequest, "需要二次确认（confirm: true）")
		return
	}
	provider := strings.TrimSpace(strField(body, "provider"))
	baseURL := strings.TrimSpace(strField(body, "base_url"))
	modelName := strings.TrimSpace(strField(body, "model_name"))
	if provider == "" || modelName == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "provider 与 model_name 必填")
		return
	}
	if err := requireBudgetEligibleProvider(tenantID, provider, baseURL); err != nil {
		writeBudgetGateError(w, err)
		return
	}
	newLimit := strField(body, "new_budget_limit")
	if !isNonNegativeDecimal(newLimit) {
		writeErrorJSON(w, r, http.StatusBadRequest, "new_budget_limit 格式无效或预算不能为负")
		return
	}
	limCopy := newLimit
	_, err = upsertTaskModelBudget(budgetConn, taskModelBudgetRow{
		TodoID: snap.ID, WorkspaceID: snap.WorkspaceID, CompanyID: tenantID,
		Provider: provider, BaseURL: baseURL, ModelName: modelName,
		BudgetLimit: &limCopy, BudgetLimitSource: "raised",
	})
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	items, err := buildTaskBudgetItemsLocal(snap)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "currency": "CNY"})
}

type eligibleBudgetModel struct {
	Provider  string
	BaseURL   string
	ModelName string
}

func eligibleBudgetModels(companyID string) []eligibleBudgetModel {
	params, err := loadTenantFeatureParams(companyID)
	if err != nil || params == nil || !params.LLMBudgetEnabled {
		return nil
	}
	out := make([]eligibleBudgetModel, 0)
	for _, entry := range parseProvidersJSON(params.ProvidersJSON) {
		if !isBudgetEligibleProvider(entry, true) {
			continue
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", entry["provider"]))
		baseURL := strings.TrimSpace(fmt.Sprintf("%v", entry["base_url"]))
		models := supportedModelsFromEntry(entry)
		for _, m := range models {
			out = append(out, eligibleBudgetModel{Provider: provider, BaseURL: baseURL, ModelName: m})
		}
	}
	return out
}

func supportedModelsFromEntry(entry map[string]any) []string {
	raw, ok := entry["supported_models"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func budgetModelKey(provider, baseURL, modelName string) string {
	return strings.TrimSpace(provider) + "|" + normalizeBaseURL(baseURL) + "|" + strings.TrimSpace(modelName)
}

func isNonNegativeDecimal(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false
	}
	return f >= 0
}

func ensureCanEditWorkspaceBudgetDefaults(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) bool {
	userID := effectiveUserID(r)
	_, isAdmin, err := resolveUserMember(r, tenantID, userID)
	if err == nil && isAdmin {
		return true
	}
	if lookupIsCompanyCreator(tenantID, userID) {
		return true
	}
	if isWorkspaceAdminOrEditor(r.Context(), tenantID, workspaceID, userID) {
		return true
	}
	writeErrorJSON(w, r, http.StatusForbidden, "无权修改工作空间预算默认")
	return false
}

func ensureCanModifyTaskBudget(w http.ResponseWriter, r *http.Request, tenantID string, snap *budgetTaskSnap) bool {
	userID := effectiveUserID(r)
	memberID, isAdmin, err := resolveUserMember(r, tenantID, userID)
	if err == nil && isAdmin {
		return true
	}
	if lookupIsCompanyCreator(tenantID, userID) {
		return true
	}
	if isWorkspaceAdminOrEditor(r.Context(), tenantID, snap.WorkspaceID, userID) {
		return true
	}
	if memberID != "" && snap.OwnerID != "" && memberID == snap.OwnerID {
		return true
	}
	writeErrorJSON(w, r, http.StatusForbidden, "无权修改任务预算")
	return false
}

func isWorkspaceAdminOrEditor(ctx context.Context, tenantID, workspaceID, userID string) bool {
	rows, err := listWorkspaceAccessHTTP(ctx, tenantID, workspaceID)
	if err != nil {
		return false
	}
	for _, row := range rows {
		// workspace-permissions 富化响应使用 user 键（旧 raw 行是 user_id）
		uid := strField(row, "user")
		if uid == "" {
			uid = strField(row, "user_id")
		}
		if uid != userID {
			continue
		}
		perm := strField(row, "role")
		if perm == "" {
			perm = strField(row, "permission")
		}
		if perm == "admin" || perm == "edit" {
			return true
		}
	}
	return false
}

func fetchWorkspaceHTTP(ctx context.Context, workspaceID string) (map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("project service url not configured")
	}
	u := base + "/api/internal/workspaces/" + url.PathEscape(workspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("project service status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func listWorkspaceAccessHTTP(ctx context.Context, tenantID, workspaceID string) ([]map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("project service url not configured")
	}
	u := fmt.Sprintf("%s/api/projects/workspace-access/tenant_id/%s/workspace-permissions?workspace_id=%s",
		base, url.PathEscape(tenantID), url.QueryEscape(workspaceID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("X-Auth-User-Id", "internal")
	if cfg.GatewayInternalSecret != "" {
		req.Header.Set("X-TaskGateway-Internal-Secret", cfg.GatewayInternalSecret)
		req.Header.Set("X-Gateway-Auth-Verified", "1")
		req.Header.Set("X-User-Id", "internal")
	}
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workspace-access status=%d", resp.StatusCode)
	}
	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	switch v := body.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out, nil
	case map[string]any:
		if items, ok := v["items"].([]any); ok {
			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if m, ok := item.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out, nil
		}
	}
	return nil, nil
}

func fetchTaskInWorkspaceHTTP(tenantID, workspaceID, taskID string) (*budgetTaskSnap, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("task service url not configured")
	}
	u := base + "/api/tasks/" + url.PathEscape(taskID) + "/"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task service status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	ws := strField(body, "workspace_id")
	if ws != "" && ws != workspaceID {
		return nil, nil
	}
	owner := strField(body, "owner_id")
	if owner == "" {
		owner = strField(body, "owner")
	}
	tid := strField(body, "tenant_id")
	if tid == "" {
		tid = tenantID
	}
	id := strField(body, "id")
	if id == "" {
		id = taskID
	}
	return &budgetTaskSnap{ID: id, TenantID: tid, WorkspaceID: workspaceID, OwnerID: owner}, nil
}

func buildTaskBudgetItemsLocal(snap *budgetTaskSnap) ([]map[string]any, error) {
	conn := getBudgetDB()
	if conn == nil {
		return nil, fmt.Errorf("budget db not open")
	}
	companyID := snap.TenantID
	defaults, err := listWorkspaceBudgetDefaults(conn, snap.WorkspaceID, companyID)
	if err != nil {
		return nil, err
	}
	defaultMap := map[string]workspaceBudgetDefaultRow{}
	for _, d := range defaults {
		defaultMap[budgetModelKey(d.Provider, d.BaseURL, d.ModelName)] = d
	}
	taskRows, err := listTaskModelBudgets(conn, snap.ID)
	if err != nil {
		return nil, err
	}
	usages, err := listTaskModelBudgetUsage(conn, snap.ID)
	if err != nil {
		return nil, err
	}
	usageMap := map[string]budgetUsageRow{}
	for _, u := range usages {
		usageMap[budgetModelKey(u.Provider, u.BaseURL, u.ModelName)] = u
	}
	items := make([]map[string]any, 0, len(taskRows))
	for _, tr := range taskRows {
		key := budgetModelKey(tr.Provider, tr.BaseURL, tr.ModelName)
		def := defaultMap[key]
		usage := usageMap[key]
		inPrice, outPrice := "0", "0"
		if def.Provider != "" {
			inPrice = def.InputPricePer1M
			outPrice = def.OutputPricePer1M
		}
		effective := "0"
		if tr.BudgetLimit != nil && strings.TrimSpace(*tr.BudgetLimit) != "" {
			effective = strings.TrimSpace(*tr.BudgetLimit)
		} else if def.BudgetLimit != "" {
			effective = def.BudgetLimit
		}
		spent := "0"
		var inTok, outTok int64
		if usage.TodoID != "" || usage.ID != "" {
			spent = usage.SpentAmount
			inTok = usage.InputTokens
			outTok = usage.OutputTokens
		}
		exhausted := false
		if ef, err1 := strconv.ParseFloat(effective, 64); err1 == nil && ef > 0 {
			if sf, err2 := strconv.ParseFloat(spent, 64); err2 == nil {
				exhausted = sf >= ef
			}
		}
		var lim any
		if tr.BudgetLimit != nil {
			lim = *tr.BudgetLimit
		} else {
			lim = nil
		}
		items = append(items, map[string]any{
			"provider":               tr.Provider,
			"base_url":               tr.BaseURL,
			"model_name":             tr.ModelName,
			"input_price_per_1m":     inPrice,
			"output_price_per_1m":    outPrice,
			"budget_limit":           lim,
			"effective_budget_limit": effective,
			"budget_limit_source":    tr.BudgetLimitSource,
			"input_tokens":           inTok,
			"output_tokens":          outTok,
			"spent_amount":           spent,
			"exhausted":              exhausted,
		})
	}
	return items, nil
}
