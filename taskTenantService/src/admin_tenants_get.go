package main

import (
	"net/http"
	"strings"
)

// handleAdminTenantGet GET /api/system-admin/accounts/admin/tenants/{id}/
func handleAdminTenantGet(w http.ResponseWriter, r *http.Request) {
	if !requireAdminTenantsAccess(w, r) {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "tenant id required")
		return
	}
	company, err := getCompanyByID(id)
	if err != nil {
		logError("admin_tenant_get db error", traceIDForError(r))
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if company == nil {
		logInfo("admin_tenant_get not_found id_len="+itoa(len(id)), traceIDForError(r))
		writeErrorDetail(w, r, http.StatusNotFound, "tenant not found")
		return
	}
	contacts := fetchCreatorContactsFn([]string{company.CreatorID})
	item := companyToJSON(company)
	attachCreatorContacts(item, company.CreatorID, contacts)
	logInfo("admin_tenant_get ok id_len="+itoa(len(id)), traceIDForError(r))
	writeJSON(w, http.StatusOK, item)
}
