package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func fpTestRequest(method, view, context string) *http.Request {
	req := httptest.NewRequest(method, "/api/tenant/c1/feature-params/", nil)
	if view != "" {
		q := req.URL.Query()
		q.Set("view", view)
		req.URL.RawQuery = q.Encode()
	}
	if context != "" {
		req.Header.Set(headerFeatureParamsAccessContext, context)
	}
	if context == "" && view == "" {
		req.Header.Set("Authorization", "Token abc")
	}
	return req
}

func TestResolveFeatureParamsAccessSummary(t *testing.T) {
	d := resolveFeatureParamsAccess(fpTestRequest(http.MethodGet, "summary", ""), fpResourceCompany)
	if !d.Allowed || d.ViewMode != fpViewSummary {
		t.Fatalf("expected summary allowed, got %+v", d)
	}
}

func TestResolveFeatureParamsAccessCompanySettings(t *testing.T) {
	d := resolveFeatureParamsAccess(fpTestRequest(http.MethodGet, "", fpContextCompanySettings), fpResourceCompany)
	if !d.Allowed || d.ViewMode != fpViewFull {
		t.Fatalf("expected full allowed, got %+v", d)
	}
}

func TestResolveFeatureParamsAccessWrongContext(t *testing.T) {
	d := resolveFeatureParamsAccess(fpTestRequest(http.MethodGet, "", fpContextWorkspaceSettings), fpResourceCompany)
	if d.Allowed || d.ViewMode != fpViewDenied {
		t.Fatalf("expected denied, got %+v", d)
	}
}

func TestResolveFeatureParamsAccessMissingDenied(t *testing.T) {
	d := resolveFeatureParamsAccess(fpTestRequest(http.MethodGet, "", ""), fpResourceCompany)
	if d.Allowed {
		t.Fatalf("expected denied without context, got %+v", d)
	}
}

func TestRedactFeatureParamsPayloadNested(t *testing.T) {
	payload := map[string]any{
		"providers":      []any{map[string]any{"provider": "openai", "api_key": "sk"}},
		"extra_env_vars": []any{map[string]any{"key": "A", "value": "1"}},
		"env_preview":    map[string]any{"OPENAI_API_KEY": "sk"},
		"company_config": map[string]any{
			"providers":      []any{map[string]any{"provider": "x", "api_key": "y"}},
			"extra_env_vars": []any{map[string]any{"key": "B", "value": "2"}},
		},
	}
	out := redactFeatureParamsPayload(payload)
	providers, _ := out["providers"].([]map[string]any)
	if len(providers) == 0 || providers[0]["api_key"] != "" {
		t.Fatalf("expected redacted api_key, got %+v", out["providers"])
	}
	extras, _ := out["extra_env_vars"].([]any)
	if len(extras) != 0 {
		t.Fatalf("expected empty extra_env_vars, got %v", extras)
	}
	if _, ok := out["env_preview"]; ok {
		t.Fatal("env_preview should be removed")
	}
	co, _ := out["company_config"].(map[string]any)
	coProviders, _ := co["providers"].([]map[string]any)
	if len(coProviders) == 0 || coProviders[0]["api_key"] != "" {
		t.Fatalf("expected nested redact, got %+v", co)
	}
}

func TestDetectFeatureParamsAuthMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/feature-params/", nil)
	req.Header.Set("Authorization", "Token abc")
	if got := detectFeatureParamsAuthMethod(req); got != "token" {
		t.Fatalf("auth method=%q want token", got)
	}
}
