package main

import (
	"net/http"
	"strings"
	"tracelog"
)

// handleSystemAdminLicenseAgreementRouter routes /api/system_admin/license-agreement/...
// 同时兼容连字符形态 /api/system-admin/license-agreement/...（Convention alias，前端统一使用连字符）。
func handleSystemAdminLicenseAgreementRouter(w http.ResponseWriter, r *http.Request) {
	// Remove the base path (underscore or dash form) to get the sub-path
	p := strings.TrimPrefix(r.URL.Path, "/api/system_admin/license-agreement")
	p = strings.TrimPrefix(p, "/api/system-admin/license-agreement")
	p = strings.TrimPrefix(p, "/")

	switch {
	case p == "":
		handleAdminLicenseAgreements(w, r)
	case p == "user-consents/":
		handleLicenseUserConsents(w, r)
	case strings.HasSuffix(p, "/"):
		id := strings.TrimSuffix(p, "/")
		handleAdminLicenseAgreementDetail(w, r, id)
	default:
		handleAdminLicenseAgreementDetail(w, r, p)
	}
}

// handleSystemAdminPrivacyPolicyRouter routes /api/system_admin/privacy-policy/...
// 同时兼容连字符形态 /api/system-admin/privacy-policy/...（Convention alias）。
func handleSystemAdminPrivacyPolicyRouter(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/system_admin/privacy-policy")
	p = strings.TrimPrefix(p, "/api/system-admin/privacy-policy")
	p = strings.TrimPrefix(p, "/")

	switch {
	case p == "":
		handleAdminPrivacyPolicies(w, r)
	case p == "user-consents/":
		handlePrivacyUserConsents(w, r)
	case strings.HasSuffix(p, "/"):
		id := strings.TrimSuffix(p, "/")
		handleAdminPrivacyPolicyDetail(w, r, id)
	default:
		handleAdminPrivacyPolicyDetail(w, r, p)
	}
}

// handleLicenseUserConsents returns user consent records for license agreements.
func handleLicenseUserConsents(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if uid == "" {
		writeErrorJSON(w, http.StatusBadRequest, "缺少 user_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rows, err := db.Query(`
		SELECT ulac.id, ulac.user_id, ulac.license_agreement_id, ulac.consented_at, ulac.context, ulac.client_ip,
			la.title, la.version, la.is_material_change
		FROM billing_user_license_agreement_consents ulac
		JOIN billing_license_agreements la ON la.id = ulac.license_agreement_id
		WHERE ulac.user_id = ?
		ORDER BY ulac.consented_at DESC
		LIMIT 200`, uid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, userID, agreementID, consentedAt, context, clientIP, title, version string
		var isMaterial int
		if err := rows.Scan(&id, &userID, &agreementID, &consentedAt, &context, &clientIP, &title, &version, &isMaterial); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":                   id,
			"user_id":              userID,
			"license_agreement_id": agreementID,
			"license_title":        title,
			"license_version":      version,
			"is_material_change":   isMaterial == 1,
			"context":              context,
			"consented_at":         consentedAt,
			"client_ip":            clientIP,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

// handlePrivacyUserConsents returns user consent records for privacy policies.
func handlePrivacyUserConsents(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if uid == "" {
		writeErrorJSON(w, http.StatusBadRequest, "缺少 user_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rows, err := db.Query(`
		SELECT uppc.id, uppc.user_id, uppc.privacy_policy_id, uppc.consented_at, uppc.context, uppc.client_ip,
			pp.title, pp.version, pp.is_material_change
		FROM billing_user_privacy_policy_consents uppc
		JOIN billing_privacy_policies pp ON pp.id = uppc.privacy_policy_id
		WHERE uppc.user_id = ?
		ORDER BY uppc.consented_at DESC
		LIMIT 200`, uid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, userID, policyID, consentedAt, context, clientIP, title, version string
		var isMaterial int
		if err := rows.Scan(&id, &userID, &policyID, &consentedAt, &context, &clientIP, &title, &version, &isMaterial); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":                 id,
			"user_id":            userID,
			"privacy_policy_id":  policyID,
			"privacy_title":      title,
			"privacy_version":    version,
			"is_material_change": isMaterial == 1,
			"context":            context,
			"consented_at":       consentedAt,
			"client_ip":          clientIP,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}
