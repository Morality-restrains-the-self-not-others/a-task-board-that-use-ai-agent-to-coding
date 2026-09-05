package main

import (
	"net/http"
	"strconv"
	"strings"

	"authz"
)

const adminTenantListSearchCap = 500

func requireAdminTenantsAccess(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		logWarn("admin_tenants unauthorized missing gateway verified", traceIDForError(r))
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return false
	}
	if !authz.IsPlatformStaff(r) {
		logWarn("admin_tenants forbidden not platform staff", traceIDForError(r))
		writeErrorDetail(w, r, http.StatusForbidden, "superuser or staff required")
		return false
	}
	return true
}

// handleAdminTenants GET /api/system-admin/accounts/admin/tenants/
// Paginated company directory for the system-admin users page 租户 tab.
// Search semantics match tenant-options (name / id / creator email or phone).
func handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if !requireAdminTenantsAccess(w, r) {
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	limit := parseBoundInt(r.URL.Query().Get("limit"), 50, 1, 100)
	offset := parseBoundInt(r.URL.Query().Get("offset"), 0, 0, 1000000)

	companies, total, err := listAdminTenants(search, limit, offset)
	if err != nil {
		logError("admin_tenants db error", traceIDForError(r))
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	contacts := fetchCreatorContactsFn(creatorIDsFromCompanies(companies))
	items := make([]map[string]interface{}, 0, len(companies))
	for i := range companies {
		item := companyToJSON(&companies[i])
		attachCreatorContacts(item, companies[i].CreatorID, contacts)
		items = append(items, item)
	}

	logInfo(
		"admin_tenants listed count="+itoa(len(items))+
			" total="+itoa(total)+
			" limit="+itoa(limit)+
			" offset="+itoa(offset)+
			" search_len="+itoa(len(search)),
		traceIDForError(r),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func listAdminTenants(search string, limit, offset int) ([]companyRow, int, error) {
	if search == "" {
		total, err := countCompanies("")
		if err != nil {
			return nil, 0, err
		}
		rows, err := searchCompaniesPage("", limit, offset)
		if err != nil {
			return nil, 0, err
		}
		return rows, total, nil
	}
	merged, err := searchCompanies(search, adminTenantListSearchCap)
	if err != nil {
		return nil, 0, err
	}
	extra := companiesByCreators(searchCreatorIDsFn(search))
	merged = mergeCompanyRows(merged, extra, adminTenantListSearchCap)
	return sliceCompanyPage(merged, offset, limit), len(merged), nil
}

func sliceCompanyPage(rows []companyRow, offset, limit int) []companyRow {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return []companyRow{}
	}
	if offset >= len(rows) {
		return []companyRow{}
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func parseBoundInt(raw string, fallback, min, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
