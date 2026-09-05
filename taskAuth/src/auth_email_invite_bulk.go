package main

import (
	"fmt"
	"log"
	"net/http"

	"taskAuth/domain"
)

// --- Admin: bulk resend email invitations ---

// handleBulkResendEmailInvitations resends email for all failed or queued invitations.
// Admin-only: requires superuser token.
// POST /api/system-admin/email-invitations/bulk-resend/
func handleBulkResendEmailInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		writeError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !isSuper {
		writeError(w, r, http.StatusForbidden, "仅管理员可操作")
		return
	}

	// Find all pending invitations with delivery issues (failed or queued)
	rows, err := db.Query(`
		SELECT id, email, token, COALESCE(email_send_attempts, 0),
			       COALESCE(invite_reason, ''), COALESCE(account_expires_at, ''), COALESCE(assigned_role, '')
		FROM auth_email_registration_invite
		WHERE status = 'pending' AND expires_at > ?
		ORDER BY created_at DESC LIMIT 100`, timeNowUTC(),
	)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type resendTarget struct {
		id               int64
		email            string
		token            string
		attempts         int
		inviteReason     string
		accountExpiresAt string
		assignedRole     string
	}
	var targets []resendTarget
	for rows.Next() {
		var t resendTarget
		if err := rows.Scan(&t.id, &t.email, &t.token, &t.attempts, &t.inviteReason, &t.accountExpiresAt, &t.assignedRole); err != nil {
			continue
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "没有待处理的邀请",
			"count":   0,
		})
		return
	}

	frontendBase := cfg.FrontendBase
	if frontendBase == "" {
		frontendBase = "http://localhost:4000"
	}

	successCount := 0
	failCount := 0
	var failedEmails []string

	for _, t := range targets {
		inviteURL := fmt.Sprintf("%s/auth/register/?invite_token=%s", frontendBase, t.token)
		if domain.ShouldSkipInviteEmail(isEmailUnsubscribed(t.email)) {
			_, _ = db.Exec(`UPDATE auth_email_registration_invite SET delivery_status = ? WHERE id = ?`,
				"skipped_unsubscribed", t.id)
			successCount++
			continue
		}
		bulkCtx := map[string]interface{}{
			"invite_url":         inviteURL,
			"email":              t.email,
			"invitation_id":      fmt.Sprintf("%d", t.id),
			"invite_reason":      t.inviteReason,
			"account_expires_at": t.accountExpiresAt,
			"assigned_role":      t.assignedRole,
		}
		attachUnsubscribeURL(bulkCtx, t.email)
		method, sendErr := publishEmailSent(r.Context(), t.email, "您已获得SaaS平台注册邀请", "email_registration_invite", bulkCtx)

		deliveryStatus := "queued"
		var deliveryError string
		if sendErr != nil {
			deliveryStatus = "failed"
			deliveryError = sendErr.Error()
			failCount++
			failedEmails = append(failedEmails, t.email)
		} else if method == "smtp" {
			deliveryStatus = "delivered"
			successCount++
		} else {
			successCount++
		}

		methodStr := string(method)
		if sendErr != nil {
			methodStr = "fallback"
		}
		recordDeliveryAttempt(t.id, t.attempts+1, methodStr, deliveryStatus, deliveryError)

		// Update the invitation record
		if deliveryError != "" {
			_, _ = db.Exec(`
				UPDATE auth_email_registration_invite
				SET email_sent_at = ?, email_send_attempts = email_send_attempts + 1, delivery_status = ?, delivery_error = ?
				WHERE id = ?`, timeNowUTC(), deliveryStatus, deliveryError, t.id)
		} else {
			_, _ = db.Exec(`
				UPDATE auth_email_registration_invite
				SET email_sent_at = ?, email_send_attempts = email_send_attempts + 1, delivery_status = ?
				WHERE id = ?`, timeNowUTC(), deliveryStatus, t.id)
		}
	}

	log.Printf("[taskAuth] bulk resend: %d succeeded, %d failed out of %d (by=%s)", successCount, failCount, len(targets), userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       fmt.Sprintf("批量重发完成：%d 成功，%d 失败", successCount, failCount),
		"total":         len(targets),
		"success_count": successCount,
		"fail_count":    failCount,
		"failed_emails": failedEmails,
	})
}
