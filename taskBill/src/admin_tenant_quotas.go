package main

import (
	"log/slog"
	"net/http"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminTenantQuotas GET /api/system-admin/tenant-quotas/tenant_id/{id}/
func handleSystemAdminTenantQuotas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		slog.WarnContext(r.Context(), "admin_tenant_quotas_unauthorized",
			"level", "warn",
			"reason", "missing_gateway_verify",
		)
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		slog.WarnContext(r.Context(), "admin_tenant_quotas_forbidden",
			"level", "warn",
			"reason", "not_platform_staff",
		)
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		slog.WarnContext(r.Context(), "admin_tenant_quotas_bad_tenant_id",
			"level", "warn",
		)
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	slog.InfoContext(r.Context(), "admin_tenant_quotas_ok",
		"tenant_id", formatID(tid),
	)
	writeResourceQuotas(w, r, tid)
}
