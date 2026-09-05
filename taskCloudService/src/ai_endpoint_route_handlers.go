package main

import (
	"fmt"
	"net/http"
	"strings"
)

// handleInternalAIEndpointRoutes serves taskAIEndPoint route/credentials.
// Feature params loaded from task_cloud DB (OPT-052: migrated from saas-backend).
// Paths under /api/internal/ai-endpoint/:
//   POST resolve-route/
//   POST upstream-credentials/
func handleInternalAIEndpointRoutes(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/ai-endpoint/")
	path = strings.Trim(path, "/")
	switch path {
	case "resolve-route":
		handleInternalResolveProviderRoute(w, r)
	case "upstream-credentials":
		handleInternalUpstreamCredentials(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalResolveProviderRoute(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "bad json"})
		return
	}
	tenantID := strField(body, "tenant_id")
	provider := strField(body, "provider")
	if tenantID == "" || provider == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "missing fields"})
		return
	}
	route, err := resolveProviderRoute(tenantID, provider)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	if route == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "provider route not found"})
		return
	}
	writeJSON(w, http.StatusOK, route)
}

func handleInternalUpstreamCredentials(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "bad json"})
		return
	}
	tenantID := strField(body, "tenant_id")
	provider := strField(body, "provider")
	baseURL := strField(body, "base_url")
	if tenantID == "" || provider == "" || baseURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "missing fields"})
		return
	}
	cred, err := upstreamCredentials(tenantID, provider, baseURL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	if cred == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "credentials not found"})
		return
	}
	writeJSON(w, http.StatusOK, cred)
}

// resolveProviderRoute mirrors Django resolve_provider_route:
// first TenantFeatureParams provider with matching name and use_sub_token=true.
func resolveProviderRoute(tenantID, provider string) (map[string]any, error) {
	params, err := loadTenantFeatureParams(tenantID)
	if err != nil {
		return nil, err
	}
	if params == nil {
		return nil, nil
	}
	target := strings.TrimSpace(provider)
	for _, item := range parseProvidersJSON(params.ProvidersJSON) {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		if name != target {
			continue
		}
		if !providerBool(item["use_sub_token"]) {
			continue
		}
		budgetEnabled := isBudgetEligibleProvider(item, params.LLMBudgetEnabled)
		base := normalizeBaseURL(fmt.Sprintf("%v", item["base_url"]))
		return map[string]any{
			"provider":          target,
			"upstream_base_url": base,
			"use_sub_token":     true,
			"budget_enabled":    budgetEnabled,
		}, nil
	}
	return nil, nil
}

// upstreamCredentials mirrors Django upstream_credentials_for_provider.
func upstreamCredentials(tenantID, provider, baseURL string) (map[string]any, error) {
	params, err := loadTenantFeatureParams(tenantID)
	if err != nil {
		return nil, err
	}
	if params == nil {
		return nil, nil
	}
	target := strings.TrimSpace(provider)
	norm := normalizeBaseURL(baseURL)
	for _, item := range parseProvidersJSON(params.ProvidersJSON) {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", item["provider"]))
		if name != target {
			continue
		}
		itemBase := normalizeBaseURL(fmt.Sprintf("%v", item["base_url"]))
		if itemBase != norm {
			continue
		}
		apiKey := strings.TrimSpace(fmt.Sprintf("%v", item["api_key"]))
		if apiKey == "" || apiKey == "<nil>" {
			return nil, nil
		}
		return map[string]any{
			"provider":      target,
			"base_url":      norm,
			"api_key":       apiKey,
			"derive_needed": providerBool(item["use_sub_token"]),
		}, nil
	}
	return nil, nil
}
