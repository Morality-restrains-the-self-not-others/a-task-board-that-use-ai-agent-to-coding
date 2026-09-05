package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// rewriteSubTokenProvidersForProxy: sub-token providers get gateway base_url +
// proxy_token api_key + proxy_mode (SSOT; Python rewrite helper removed).
func rewriteSubTokenProvidersForProxy(
	providers []map[string]any,
	tenantID, workspaceID, taskID, proxyToken, gatewayBase string,
) []map[string]any {
	gatewayBase = strings.TrimRight(strings.TrimSpace(gatewayBase), "/")
	token := strings.TrimSpace(proxyToken)
	out := make([]map[string]any, 0, len(providers))
	for _, item := range providers {
		if item == nil {
			continue
		}
		copy := cloneProviderMap(item)
		if !providerBool(copy["use_sub_token"]) {
			out = append(out, copy)
			continue
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", copy["provider"]))
		if provider == "" || provider == "<nil>" {
			out = append(out, copy)
			continue
		}
		copy["base_url"] = fmt.Sprintf(
			"%s/api/tenant/%s/workspace/%s/task/%s/llm/%s/v1",
			gatewayBase, tenantID, workspaceID, taskID, provider,
		)
		copy["api_key"] = token
		copy["proxy_mode"] = "task_ai_endpoint"
		out = append(out, copy)
	}
	return out
}

func cloneProviderMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func providerBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return strings.EqualFold(x, "true") || x == "1"
	default:
		return false
	}
}

// coerceProvidersBaseURLsInEnv repairs accidentally concatenated absolute URLs in
// the env payload before container delivery (covers persisted paste accidents
// without a re-save). Operator-typed paths are not rewritten.
func coerceProvidersBaseURLsInEnv(env map[string]string) []map[string]any {
	providers := parseProvidersJSON(env[envProvidersJSON])
	if len(providers) == 0 {
		return providers
	}
	changed := false
	for _, item := range providers {
		if item == nil {
			continue
		}
		provider := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		before := strings.TrimSpace(fmt.Sprintf("%v", item["base_url"]))
		after := normalizeProviderBaseURL(provider, before)
		if after != before {
			item["base_url"] = after
			changed = true
		}
	}
	if !changed {
		return providers
	}
	b, err := json.Marshal(providers)
	if err != nil {
		logWarn("feature_params: marshal coerced providers failed: "+err.Error(), "")
		return providers
	}
	env[envProvidersJSON] = string(b)
	return providers
}

func applyProxyRewriteToEnv(
	env map[string]string,
	tenantID, workspaceID, taskID, proxyToken string,
) []map[string]any {
	providers := coerceProvidersBaseURLsInEnv(env)
	if len(providers) == 0 {
		return providers
	}
	token := strings.TrimSpace(proxyToken)
	if !taskAIEndpointEnabled() || token == "" {
		return providers
	}
	rewritten := rewriteSubTokenProvidersForProxy(
		providers, tenantID, workspaceID, taskID, token, taskAIEndpointPublicBase(),
	)
	b, err := json.Marshal(rewritten)
	if err != nil {
		logWarn("feature_params: marshal rewritten providers failed: "+err.Error(), "")
		return providers
	}
	env[envProvidersJSON] = string(b)
	return rewritten
}
