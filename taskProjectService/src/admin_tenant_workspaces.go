package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"authz"
)

func conventionPathValue(path, key string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == key && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func parseAdminPageBound(raw string, fallback, min, max int) int {
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

// handleAdminTenantWorkspaces GET /api/system-admin/tenant-workspaces/tenant_id/{id}/
// Platform staff list all workspaces for a tenant (no membership mine filter).
func handleAdminTenantWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		log.Printf("[taskProjectService] WARN admin_tenant_workspaces unauthorized")
		writeError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	if !authz.IsPlatformStaff(r) {
		log.Printf("[taskProjectService] WARN admin_tenant_workspaces forbidden not platform staff")
		writeError(w, r, http.StatusForbidden, "superuser or staff required")
		return
	}
	tenantID := strings.TrimSpace(conventionPathValue(r.URL.Path, "tenant_id"))
	if tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "tenant_id required")
		return
	}
	limit := parseAdminPageBound(r.URL.Query().Get("limit"), 50, 1, 100)
	offset := parseAdminPageBound(r.URL.Query().Get("offset"), 0, 0, 1000000)

	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM project_workspace_entries WHERE company_id=?`,
		tenantID,
	).Scan(&total); err != nil {
		log.Printf("[taskProjectService] admin_tenant_workspaces count error: %v", err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := db.Query(
		`SELECT id, name, is_default, created_at FROM project_workspace_entries
		 WHERE company_id=? ORDER BY name LIMIT ? OFFSET ?`,
		tenantID, limit, offset,
	)
	if err != nil {
		log.Printf("[taskProjectService] admin_tenant_workspaces query error: %v", err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name string
		var isDef bool
		var created time.Time
		if err := rows.Scan(&id, &name, &isDef, &created); err != nil {
			log.Printf("[taskProjectService] admin_tenant_workspaces scan error: %v", err)
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		items = append(items, map[string]interface{}{
			"id":         id,
			"name":       name,
			"is_default": isDef,
			"created_at": created.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
