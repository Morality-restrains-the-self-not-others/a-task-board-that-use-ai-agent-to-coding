package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"tracelog"
)

func handleWorkspaceFeatureParams(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	userID, ok := ensureFeatureParamsAuthenticated(w, r)
	if !ok {
		return
	}
	exists, err := verifyTenantCompanyExistsFP(tenantID)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "租户校验失败")
		return
	}
	if !exists {
		writeFeatureParamsMessage(w, r, http.StatusNotFound, "租户不存在")
		return
	}
	_, _, memberOK := resolveTenantMemberFP(w, r, tenantID, userID)
	if !memberOK {
		return
	}

	ws, err := fetchWorkspaceHTTP(r.Context(), workspaceID)
	if err != nil || ws == nil {
		writeFeatureParamsMessage(w, r, http.StatusNotFound, "工作空间不存在")
		return
	}
	companyID := strField(ws, "company_id")
	if companyID == "" {
		companyID = strField(ws, "company")
	}
	if companyID != tenantID {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, "工作空间不属于该租户")
		return
	}
	if !hasWorkspaceAccessForUser(r.Context(), tenantID, workspaceID, userID) {
		writeFeatureParamsMessage(w, r, http.StatusForbidden, "无权访问该工作空间")
		return
	}

	decision := resolveFeatureParamsAccess(r, fpResourceWorkspace)
	if !decision.Allowed {
		recordFeatureParamsAccessAudit(r, userID, tenantID, workspaceID, fpResourceWorkspace, decision, http.StatusForbidden)
		writeFeatureParamsMessage(w, r, http.StatusForbidden, decision.Message)
		return
	}

	wsID := strField(ws, "id")
	if wsID == "" {
		wsID = workspaceID
	}
	wsName := strField(ws, "name")
	allowPersonal := workspaceAllowPersonal(ws)

	switch r.Method {
	case http.MethodGet:
		handleWorkspaceFeatureParamsGet(w, r, tenantID, wsID, wsName, allowPersonal, userID, decision)
	case http.MethodPost:
		handleWorkspaceFeatureParamsPost(w, r, tenantID, wsID, wsName, allowPersonal, userID, decision)
	default:
		writeFeatureParamsMessage(w, r, http.StatusMethodNotAllowed, "不支持的请求方法")
	}
}

func workspaceAllowPersonal(ws map[string]any) bool {
	switch v := ws["allow_personal_feature_params"].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case string:
		return strings.EqualFold(v, "true") || v == "1"
	default:
		return false
	}
}

func hasWorkspaceAccessForUser(ctx context.Context, tenantID, workspaceID, userID string) bool {
	rows, err := listWorkspaceAccessHTTP(ctx, tenantID, workspaceID)
	if err != nil {
		return false
	}
	if len(rows) == 0 {
		return true
	}
	for _, row := range rows {
		// workspace-permissions 富化响应使用 user 键（旧 raw 行是 user_id）
		uid := strField(row, "user")
		if uid == "" {
			uid = strField(row, "user_id")
		}
		if uid == userID {
			return true
		}
	}
	return false
}

func isWorkspaceAdminForFeatureParams(ctx context.Context, tenantID, workspaceID, userID string) bool {
	rows, err := listWorkspaceAccessHTTP(ctx, tenantID, workspaceID)
	if err != nil {
		return false
	}
	for _, row := range rows {
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
		return perm == "admin"
	}
	return false
}

func handleWorkspaceFeatureParamsGet(
	w http.ResponseWriter, r *http.Request,
	tenantID, workspaceID, wsName string, allowPersonal bool,
	userID string, decision featureParamsAccessDecision,
) {
	companyParams, _ := loadTenantFeatureParams(tenantID)
	wsLoad, err := loadWorkspaceFeatureParamsFull(workspaceID)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "读取功能参数失败")
		return
	}
	data := serializeWorkspaceFeatureParamsData(wsLoad, companyParams, workspaceID)
	if decision.ViewMode == fpViewSummary {
		data = redactFeatureParamsPayload(data)
	}
	isAdmin := isWorkspaceAdminForFeatureParams(r.Context(), tenantID, workspaceID, userID)
	if memberID, isTenantAdmin, err := resolveUserMember(r, tenantID, userID); err == nil && memberID != "" {
		_ = memberID
		isAdmin = isAdmin || isTenantAdmin
	}
	recordFeatureParamsAccessAudit(r, userID, tenantID, workspaceID, fpResourceWorkspace, decision, http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":               data,
		"is_workspace_admin": isAdmin,
		"workspace": map[string]any{
			"id":                            workspaceID,
			"name":                          wsName,
			"allow_personal_feature_params": allowPersonal,
		},
	})
}

func handleWorkspaceFeatureParamsPost(
	w http.ResponseWriter, r *http.Request,
	tenantID, workspaceID, wsName string, allowPersonal bool,
	userID string, decision featureParamsAccessDecision,
) {
	isAdmin := isWorkspaceAdminForFeatureParams(r.Context(), tenantID, workspaceID, userID)
	if _, isTenantAdmin, err := resolveUserMember(r, tenantID, userID); err == nil && isTenantAdmin {
		isAdmin = true
	}
	if !isAdmin {
		recordFeatureParamsAccessAudit(r, userID, tenantID, workspaceID, fpResourceWorkspace, decision, http.StatusForbidden)
		writeFeatureParamsMessage(w, r, http.StatusForbidden, "仅工作空间管理员或租户管理员可修改配置")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeFeatureParamsMessage(w, r, http.StatusBadRequest, "无效 JSON")
		return
	}
	useDefault := true
	if raw, ok := body["use_company_default"]; ok {
		useDefault = parseBoolish(raw, true)
	}
	payload := map[string]interface{}{
		"workspace_id":        workspaceID,
		"company_id":          tenantID,
		"use_company_default": useDefault,
		"extra_env_vars":      extraEnvVarsFromBody(body),
	}
	if !useDefault {
		providers := normalizeProvidersList(body["providers"])
		forceSubTokenDisabled(providers) // OPT-20260816-016: 派生子 Key 功能下线，写路径强制关闭
		maxSteps, stepErr := parseAgentMaxSteps(body, 200)
		if stepErr != "" {
			writeFeatureParamsMessage(w, r, http.StatusBadRequest, stepErr)
			return
		}
		payload["providers"] = providers
		payload["agent_model"] = strings.TrimSpace(strField(body, "agent_model"))
		payload["agent_model_provider"] = strings.ToLower(strings.TrimSpace(strField(body, "agent_model_provider")))
		payload["agent_max_steps"] = maxSteps
		payload["summary_model"] = strings.TrimSpace(strField(body, "summary_model"))
		payload["summary_model_provider"] = strings.ToLower(strings.TrimSpace(strField(body, "summary_model_provider")))
		payload["llm_budget_enabled"] = computeLLMBudgetEnabled(providers)
	}
	if _, ok := body["allow_personal_feature_params"]; ok {
		allowPersonal = parseBoolish(body["allow_personal_feature_params"], allowPersonal)
		payload["allow_personal_feature_params"] = allowPersonal
		if err := patchWorkspaceAllowPersonalHTTP(r.Context(), tenantID, workspaceID, allowPersonal); err != nil {
			logWarn("patch workspace allow_personal failed: "+err.Error(), "")
		}
	}
	paramsMap, status, err := upsertWorkspaceFeatureParamsHTTP(payload)
	if err != nil || status >= 400 {
		writeFeatureParamsMessage(w, r, http.StatusBadGateway, "保存失败")
		return
	}
	companyParams, _ := loadTenantFeatureParams(tenantID)
	wsLoad := &workspaceFeatureParamsLoad{Found: true, UseCompanyDefault: useDefault, Params: paramsMap}
	recordFeatureParamsAccessAudit(r, userID, tenantID, workspaceID, fpResourceWorkspace, decision, http.StatusOK)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "保存成功",
		"data":    serializeWorkspaceFeatureParamsData(wsLoad, companyParams, workspaceID),
	})
}

func parseBoolish(v any, defaultVal bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	default:
		return defaultVal
	}
}

func patchWorkspaceAllowPersonalHTTP(ctx context.Context, tenantID, workspaceID string, allow bool) error {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return fmt.Errorf("project service url not configured")
	}
	u := base + "/api/projects/workspaces/tenant_id/" + url.PathEscape(tenantID) + "/" + url.PathEscape(workspaceID)
	body, _ := json.Marshal(map[string]any{"allow_personal_feature_params": allow})
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := featureParamsHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("project service status=%d", resp.StatusCode)
	}
	return nil
}

func serializeWorkspaceFeatureParamsData(wsLoad *workspaceFeatureParamsLoad, companyParams *tenantFeatureParamsRow, workspaceID string) map[string]any {
	base := map[string]any{
		"use_company_default":    true,
		"providers":              []map[string]any{},
		"agent_model":            "",
		"agent_model_provider":   "",
		"agent_max_steps":        "200",
		"summary_model":          "",
		"summary_model_provider": "",
		"extra_env_vars":         []map[string]any{},
	}
	if wsLoad != nil && wsLoad.Found && wsLoad.Params != nil {
		p := wsLoad.Params
		base["use_company_default"] = wsLoad.UseCompanyDefault
		providers := providersFromAny(p["providers"])
		extra := parseExtraEnvVarsAny(p["extra_env_vars"])
		maxSteps := intFromAny(p["agent_max_steps"], 200)
		stored := map[string]any{
			"providers":              providers,
			"agent_model":            strField(p, "agent_model"),
			"agent_model_provider":   strField(p, "agent_model_provider"),
			"agent_max_steps":        fmt.Sprintf("%d", maxSteps),
			"summary_model":          strField(p, "summary_model"),
			"summary_model_provider": strField(p, "summary_model_provider"),
			"extra_env_vars":         extra,
		}
		if id := strField(p, "id"); id != "" {
			base["id"] = id
		}
		if wsLoad.UseCompanyDefault {
			base["workspace_config"] = stored
		} else {
			for k, v := range stored {
				base[k] = v
			}
		}
	}
	if companyParams != nil {
		coProviders := providersFromJSONString(companyParams.ProvidersJSON)
		coExtra := parseExtraEnvVarsJSON(companyParams.ExtraEnvJSON)
		coSteps := companyParams.AgentMaxSteps
		if coSteps <= 0 {
			coSteps = 200
		}
		coCfg := map[string]any{
			"id":                     companyParams.ID,
			"providers":              coProviders,
			"agent_model":            companyParams.AgentModel,
			"agent_model_provider":   companyParams.AgentProvider,
			"agent_max_steps":        fmt.Sprintf("%d", coSteps),
			"summary_model":          companyParams.SummaryModel,
			"summary_model_provider": companyParams.SummaryProvider,
			"extra_env_vars":         coExtra,
		}
		base["company_config"] = coCfg
	}
	useCompany := true
	if v, ok := base["use_company_default"].(bool); ok {
		useCompany = v
	}
	previewScope := "workspace"
	previewName := "工作空间配置"
	previewID := strField(base, "id")
	previewSrc := base
	if useCompany {
		previewScope = "company"
		previewName = "公司默认"
		if co, ok := base["company_config"].(map[string]any); ok {
			previewSrc = co
			previewID = strField(co, "id")
		}
	} else if workspaceID != "" {
		previewName = fmt.Sprintf("工作空间配置:%s", workspaceID)
	}
	cfgRow := workspacePreviewConfigRow(base, previewSrc, previewScope, previewID, previewName)
	base["env_preview"] = envPreviewFromConfig(cfgRow)
	// OPT-20260809-019: 来源可用性标志随 data 下发，summary 脱敏时保留。
	// company / workspace 各自看本级 extra_env_vars 或 LLM provider/模型，不把公司配置算成工作空间可用。
	workspaceExtra := []map[string]any{}
	workspaceProviders := []map[string]any{}
	workspaceModel, workspaceProvider := "", ""
	if wsLoad != nil && wsLoad.Params != nil {
		workspaceExtra = parseExtraEnvVarsAny(wsLoad.Params["extra_env_vars"])
		workspaceProviders = providersFromAny(wsLoad.Params["providers"])
		workspaceModel = strField(wsLoad.Params, "agent_model")
		workspaceProvider = strField(wsLoad.Params, "agent_model_provider")
	}
	companyExtra := []map[string]any{}
	companyProviders := []map[string]any{}
	companyModel, companyProvider := "", ""
	if companyParams != nil {
		companyExtra = parseExtraEnvVarsJSON(companyParams.ExtraEnvJSON)
		companyProviders = providersFromJSONString(companyParams.ProvidersJSON)
		companyModel = companyParams.AgentModel
		companyProvider = companyParams.AgentProvider
	}
	base["env_var_sources_available"] = featureParamsSourcesAvailable(
		featureParamsSourceConfigured(companyExtra, companyProviders, companyModel, companyProvider),
		featureParamsSourceConfigured(workspaceExtra, workspaceProviders, workspaceModel, workspaceProvider),
	)
	return base
}

func workspacePreviewConfigRow(base, previewSrc map[string]any, scope, id, name string) *featureParamsConfig {
	providersRaw, _ := json.Marshal(previewSrc["providers"])
	extraRaw, _ := json.Marshal(previewSrc["extra_env_vars"])
	steps := 200
	if s := strField(previewSrc, "agent_max_steps"); s != "" {
		if n, err := fmt.Sscanf(s, "%d", &steps); err != nil || n != 1 {
			steps = 200
		}
	}
	return &featureParamsConfig{
		ID:              id,
		ProvidersJSON:   string(providersRaw),
		AgentModel:      strField(previewSrc, "agent_model"),
		AgentProvider:   strField(previewSrc, "agent_model_provider"),
		AgentMaxSteps:   steps,
		SummaryModel:    strField(previewSrc, "summary_model"),
		SummaryProvider: strField(previewSrc, "summary_model_provider"),
		ExtraEnvJSON:    string(extraRaw),
		DisplayName:     name,
		Scope:           scope,
	}
}

func providersFromAny(v any) []map[string]any {
	switch t := v.(type) {
	case string:
		return providersFromJSONString(t)
	case []any:
		return normalizeProvidersList(t)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return []map[string]any{}
		}
		return providersFromJSONString(string(b))
	}
}

func parseExtraEnvVarsAny(v any) []map[string]any {
	switch t := v.(type) {
	case string:
		return parseExtraEnvVarsJSON(t)
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case []map[string]any:
		return t
	default:
		return []map[string]any{}
	}
}
