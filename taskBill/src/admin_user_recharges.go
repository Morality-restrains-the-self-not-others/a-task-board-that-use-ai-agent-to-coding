package main

import (
	"fmt"
	"net/http"
	"strings"

	"authz"
	"tracelog"
)

// handleSystemAdminUserRecharges GET /api/system_admin/users/{uid}/recharges/
// System admin view: user's recharge/payment history + associated license consent records.
// v63: 平台角色（super_admin/employee）判定取代遗留 X-Auth-Superuser/X-Auth-Staff 头。
// OPT-049: Migrated from Django 2026-07-30.
// 2026-08-19: admin_grant 标注「无需签署」；用户支付挂载支付服务条款签署。
func handleSystemAdminUserRecharges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// Gateway forward-auth
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// Extract user_id from path: /api/system_admin/users/{uid}/recharges/
	userID := strings.TrimSpace(r.PathValue("uid"))
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// Fetch recharge history
	recharges, err := listUserRecharges(r.Context(), userID, 0)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if recharges == nil {
		recharges = []map[string]interface{}{}
	}

	// Fetch license consents for the user
	consents, err := listUserLicenseConsents(userID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if consents == nil {
		consents = []map[string]interface{}{}
	}

	recharges = enrichRechargesWithConsent(recharges, consents)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recharges": recharges,
		"consents":  consents,
	})
}

// listUserLicenseConsents returns license agreement consent records for a user.
func listUserLicenseConsents(userID string) ([]map[string]interface{}, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return []map[string]interface{}{}, nil
	}
	rows, err := db.Query(`
		SELECT ulac.id, la.version, la.document_kind, ulac.consented_at,
		       COALESCE(la.content, '') AS content_snapshot
		FROM billing_user_license_agreement_consents ulac
		JOIN billing_license_agreements la ON la.id = ulac.license_agreement_id
		WHERE ulac.user_id = ?
		ORDER BY ulac.consented_at DESC
		LIMIT 500`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, version, documentKind, consentedAt, contentSnapshot string
		if err := rows.Scan(&id, &version, &documentKind, &consentedAt, &contentSnapshot); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":                id,
			"agreement_version": version,
			"document_kind":     documentKind,
			"consented_at":      consentedAt,
			"content_snapshot":  contentSnapshot,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, rows.Err()
}

// enrichRechargesWithConsent labels admin_grant rows as not requiring payment-terms consent,
// and attaches the consent actually signed for that payment (billing_resource_order.consent_id,
// OPT-20260825-017). Rows without an order-scoped consent fall back to the newest payment-terms
// consent when present.
func enrichRechargesWithConsent(recharges, consents []map[string]interface{}) []map[string]interface{} {
	var paymentConsent map[string]interface{}
	consentByID := make(map[string]map[string]interface{}, len(consents))
	for _, c := range consents {
		cid, _ := c["id"].(string)
		if cid != "" {
			consentByID[cid] = c
		}
		kind, _ := c["document_kind"].(string)
		if paymentConsent == nil && isPaymentTermsKind(kind) {
			paymentConsent = c
			// consents are ordered newest-first; keep iterating to index all by id
		}
	}

	out := make([]map[string]interface{}, 0, len(recharges))
	for _, row := range recharges {
		enriched := make(map[string]interface{}, len(row)+4)
		for k, v := range row {
			enriched[k] = v
		}
		ptsSrc, _ := enriched["points_source_type"].(string)
		if ptsSrc == "admin_grant" {
			enriched["consent_required"] = false
			enriched["consent_note"] = "系统赠送，无需支付签署"
			if ch, _ := enriched["channel"].(string); strings.TrimSpace(ch) == "" {
				enriched["channel"] = "admin_grant"
			}
			out = append(out, enriched)
			continue
		}
		enriched["consent_required"] = true
		// OPT-20260825-017: 优先挂订单当时签署的 consent；仅无订单 consent 才回退最新支付条款。
		if oid := strings.TrimSpace(stringOf(enriched["order_consent_id"])); oid != "" {
			if c, ok := consentByID[oid]; ok {
				enriched["consent"] = c
				enriched["consent_source"] = "order"
				out = append(out, enriched)
				continue
			}
		}
		if paymentConsent != nil {
			enriched["consent"] = paymentConsent
			enriched["consent_source"] = "fallback"
		}
		out = append(out, enriched)
	}
	return out
}

func stringOf(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}
