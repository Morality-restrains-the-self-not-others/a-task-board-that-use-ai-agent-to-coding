package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkspaceBudgetDefaultsCRUD(t *testing.T) {
	setupBudgetTestDB(t)
	cfg.InternalSecret = ""

	upsertBody := `{
		"workspace_id":"w1","company_id":"c1",
		"items":[{"provider":"openai","base_url":"https://api.openai.com/v1","model_name":"gpt-4.1",
			"input_price_per_1m":"18","output_price_per_1m":"72","budget_limit":"50"}]
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/budget/workspace-defaults/upsert/", strings.NewReader(upsertBody))
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("upsert status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/internal/budget/workspace-defaults/?workspace_id=w1&company_id=c1", nil)
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	items, _ := listed["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%v", listed)
	}

	delBody := `{
		"workspace_id":"w1","company_id":"c1",
		"items":[{"provider":"openai","base_url":"https://api.openai.com/v1","model_name":"gpt-4.1"}]
	}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/internal/budget/workspace-defaults/delete/", strings.NewReader(delBody))
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	rows, err := listWorkspaceBudgetDefaults(getBudgetDB(), "w1", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(rows))
	}
}

func TestTaskBudgetConfigAndUsageList(t *testing.T) {
	setupBudgetTestDB(t)
	cfg.InternalSecret = ""
	seedBudgetTenant(t, "c1")

	upsertBody := `{
		"todo_id":"task1","workspace_id":"w1","company_id":"c1",
		"items":[{"provider":"openai","base_url":"https://api.openai.com/v1","model_name":"gpt-4.1",
			"budget_limit":"80","budget_limit_source":"override"}]
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/budget/task-budgets/upsert/", strings.NewReader(upsertBody))
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("upsert status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/internal/budget/task-budgets/?todo_id=task1", nil)
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list budgets status=%d body=%s", rec.Code, rec.Body.String())
	}

	_, _, err := recordModelBudgetUsageDelta(getBudgetDB(), "task1", "w1", "c1", budgetRecordItem{
		Provider: "openai", BaseURL: "https://api.openai.com/v1", ModelName: "gpt-4.1",
		InputTokensDelta: 1000, OutputTokensDelta: 0, IdempotencyKey: "cfg-usage-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/internal/budget/task-usage/?todo_id=task1", nil)
	handleInternalBudgetRoutes(rec, req)
	if rec.Code != 200 {
		t.Fatalf("usage list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("usage items=%v", body)
	}
}
