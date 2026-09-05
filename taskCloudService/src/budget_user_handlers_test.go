package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gatewayauth"
)

func TestUserWorkspaceModelBudgetDefaults(t *testing.T) {
	store := setupBudgetTestDB(t)
	cfg.GatewayInternalSecret = "gw-secret"
	store.putMember("c1", "admin-u", "m-admin", true)
	store.putMember("c1", "u1", "m1", false)
	seedBudgetTenant(t, "c1")

	// mock project service workspace
	prevProj := cfg.ProjectServiceURL
	projSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/internal/workspaces/") {
			writeJSON(w, 200, map[string]any{"id": "w1", "company_id": "c1", "name": "WS"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			writeJSON(w, 200, []map[string]any{})
			return
		}
		writeJSON(w, 404, map[string]string{"error": "not found"})
	}))
	t.Cleanup(projSrv.Close)
	cfg.ProjectServiceURL = projSrv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prevProj })

	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/model-budget-defaults/tenant_id/c1/workspace_id/w1/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET defaults status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 eligible model, got %v", body)
	}

	patch := `{"items":[{"provider":"openai","base_url":"https://api.openai.com/v1","model_name":"gpt-4.1","input_price_per_1m":"10","output_price_per_1m":"20","budget_limit":"100"}]}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/cloud/model-budget-defaults/tenant_id/c1/workspace_id/w1/", strings.NewReader(patch))
	req.Header.Set(gatewayauth.HeaderUserID, "admin-u")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("PATCH defaults status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUserTaskModelBudgetsAndRaise(t *testing.T) {
	store := setupBudgetTestDB(t)
	cfg.GatewayInternalSecret = "gw-secret"
	store.putMember("c1", "admin-u", "m-admin", true)
	store.putMember("c1", "u1", "m1", false)
	seedBudgetTenant(t, "c1")

	prevTask := cfg.TaskServiceURL
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"id": "task1", "tenant_id": "c1", "workspace_id": "w1", "owner_id": "m1",
		})
	}))
	t.Cleanup(taskSrv.Close)
	cfg.TaskServiceURL = taskSrv.URL
	t.Cleanup(func() { cfg.TaskServiceURL = prevTask })

	prevProj := cfg.ProjectServiceURL
	projSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, []map[string]any{})
	}))
	t.Cleanup(projSrv.Close)
	cfg.ProjectServiceURL = projSrv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prevProj })

	// seed a task budget row
	lim := "10"
	if _, err := upsertTaskModelBudget(getBudgetDB(), taskModelBudgetRow{
		TodoID: "task1", WorkspaceID: "w1", CompanyID: "c1",
		Provider: "openai", BaseURL: "https://api.openai.com/v1", ModelName: "gpt-4.1",
		BudgetLimit: &lim, BudgetLimitSource: "inherited",
	}); err != nil {
		t.Fatal(err)
	}
	_, _ = upsertTenantBudgetPermission(getBudgetDB(), tenantBudgetPermissionRow{
		CompanyID: "c1", SubjectType: budgetSubjectMember, SubjectID: "m1", CanRaiseTaskBudget: true,
	})

	mux := http.NewServeMux()
	mountRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/model-budgets/tenant_id/c1/workspace_id/w1/task_id/task1/", nil)
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET budgets status=%d body=%s", rec.Code, rec.Body.String())
	}

	raiseBody := `{"provider":"openai","base_url":"https://api.openai.com/v1","model_name":"gpt-4.1","new_budget_limit":"20","confirm":true}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/cloud/model-budgets/raise/tenant_id/c1/workspace_id/w1/task_id/task1/", strings.NewReader(raiseBody))
	req.Header.Set(gatewayauth.HeaderUserID, "u1")
	req.Header.Set(gatewayauth.HeaderGatewayVerified, "1")
	req.Header.Set(gatewayauth.HeaderGatewaySecret, cfg.GatewayInternalSecret)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("raise status=%d body=%s", rec.Code, rec.Body.String())
	}
	rows, err := listTaskModelBudgets(getBudgetDB(), "task1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].BudgetLimitSource != "raised" || rows[0].BudgetLimit == nil || *rows[0].BudgetLimit != "20" {
		t.Fatalf("rows=%+v", rows)
	}
}
