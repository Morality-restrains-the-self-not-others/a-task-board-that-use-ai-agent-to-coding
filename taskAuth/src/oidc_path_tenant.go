package main

import (
	"log/slog"
	"net/http"
	"strings"

	"taskAuth/domain"
	"tracelog"
)

func oidcPathTenantID(r *http.Request) string {
	return strings.TrimSpace(r.PathValue("tenant_id"))
}

func rejectOidcPathTenantMismatch(w http.ResponseWriter, r *http.Request, ownerCompanyID, clientID string) bool {
	pathTid := oidcPathTenantID(r)
	if err := domain.CheckOidcPathTenant(pathTid, ownerCompanyID); err != nil {
		slog.WarnContext(r.Context(), "oidc_path_tenant_mismatch",
			"path_tenant_id", pathTid, "owner_company_id", ownerCompanyID,
			"client_id", clientID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusBadRequest, oidcErr(r, map[string]string{
			"error":             "unauthorized_client",
			"error_description": "client does not belong to this tenant",
		}))
		return true
	}
	return false
}

func ownerCompanyIDFromAudience(aud string) string {
	if cid, err := domain.OwnerCompanyIDFromClientID(strings.TrimSpace(aud)); err == nil {
		return cid
	}
	return ""
}

func audienceFromClaims(claims map[string]interface{}) string {
	switch v := claims["aud"].(type) {
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		if len(v) == 0 {
			return ""
		}
		s, _ := v[0].(string)
		return strings.TrimSpace(s)
	default:
		return ""
	}
}
