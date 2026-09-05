package main

import (
	"net/http"
	"strings"

	"authz"
)

// handleAdminTenantOptions GET /api/system_admin/accounts/admin/tenant-options/
// System admin company/tenant search dropdown (used in GrantPoints + OrderRecords pages).
// Supports ?search= for company name, company id, or creator email/phone.
// Each option includes creator email/phone (empty string when missing).
// v63: 平台角色（super_admin/employee）判定取代遗留 X-Auth-Superuser/X-Auth-Staff 头。
func handleAdminTenantOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "superuser or staff required")
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	companies, err := searchCompanies(search, 80)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if search != "" {
		extra := companiesByCreators(searchCreatorIDsFn(search))
		companies = mergeCompanyRows(companies, extra, 80)
	}

	contacts := fetchCreatorContactsFn(creatorIDsFromCompanies(companies))
	out := make([]map[string]interface{}, 0, len(companies))
	for i := range companies {
		item := companyToJSON(&companies[i])
		attachCreatorContacts(item, companies[i].CreatorID, contacts)
		out = append(out, item)
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	logInfo("tenant_options listed count="+itoa(len(out)), traceIDForError(r))
	writeJSON(w, http.StatusOK, out)
}
