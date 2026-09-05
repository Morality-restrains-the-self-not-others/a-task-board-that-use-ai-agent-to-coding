package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

func disableReferrerCommissionEligibility(referrerUserID, reason string) (edges int64, voided int64, err error) {
	referrerUserID = strings.TrimSpace(referrerUserID)
	if referrerUserID == "" {
		return 0, 0, fmt.Errorf("referrer_user_id required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "qualification_revoked"
	}
	now := utcNow()
	res, err := db.Exec(`
		UPDATE billing_referral_edge
		SET commission_eligible = 0, updated_at = ?
		WHERE referrer_user_id = ? AND commission_eligible <> 0`,
		now, referrerUserID,
	)
	if err != nil {
		return 0, 0, err
	}
	edges, _ = res.RowsAffected()
	res, err = db.Exec(`
		UPDATE billing_referral_commission_accrual
		SET status = ?, voided_at = ?, void_reason = ?, updated_at = ?
		WHERE referrer_user_id = ? AND status = ?`,
		referralAccrualVoided, now, reason, now, referrerUserID, referralAccrualPending,
	)
	if err != nil {
		return edges, 0, err
	}
	voided, _ = res.RowsAffected()
	return edges, voided, nil
}

func handleInternalDisableReferralEligibility(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	referrer := strings.TrimSpace(stringField(body, "referrer_user_id"))
	reason := strings.TrimSpace(stringField(body, "reason"))
	edges, voided, err := disableReferrerCommissionEligibility(referrer, reason)
	if err != nil {
		slog.Warn("referral disable eligibility failed",
			"level", "warn",
			"referrer_user_id", referrer,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	slog.Info("referral disable eligibility",
		"level", "info",
		"referrer_user_id", referrer,
		"edges", edges,
		"voided", voided,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"edges_updated": edges,
		"voided":        voided,
	})
}
