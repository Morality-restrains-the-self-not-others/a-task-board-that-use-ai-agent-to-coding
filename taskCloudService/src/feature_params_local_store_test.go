package main

import (
	"testing"
)

func setupFeatureParamsLocalTestDB(t *testing.T) {
	t.Helper()
	setupCloudTestDB(t)
}

func TestUpsertAndLoadTenantFeatureParamsLocal(t *testing.T) {
	setupFeatureParamsLocalTestDB(t)
	params, status, err := upsertTenantFeatureParamsLocal(map[string]interface{}{
		"company_id":           "co-1",
		"providers":            []map[string]any{{"provider": "openai", "api_key": "sk-test"}},
		"agent_model":          "gpt-4.1",
		"agent_model_provider": "openai",
		"agent_max_steps":      100,
		"llm_budget_enabled":   false,
		"extra_env_vars":       []map[string]any{},
	})
	if err != nil || status != 200 {
		t.Fatalf("upsert tenant: status=%d err=%v", status, err)
	}
	if strField(params, "company_id") != "co-1" {
		t.Fatalf("params=%v", params)
	}
	providers, ok := params["providers"].([]any)
	if !ok || len(providers) == 0 {
		t.Fatalf("expected parsed providers array, got %T %v", params["providers"], params["providers"])
	}
	row, err := loadTenantFeatureParams("co-1")
	if err != nil {
		t.Fatal(err)
	}
	if row == nil || row.AgentModel != "gpt-4.1" || row.AgentMaxSteps != 100 {
		t.Fatalf("row=%+v", row)
	}
}

func TestPersonalFeatureParamsListAndDeleteLocal(t *testing.T) {
	setupFeatureParamsLocalTestDB(t)
	created, status, err := upsertPersonalFeatureParamsLocal(map[string]interface{}{
		"user_id":              "u-local",
		"company_id":           "co-1",
		"name":                 "daily",
		"providers":            []map[string]any{},
		"agent_model":          "m1",
		"agent_model_provider": "p1",
		"agent_max_steps":      200,
		"extra_env_vars":       []map[string]any{},
	})
	if err != nil || status != 201 {
		t.Fatalf("create personal: status=%d err=%v", status, err)
	}
	configID := strField(created, "id")
	items, err := listPersonalFeatureParamsLocal("u-local")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || strField(items[0], "id") != configID {
		t.Fatalf("items=%v", items)
	}
	delStatus, err := deletePersonalFeatureParamsLocal(configID, "u-local")
	if err != nil || delStatus != 200 {
		t.Fatalf("delete: status=%d err=%v", delStatus, err)
	}
	items, err = listPersonalFeatureParamsLocal("u-local")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list after delete, got %v", items)
	}
}

func TestWriteFeatureParamsSnapshotLocal(t *testing.T) {
	setupFeatureParamsLocalTestDB(t)
	cfg := &featureParamsConfig{
		ID: "cfg-1", ProvidersJSON: `[{"provider":"openai","api_key":"sk"}]`,
		AgentModel: "gpt-4.1", AgentProvider: "openai", AgentMaxSteps: 200, ExtraEnvJSON: "[]",
	}
	env := map[string]string{
		envAgentModel: "gpt-4.1", envAgentProvider: "openai", envAgentMaxSteps: "200",
	}
	meta := featureParamsSnapshotMeta{Source: "company", SourceConfigID: "cfg-1", SourceDisplayName: "公司默认"}
	if err := writeFeatureParamsSnapshot("task-local", "w1", "co-1", meta, env, cfg); err != nil {
		t.Fatal(err)
	}
	n, err := countFeatureParamsSnapshots("task-local")
	if err != nil || n != 1 {
		t.Fatalf("snapshot count=%d err=%v", n, err)
	}
}
