package main

import (
	"net/http"
	"strings"
)

// handleInternalTenantBudgetEnabled is a lightweight internal endpoint that returns
// llm_budget_enabled for a given company. Used by taskTenantService member-handler
// to avoid the Django HTTP hop (OPT-026 Phase 3).
func handleInternalTenantBudgetEnabled(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id required")
		return
	}
	row, err := loadTenantFeatureParams(companyID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	enabled := false
	if row != nil {
		enabled = row.LLMBudgetEnabled
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"found":              row != nil,
		"llm_budget_enabled": enabled,
	})
}

func handleInternalFeatureParamsEnv(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var tenantID, workspaceID, taskID, companyID, proxyToken string
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		tenantID = strings.TrimSpace(q.Get("tenant_id"))
		workspaceID = strings.TrimSpace(q.Get("workspace_id"))
		taskID = strings.TrimSpace(q.Get("task_id"))
		companyID = strings.TrimSpace(q.Get("company_id"))
		if companyID == "" {
			companyID = tenantID
		}
		proxyToken = strings.TrimSpace(q.Get("proxy_token"))
	} else {
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
			return
		}
		tenantID = strField(body, "tenant_id")
		workspaceID = strField(body, "workspace_id")
		taskID = strField(body, "task_id")
		companyID = strField(body, "company_id")
		if companyID == "" {
			companyID = tenantID
		}
		proxyToken = strField(body, "proxy_token")
	}
	if tenantID == "" || workspaceID == "" || taskID == "" || companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id, workspace_id, task_id, company_id required")
		return
	}
	out, err := resolveFeatureParamsEnvLocal(r.Context(), getBudgetDB(), tenantID, workspaceID, taskID, companyID, proxyToken)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}
