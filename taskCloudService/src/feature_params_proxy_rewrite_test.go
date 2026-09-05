package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCoerceProvidersBaseURLsInEnvRepairsConcatenatedBaseURL(t *testing.T) {
	env := map[string]string{
		envProvidersJSON: `[{"provider":"deepseek","api_key":"sk","base_url":"https://api.deepseek.com/v1https://api.deepseek.com","use_sub_token":false}]`,
	}
	got := coerceProvidersBaseURLsInEnv(env)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0]["base_url"] != "https://api.deepseek.com" {
		t.Fatalf("base_url=%v", got[0]["base_url"])
	}
	if strings.Contains(env[envProvidersJSON], "/v1https://") {
		t.Fatalf("concat still present: %s", env[envProvidersJSON])
	}
}

func TestCoerceProvidersBaseURLsInEnvKeepsOperatorTypedAnthropicPath(t *testing.T) {
	env := map[string]string{
		envProvidersJSON: `[{"provider":"deepseek","api_key":"sk","base_url":"https://api.deepseek.com/anthropic","use_sub_token":false}]`,
	}
	got := coerceProvidersBaseURLsInEnv(env)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0]["base_url"] != "https://api.deepseek.com/anthropic" {
		t.Fatalf("base_url=%v", got[0]["base_url"])
	}
}

func TestRewriteSubTokenProvidersForProxy(t *testing.T) {
	out := rewriteSubTokenProvidersForProxy(
		[]map[string]any{
			{
				"provider":       "deepseek",
				"api_key":        "sk-master",
				"base_url":       "https://api.deepseek.com/v1",
				"use_sub_token":  true,
				"budget_enabled": true,
			},
			{
				"provider":      "openai",
				"api_key":       "sk-master2",
				"base_url":      "https://api.openai.com/v1",
				"use_sub_token": false,
			},
		},
		"t1", "w1", "k1", "proxy-tok", "http://127.0.0.1:8013",
	)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0]["api_key"] != "proxy-tok" {
		t.Fatalf("api_key=%v", out[0]["api_key"])
	}
	if out[0]["proxy_mode"] != "task_ai_endpoint" {
		t.Fatalf("proxy_mode=%v", out[0]["proxy_mode"])
	}
	baseURL := fmt.Sprintf("%v", out[0]["base_url"])
	if !strings.Contains(baseURL, "/llm/deepseek/v1") {
		t.Fatalf("base_url=%s", baseURL)
	}
	wantURL := "http://127.0.0.1:8013/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1"
	if baseURL != wantURL {
		t.Fatalf("base_url=%s want=%s", baseURL, wantURL)
	}
	if out[1]["api_key"] != "sk-master2" {
		t.Fatalf("master api_key mutated: %v", out[1]["api_key"])
	}
	if _, ok := out[1]["proxy_mode"]; ok {
		t.Fatalf("master should not have proxy_mode")
	}
}

func TestRewriteSubTokenProvidersSkipsEmptyProvider(t *testing.T) {
	out := rewriteSubTokenProvidersForProxy(
		[]map[string]any{
			{"provider": "", "api_key": "sk", "use_sub_token": true, "base_url": "https://x"},
		},
		"t1", "w1", "k1", "tok", "http://gw",
	)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0]["api_key"] != "sk" {
		t.Fatalf("empty provider should not rewrite api_key")
	}
}

func TestFeatureParamsEnvProxyRewriteAndApiKeyUsage(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"deepseek","api_key":"sk-master","base_url":"https://api.deepseek.com/v1","supported_models":["m1"],"use_sub_token":true,"budget_enabled":true},{"provider":"openai","api_key":"sk-master2","base_url":"https://api.openai.com/v1","supported_models":["gpt"],"use_sub_token":false}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "c1", ProvidersJSON: providers,
		AgentModel: "m1", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)

	t.Setenv("TASK_AI_ENDPOINT_ENABLED", "true")
	t.Setenv("TASK_AI_ENDPOINT_PUBLIC_BASE", "http://127.0.0.1:8013")

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-proxy", "c1", "proxy-tok")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env["TASK_AI_ENDPOINT_BASE_URL"] != "http://127.0.0.1:8013" {
		t.Fatalf("base=%s", out.Env["TASK_AI_ENDPOINT_BASE_URL"])
	}
	if out.Env["TASK_LLM_PROXY_TOKEN"] != "proxy-tok" {
		t.Fatalf("token=%s", out.Env["TASK_LLM_PROXY_TOKEN"])
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out.Env[envProvidersJSON]), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("providers=%v", got)
	}
	if got[0]["api_key"] != "proxy-tok" || got[0]["proxy_mode"] != "task_ai_endpoint" {
		t.Fatalf("sub rewritten=%v", got[0])
	}
	if !strings.Contains(fmt.Sprintf("%v", got[0]["base_url"]), "/llm/deepseek/v1") {
		t.Fatalf("base_url=%v", got[0]["base_url"])
	}
	if got[1]["api_key"] != "sk-master2" {
		t.Fatalf("master mutated=%v", got[1])
	}

	n := store.apiKeyUsageCount("task-proxy")
	if n != 2 {
		t.Fatalf("usage rows=%d", n)
	}
	var keyType, hash string
	for _, row := range store.apiKeyUsageRows("task-proxy") {
		if fmt.Sprintf("%v", row["provider_name"]) == "deepseek" {
			keyType = fmt.Sprintf("%v", row["key_type"])
			hash = fmt.Sprintf("%v", row["api_key_hash"])
		}
	}
	if keyType != "sub" {
		t.Fatalf("key_type=%s", keyType)
	}
	sum := sha256.Sum256([]byte("proxy-tok"))
	wantHash := fmt.Sprintf("%x", sum)
	if hash != wantHash {
		t.Fatalf("hash=%s want=%s", hash, wantHash)
	}
}

func TestFeatureParamsEnvProxyRewriteFromConf(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"deepseek","api_key":"sk-master","base_url":"https://api.deepseek.com/v1","supported_models":["m1"],"use_sub_token":true}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "c1", ProvidersJSON: providers,
		AgentModel: "m1", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)

	prevEnabled := cfg.TaskAIEndpointEnabled
	prevBase := cfg.TaskAIEndpointPublicBase
	cfg.TaskAIEndpointEnabled = true
	cfg.TaskAIEndpointPublicBase = "http://127.0.0.1:8013"
	t.Cleanup(func() {
		cfg.TaskAIEndpointEnabled = prevEnabled
		cfg.TaskAIEndpointPublicBase = prevBase
	})
	_ = os.Unsetenv("TASK_AI_ENDPOINT_ENABLED")
	_ = os.Unsetenv("TASK_AI_ENDPOINT_PUBLIC_BASE")

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-conf-proxy", "c1", "proxy-tok")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env["TASK_AI_ENDPOINT_BASE_URL"] != "http://127.0.0.1:8013" {
		t.Fatalf("base=%s", out.Env["TASK_AI_ENDPOINT_BASE_URL"])
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out.Env[envProvidersJSON]), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["api_key"] != "proxy-tok" || got[0]["proxy_mode"] != "task_ai_endpoint" {
		t.Fatalf("conf-enabled rewrite failed: %v", got)
	}
}

func TestFeatureParamsEnvNoRewriteWithoutEndpoint(t *testing.T) {
	store := setupBudgetTestDB(t)
	providers := `[{"provider":"deepseek","api_key":"sk-master","base_url":"https://api.deepseek.com/v1","use_sub_token":true}]`
	store.putTenant(&tenantFeatureParamsRow{
		ID: "1", CompanyID: "c1", ProvidersJSON: providers,
		AgentModel: "m1", AgentProvider: "deepseek", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	})
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)
	t.Setenv("TASK_AI_ENDPOINT_ENABLED", "false")
	_ = os.Unsetenv("TASK_AI_ENDPOINT_PUBLIC_BASE")

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-norewrite", "c1", "proxy-tok")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.Env[envProvidersJSON], "proxy-tok") {
		t.Fatalf("should not rewrite when endpoint disabled: %s", out.Env[envProvidersJSON])
	}
	n := store.apiKeyUsageCount("task-norewrite")
	if n != 1 {
		t.Fatalf("usage should still record, got %d", n)
	}
}
