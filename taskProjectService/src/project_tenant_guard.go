package main

import (
	"fmt"
	"net/http"
	"strings"
)

func requestProjectTenantID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if tid := strings.TrimSpace(requestTenantID(r)); tid != "" {
		return tid
	}
	return strings.TrimSpace(getAuthTenant(r))
}

func projectCompanyID(detail map[string]interface{}) string {
	if detail == nil {
		return ""
	}
	cid, _ := detail["company"].(string)
	return strings.TrimSpace(cid)
}

// rejectIfProjectNotInTenant 404s when the loaded project belongs to another
// company than the URL/header tenant. Empty tenant (legacy unit tests that
// call handlers without convention path) is fail-open; production /api/projects/
// always injects tenant_id before dispatch.
func rejectIfProjectNotInTenant(w http.ResponseWriter, r *http.Request, detail map[string]interface{}, tenantID string) bool {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		tid = requestProjectTenantID(r)
	}
	cid := projectCompanyID(detail)
	if tid == "" || cid == "" || cid == tid {
		return false
	}
	pid := ""
	if detail != nil {
		pid = strings.TrimSpace(fmt.Sprint(detail["id"]))
	}
	logWarn(
		fmt.Sprintf("project tenant mismatch id=%s project_company=%s url_tenant=%s", pid, cid, tid),
		r.Header.Get("X-Trace-Id"),
	)
	writeError(w, r, http.StatusNotFound, "project not found")
	return true
}
