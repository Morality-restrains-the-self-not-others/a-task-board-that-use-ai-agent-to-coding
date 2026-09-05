package main

import (
	"net/http"
	"strings"
)

const (
	headerFeatureParamsAccessContext = "X-Feature-Params-Access-Context"

	fpContextCompanySettings   = "company_settings"
	fpContextWorkspaceSettings = "workspace_settings"

	fpViewSummary = "summary"
	fpViewFull    = "full"
	fpViewDenied  = "denied"

	fpResourceCompany   = "company"
	fpResourceWorkspace = "workspace"
)

type featureParamsAccessDecision struct {
	Allowed       bool
	ViewMode      string
	AccessContext string
	Message       string
}

func featureParamsHeaderContext(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get(headerFeatureParamsAccessContext))
	return strings.ToLower(raw)
}

func detectFeatureParamsAuthMethod(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "token ") {
			return "token"
		}
		return "token"
	}
	if strings.TrimSpace(r.Header.Get("Cookie")) != "" {
		return "session"
	}
	if effectiveUserID(r) != "" {
		return "session"
	}
	return "unknown"
}

func resolveFeatureParamsAccess(r *http.Request, resource string) featureParamsAccessDecision {
	context := featureParamsHeaderContext(r)
	view := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("view")))
	allowedContexts := map[string]struct{}{fpContextWorkspaceSettings: {}}
	if resource == fpResourceCompany {
		allowedContexts = map[string]struct{}{fpContextCompanySettings: {}}
	}

	if view == fpViewSummary {
		if r.Method != http.MethodGet {
			return featureParamsAccessDecision{
				Allowed:       false,
				ViewMode:      fpViewDenied,
				AccessContext: context,
				Message:       "view=summary 仅支持 GET",
			}
		}
		ctx := context
		if ctx == "" {
			ctx = fpViewSummary
		}
		return featureParamsAccessDecision{
			Allowed:       true,
			ViewMode:      fpViewSummary,
			AccessContext: ctx,
		}
	}

	if _, ok := allowedContexts[context]; ok {
		return featureParamsAccessDecision{
			Allowed:       true,
			ViewMode:      fpViewFull,
			AccessContext: context,
		}
	}

	if context != "" {
		return featureParamsAccessDecision{
			Allowed:       false,
			ViewMode:      fpViewDenied,
			AccessContext: context,
			Message:       "访问上下文与资源不匹配；完整功能参数仅可在对应设置页访问",
		}
	}

	return featureParamsAccessDecision{
		Allowed:       false,
		ViewMode:      fpViewDenied,
		AccessContext: "",
		Message:       "完整功能参数仅可在公司参数设置或工作空间参数设置页访问；其它场景请使用 ?view=summary",
	}
}

func redactFeatureParamsProviders(providers any) []map[string]any {
	raw, ok := providers.([]any)
	if !ok {
		if typed, ok2 := providers.([]map[string]any); ok2 {
			raw = make([]any, 0, len(typed))
			for _, item := range typed {
				raw = append(raw, item)
			}
		} else {
			return nil
		}
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cleaned := map[string]any{}
		for k, v := range m {
			cleaned[k] = v
		}
		cleaned["api_key"] = ""
		out = append(out, cleaned)
	}
	return out
}

func redactFeatureParamsPayload(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range data {
		out[k] = v
	}
	if _, ok := out["providers"]; ok {
		out["providers"] = redactFeatureParamsProviders(out["providers"])
	}
	out["extra_env_vars"] = []any{}
	delete(out, "env_preview")
	for _, nestedKey := range []string{"company_config", "workspace_config"} {
		nested, ok := out[nestedKey].(map[string]any)
		if ok {
			out[nestedKey] = redactFeatureParamsPayload(nested)
		}
	}
	return out
}

func recordFeatureParamsAccessAudit(
	r *http.Request,
	userID, companyID, workspaceID, resource string,
	decision featureParamsAccessDecision,
	statusCode int,
) {
	if db == nil {
		return
	}
	clientIP := resolveClientIP(r)
	if len(clientIP) > 64 {
		clientIP = clientIP[:64]
	}
	ua := r.Header.Get("User-Agent")
	if len(ua) > 512 {
		ua = ua[:512]
	}
	referer := r.Header.Get("Referer")
	if len(referer) > 1024 {
		referer = referer[:1024]
	}
	traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))
	if len(traceID) > 64 {
		traceID = traceID[:64]
	}
	payload := map[string]interface{}{
		"user_id":        userID,
		"company_id":     companyID,
		"workspace_id":   workspaceID,
		"resource":       resource,
		"access_context": decision.AccessContext,
		"view_mode":      decision.ViewMode,
		"auth_method":    detectFeatureParamsAuthMethod(r),
		"http_method":    r.Method,
		"path":           r.URL.Path,
		"status_code":    statusCode,
		"client_ip":      clientIP,
		"user_agent":     ua,
		"referer":        referer,
		"trace_id":       traceID,
	}
	_ = insertFeatureParamsAccessAuditLocal(payload)
}
