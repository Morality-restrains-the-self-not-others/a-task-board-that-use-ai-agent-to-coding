package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var legacyFeatureParamsFields = []string{
	"trae_model", "trae_model_provider", "trae_max_steps",
	"lakeview_model", "lakeview_model_provider",
}

// writeFeatureParamsMessage sends a feature-params error response with trace_id
// extracted from the request (frontend data-traceId binding, aligned with writeErrorJSON).
func writeFeatureParamsMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]string{"message": message}
	if tid := traceIDFromRequest(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

func ensureFeatureParamsAuthenticated(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := effectiveUserID(r)
	if userID == "" {
		writeFeatureParamsMessage(w, r, http.StatusUnauthorized, "未认证")
		return "", false
	}
	return userID, true
}

// parseSupportedModelsField normalizes supported_models (list or delimited string).
// Delimiters: newline, English/Chinese comma, semicolon, whitespace — aligned with SPA parseSupportedModels.
func parseSupportedModelsField(raw any) []string {
	models := []string{}
	switch v := raw.(type) {
	case string:
		replacer := strings.NewReplacer(
			"\n", ",",
			"\r", ",",
			"，", ",",
			"；", ",",
			";", ",",
			"\t", ",",
			" ", ",",
		)
		for _, part := range strings.Split(replacer.Replace(v), ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				models = append(models, part)
			}
		}
	case []any:
		for _, item := range v {
			m := strings.TrimSpace(fmt.Sprintf("%v", item))
			if m != "" && m != "<nil>" {
				models = append(models, m)
			}
		}
	}
	return models
}

func verifyTenantCompanyExistsFP(companyID string) (bool, error) {
	// Company SSOT is taskTenantService; skip only when tenant URL unset (unit tests).
	if strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return true, nil
	}
	err := verifyCompanyExists(companyID)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "not found") {
		return false, nil
	}
	return false, err
}

func resolveTenantMemberFP(w http.ResponseWriter, r *http.Request, tenantID, userID string) (memberID string, isAdmin bool, ok bool) {
	memberID, isAdmin, err := resolveUserMember(r, tenantID, userID)
	if err != nil || memberID == "" {
		writeFeatureParamsMessage(w, r, http.StatusForbidden, "无权访问该租户")
		return "", false, false
	}
	return memberID, isAdmin, true
}

// absoluteURLSchemeStarts returns byte indexes of http(s):// in s (lowercased match).
func absoluteURLSchemeStarts(s string) []int {
	lower := strings.ToLower(s)
	var starts []int
	i := 0
	for i < len(lower) {
		httpsIdx := strings.Index(lower[i:], "https://")
		httpIdx := strings.Index(lower[i:], "http://")
		next, adv := -1, 0
		if httpsIdx >= 0 && (httpIdx < 0 || httpsIdx <= httpIdx) {
			next = i + httpsIdx
			adv = len("https://")
		} else if httpIdx >= 0 {
			next = i + httpIdx
			adv = len("http://")
		}
		if next < 0 {
			break
		}
		starts = append(starts, next)
		i = next + adv
	}
	return starts
}

// repairConcatenatedAbsoluteURL keeps the last absolute URL when two schemes were
// glued together (DeepSeek default fill + paste without replace).
func repairConcatenatedAbsoluteURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return s
	}
	starts := absoluteURLSchemeStarts(s)
	if len(starts) <= 1 {
		return strings.TrimRight(s, "/")
	}
	if strings.Contains(s[:starts[1]], "?") {
		return strings.TrimRight(s, "/")
	}
	repaired := strings.TrimRight(s[starts[len(starts)-1]:], "/")
	if repaired != strings.TrimRight(s, "/") {
		logWarn(fmt.Sprintf("event=feature_params_base_url_concat_repaired scheme_count=%d", len(starts)), "")
	}
	return repaired
}

// normalizeProviderBaseURL trims and repairs accidentally concatenated absolute
// URLs. Operator-specific paths (including /anthropic) are persisted as typed.
func normalizeProviderBaseURL(provider, baseURL string) string {
	_ = provider
	return repairConcatenatedAbsoluteURL(baseURL)
}

func normalizeProviderEntry(raw map[string]any) map[string]any {
	provider := strings.ToLower(strings.TrimSpace(strField(raw, "provider")))
	useSub := providerBool(raw["use_sub_token"])
	budgetEnabled := false
	if useSub {
		budgetEnabled = providerBool(raw["budget_enabled"])
	}
	models := parseSupportedModelsField(raw["supported_models"])
	return map[string]any{
		"provider":         provider,
		"api_key":          strings.TrimSpace(strField(raw, "api_key")),
		"base_url":         normalizeProviderBaseURL(provider, strField(raw, "base_url")),
		"supported_models": models,
		"use_sub_token":    useSub,
		"budget_enabled":   budgetEnabled,
	}
}

// forceSubTokenDisabled clears use_sub_token/budget_enabled on feature-params write paths.
// OPT-20260816-016: 派生子 Key 开关已从租户环境变量页移除，仅前端隐藏挡不住直接 POST API；
// 服务端强制关闭，防止通过 API 重新打开。仅写路径生效——存量已启用配置读回时保持原值。
func forceSubTokenDisabled(providers []map[string]any) {
	for _, p := range providers {
		p["use_sub_token"] = false
		p["budget_enabled"] = false
	}
}

func normalizeProvidersList(raw any) []map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, normalizeProviderEntry(m))
	}
	return out
}

func computeLLMBudgetEnabled(providers []map[string]any) bool {
	for _, item := range providers {
		if providerBool(item["budget_enabled"]) {
			return true
		}
	}
	return false
}

func parseExtraEnvVarsJSON(raw string) []map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []map[string]any{}
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []map[string]any{}
	}
	return out
}

func extraEnvVarsFromBody(body map[string]interface{}) []map[string]any {
	raw, ok := body["extra_env_vars"].([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// hasKeyedEnvVarEntry reports whether any extra_env_vars entry carries a non-empty
// key（与设置页「自定义变量」口径一致：仅统计 key 非空条目）。
func hasKeyedEnvVarEntry(extra []map[string]any) bool {
	for _, e := range extra {
		if strings.TrimSpace(strField(e, "key")) != "" {
			return true
		}
	}
	return false
}

// featureParamsSourceConfigured 判定某一级「智能体资源配置」是否可选用。
// 设置页注入容器的既有自定义 extra_env_vars，也有 LLM provider/模型生成的系统变量；
// 只看 extra_env_vars 会把已配置 DeepSeek 等智能体的公司来源误判为「暂无可用」。
func featureParamsSourceConfigured(extra, providers []map[string]any, agentModel, agentProvider string) bool {
	if hasKeyedEnvVarEntry(extra) {
		return true
	}
	if strings.TrimSpace(agentModel) != "" || strings.TrimSpace(agentProvider) != "" {
		return true
	}
	for _, p := range providers {
		if p == nil {
			continue
		}
		if strings.TrimSpace(strField(p, "provider")) != "" {
			return true
		}
	}
	return false
}

// featureParamsSourcesAvailable 计算公司/工作空间两级来源可用性。
// OPT-20260809-019: summary 视图会脱敏 extra_env_vars，前端无法据此判定来源可用性；
// 后端在响应中直接给出标志，前端无需拉取含 provider API key 的 full payload。
// workspace 无上下文（公司级 GET）时为 false。
func featureParamsSourcesAvailable(companyConfigured, workspaceConfigured bool) map[string]any {
	return map[string]any{
		"company":   companyConfigured,
		"workspace": workspaceConfigured,
	}
}

func envPreviewFromConfig(cfgRow *featureParamsConfig) map[string]any {
	env := serializeFeatureParamsEnv(cfgRow)
	out := map[string]any{}
	for k, v := range env {
		out[k] = v
	}
	return out
}

func providersFromJSONString(raw string) []map[string]any {
	parsed := parseProvidersJSON(raw)
	if len(parsed) == 0 {
		return []map[string]any{}
	}
	items := make([]any, 0, len(parsed))
	for _, item := range parsed {
		items = append(items, item)
	}
	return normalizeProvidersList(items)
}

func rejectLegacyFeatureParamsFields(body map[string]interface{}) string {
	for _, field := range legacyFeatureParamsFields {
		if _, ok := body[field]; ok {
			return fmt.Sprintf("字段 %s 已废弃，请使用 agent_* / summary_* 字段", field)
		}
	}
	return ""
}

func parseAgentMaxSteps(body map[string]interface{}, defaultVal int) (int, string) {
	raw, ok := body["agent_max_steps"]
	if !ok {
		return defaultVal, ""
	}
	switch v := raw.(type) {
	case float64:
		return int(v), ""
	case int:
		return v, ""
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, "agent_max_steps 必须是有效的数字"
		}
		return n, ""
	default:
		return 0, "agent_max_steps 必须是有效的数字"
	}
}

func handleTenantFeatureParamsRoute(w http.ResponseWriter, r *http.Request, tenantID, subPath string) {
	subPath = strings.Trim(subPath, "/")
	if subPath == "feature-params" || strings.HasPrefix(subPath, "feature-params/") {
		handleCompanyFeatureParams(w, r, tenantID)
		return
	}
	if strings.HasPrefix(subPath, "workspace/") {
		parts := strings.SplitN(subPath, "/", 4)
		if len(parts) >= 3 && (parts[2] == "feature-params" || strings.HasPrefix(parts[2], "feature-params")) {
			handleWorkspaceFeatureParams(w, r, tenantID, parts[1])
			return
		}
	}
	writeErrorJSON(w, r, http.StatusNotFound, "not found")
}

func handlePersonalAPIRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/personal/")
	path = strings.Trim(path, "/")
	if path == "feature-params-configs" || strings.HasPrefix(path, "feature-params-configs/") {
		rest := strings.TrimPrefix(path, "feature-params-configs")
		rest = strings.Trim(rest, "/")
		if rest == "" {
			handlePersonalFeatureParamsList(w, r)
			return
		}
		handlePersonalFeatureParamsDetail(w, r, rest)
		return
	}
	writeErrorJSON(w, r, http.StatusNotFound, "not found")
}
