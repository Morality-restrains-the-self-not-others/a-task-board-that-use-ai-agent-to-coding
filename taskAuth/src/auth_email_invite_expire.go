package main

import (
	"log/slog"
	"net/http"

	"tracelog"
)

const emailInvitesExpireDuePath = "/api/internal/taskauth/email-invites/expire-due/"

// cleanupExpiredEmailInvites marks due pending super-admin email invites as expired.
// Idempotent: already-expired / non-pending rows are left unchanged.
func cleanupExpiredEmailInvites() (int64, error) {
	result, err := db.Exec(`
		UPDATE auth_email_registration_invite
		SET status = 'expired'
		WHERE status = 'pending' AND expires_at <= ?`, timeNowUTC())
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}

// handleInternalEmailInvitesExpireDue POST /api/internal/taskauth/email-invites/expire-due/
// One-shot sweep for taskEvents timer (OPT-20260829-003). Not a process loop.
func handleInternalEmailInvitesExpireDue(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		slog.WarnContext(r.Context(), "email_invite_expire_due_forbidden",
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	n, err := cleanupExpiredEmailInvites()
	if err != nil {
		slog.ErrorContext(r.Context(), "email_invite_expire_due",
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusInternalServerError, "expire failed")
		return
	}
	slog.InfoContext(r.Context(), "email_invite_expire_due_ok",
		"expired", n,
		"trace_id", tracelog.TraceIDFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]interface{}{"expired": n})
}
