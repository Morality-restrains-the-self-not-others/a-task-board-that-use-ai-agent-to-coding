package main

import (
	"net/http"
	"strings"

	"snowflake"
)

// handleInternalInitTenantFeatureParams handles POST /api/internal/cloud/init-tenant-feature-params/
// Idempotent — initializes default TenantFeatureParams for a company.
// Called by taskEvents Go consumer (bypasses Django).
func handleInternalInitTenantFeatureParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	companyID := strings.TrimSpace(strField(body, "company_id"))
	if companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id required")
		return
	}

	// Check if already exists (idempotent)
	existing, err := queryTenantFeatureParamsRow(companyID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"feature_params_id": existing.ID,
			"already_existed":   true,
		})
		return
	}

	// Create with defaults matching Django TenantLlmConfig.default_for_company():
	//   agent_max_steps=200, providers=[], agent_model="", agent_model_provider="",
	//   summary_model="", summary_model_provider="", extra_env_vars=[]
	conn, err := featureParamsConn()
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	id := snowflake.GenerateIDString()
	now := fpNowUTC()
	_, err = conn.Exec(`
		INSERT INTO cloud_tenant_feature_params (
			id, company_id, providers, agent_model, agent_model_provider, agent_max_steps,
			summary_model, summary_model_provider, llm_budget_enabled, extra_env_vars, created_at, updated_at
		) VALUES (?, ?, '[]', '', '', 200, '', '', 0, '[]', ?, ?)`,
		id, companyID, now, now,
	)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"feature_params_id":  id,
		"agent_model":        "",
		"agent_model_provider": "",
		"agent_max_steps":    200,
		"already_existed":    false,
	})
}
