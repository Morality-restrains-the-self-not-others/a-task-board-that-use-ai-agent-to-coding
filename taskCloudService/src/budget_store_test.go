package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCalculateSpentCNY(t *testing.T) {
	got, err := calculateSpentCNY(1_000_000, 0, "18", "72")
	if err != nil {
		t.Fatal(err)
	}
	if got != "18.000000" {
		t.Fatalf("got %q", got)
	}
}

func TestRecordModelBudgetUsageDeltaIdempotent(t *testing.T) {
	setupBudgetTestDB(t)
	seedBudgetTenant(t, "c1")
	conn := getBudgetDB()
	item := budgetRecordItem{
		Provider: "openai", BaseURL: "https://api.openai.com/v1", ModelName: "gpt-4.1",
		InputTokensDelta: 1_000_000, OutputTokensDelta: 0, IdempotencyKey: "step-1",
	}
	u1, created1, err := recordModelBudgetUsageDelta(conn, "task1", "w1", "c1", item)
	if err != nil {
		t.Fatal(err)
	}
	if !created1 || u1.SpentAmount != "18.000000" {
		t.Fatalf("first: created=%v spent=%s", created1, u1.SpentAmount)
	}
	u2, created2, err := recordModelBudgetUsageDelta(conn, "task1", "w1", "c1", item)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected idempotent replay")
	}
	if u2.ID != u1.ID || u2.SpentAmount != "18.000000" {
		t.Fatalf("second: %+v", u2)
	}
	var cnt int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM cloud_task_model_budget_usage_idempotency`).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("idem rows=%d", cnt)
	}
}

func TestBudgetReserveOrDenyExhausted(t *testing.T) {
	setupBudgetTestDB(t)
	seedBudgetTenant(t, "c1")
	budget := getBudgetDB()
	_, err := budget.Exec(`
		INSERT INTO cloud_task_model_budget
		(id, todo_id, workspace_id, company_id, provider, base_url, model_name, budget_limit, budget_limit_source)
		VALUES (1, 'task1', 'w1', 'c1', 'openai', 'https://api.openai.com/v1', 'gpt-4.1', '1', 'inherited')
	`)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = recordModelBudgetUsageDelta(budget, "task1", "w1", "c1", budgetRecordItem{
		Provider: "openai", BaseURL: "https://api.openai.com/v1", ModelName: "gpt-4.1",
		InputTokensDelta: 1_000_000, OutputTokensDelta: 0, IdempotencyKey: "k-ex",
	})
	if err != nil {
		t.Fatal(err)
	}
	dec, err := checkBudgetGate(budget, "task1", "w1", "c1", "openai", "https://api.openai.com/v1", "gpt-4.1")
	if err != nil {
		t.Fatal(err)
	}
	if dec["allowed"] != false || dec["code"] != "budget_exhausted" {
		t.Fatalf("dec=%v", dec)
	}
}

func TestFeatureParamsEnvCompanyDefault(t *testing.T) {
	setupBudgetTestDB(t)
	seedBudgetTenant(t, "c1")
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)
	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task1", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env[envAgentModel] != "gpt-4.1" {
		t.Fatalf("env=%v", out.Env)
	}
	if out.Env[envScope] != "company" {
		t.Fatalf("scope=%s", out.Env[envScope])
	}
	if out.Source != "company" && out.Source != "company_default" {
		t.Fatalf("source=%s", out.Source)
	}
}

func TestHandleModelBudgetUsageWritesLocal(t *testing.T) {
	setupBudgetTestDB(t)
	seedBudgetTenant(t, "t1")
	rec := httptest.NewRecorder()
	handleModelBudgetUsage(
		rec, nil,
		map[string]any{
			"items": []any{
				map[string]any{
					"provider":            "openai",
					"base_url":            "https://api.openai.com/v1",
					"model_name":          "gpt-4.1",
					"idempotency_key":     "inbound-1",
					"input_tokens_delta":  float64(1000),
					"output_tokens_delta": float64(0),
				},
			},
		},
		"t1", "w1", "task1",
	)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"currency":"CNY"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	var n int
	if err := getBudgetDB().QueryRow(`SELECT COUNT(*) FROM cloud_task_model_budget_usage`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("usage rows in budget db=%d", n)
	}
}
