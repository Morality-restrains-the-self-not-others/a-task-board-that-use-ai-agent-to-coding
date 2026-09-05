package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"taskAuth/domain"
)

// --- Admin: resend email invitation ---

// handleResendEmailInvitation resends the email for an existing pending invitation.
// Admin-only: requires superuser token.
// POST /api/system-admin/email-invitations/resend/
func handleResendEmailInvitation(w http.ResponseWriter, r *http.Request) {
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

	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	email := strings.TrimSpace(strField(body, "email"))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, r, http.StatusBadRequest, "请输入有效的邮箱地址")
		return
	}

	// Find the existing pending invitation
	var inviteID int64
	var token string
	var expiresAt time.Time
	var inviteReason, accountExpiresAt, assignedRole string
	err = db.QueryRow(`
		SELECT id, token, expires_at, COALESCE(invite_reason, ''), COALESCE(account_expires_at, ''), COALESCE(assigned_role, '')
		FROM auth_email_registration_invite
		WHERE email = ? AND status = 'pending' AND expires_at > ?
		LIMIT 1`, email, timeNowUTC(),
	).Scan(&inviteID, &token, &expiresAt, &inviteReason, &accountExpiresAt, &assignedRole)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "未找到该邮箱的有效待处理邀请")
		return
	}
	if err != nil {
		log.Printf("[taskAuth] resend email invite lookup error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}

	// Resend the email — publishEmailSent 内部包含 Kafka 优先 + SMTP 回退
	frontendBase := cfg.FrontendBase
	if frontendBase == "" {
		frontendBase = "http://localhost:4000"
	}
	inviteURL := fmt.Sprintf("%s/auth/register/?invite_token=%s", frontendBase, token)
	if domain.ShouldSkipInviteEmail(isEmailUnsubscribed(email)) {
		_, _ = db.Exec(`UPDATE auth_email_registration_invite SET delivery_status = ? WHERE id = ?`,
			"skipped_unsubscribed", inviteID)
		slog.InfoContext(r.Context(), "invite_email_skipped", "level", "info", "reason", "unsubscribed")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":        domain.EmailUnsubscribedHint,
			"code":           "email_unsubscribed",
			"email_skipped":  true,
			"email":          email,
			"invite_token":   token,
			"invitation_url": inviteURL,
			"delivered":      false,
		})
		return
	}
	resendCtx := map[string]interface{}{
		"invite_url":         inviteURL,
		"email":              email,
		"invitation_id":      fmt.Sprintf("%d", inviteID),
		"invite_reason":      inviteReason,
		"account_expires_at": accountExpiresAt,
		"assigned_role":      assignedRole,
	}
	attachUnsubscribeURL(resendCtx, email)
	method, publishErr := publishEmailSent(r.Context(), email, "您已获得SaaS平台注册邀请", "email_registration_invite", resendCtx)

	// deliveryStatus: "delivered"=SMTP同步确认送达, "queued"=Kafka入队待投递, "failed"=投递失败
	deliveryStatus := "queued" // default to queued (Kafka async)
	var deliveryError string
	if publishErr != nil {
		deliveryStatus = "failed"
		deliveryError = publishErr.Error()
	} else if method == "smtp" {
		// SMTP direct send — delivery is confirmed
		deliveryStatus = "delivered"
	}
	// else: method == "kafka" → deliveryStatus stays "queued"

	// Record delivery attempt history
	methodStr := string(method)
	if publishErr != nil {
		methodStr = "fallback"
	}
	var currentAttempts int
	_ = db.QueryRow(`SELECT COALESCE(email_send_attempts, 0) FROM auth_email_registration_invite WHERE id = ?`, inviteID).Scan(&currentAttempts)
	recordDeliveryAttempt(inviteID, currentAttempts+1, methodStr, deliveryStatus, deliveryError)

	// Update tracking: increment send attempts, update sent_at, delivery_status and delivery_error
	var dbErr error
	if deliveryError != "" {
		_, dbErr = db.Exec(`
			UPDATE auth_email_registration_invite
			SET email_sent_at = ?, email_send_attempts = email_send_attempts + 1, delivery_status = ?, delivery_error = ?
			WHERE id = ?`, timeNowUTC(), deliveryStatus, deliveryError, inviteID)
	} else {
		_, dbErr = db.Exec(`
			UPDATE auth_email_registration_invite
			SET email_sent_at = ?, email_send_attempts = email_send_attempts + 1, delivery_status = ?
			WHERE id = ?`, timeNowUTC(), deliveryStatus, inviteID)
	}
	if dbErr != nil {
		log.Printf("[taskAuth] resend email invite: tracking update failed for inviteID=%d: %v", inviteID, dbErr)
	}

	if publishErr != nil {
		// Kafka 和 SMTP 回退均失败 — 邮件未发送
		log.Printf("[taskAuth] resend email invite: both Kafka and SMTP fallback failed for %s: %v", email, publishErr)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "邀请邮件发送失败，请稍后重试或联系管理员",
			"email":      email,
			"expires_at": expiresAt.Format("2006-01-02 15:04:05.000000"),
			"queued":     false,
			"delivered":  false,
		})
		return
	}

	delivered := deliveryStatus == "delivered"
	queued := deliveryStatus == "queued"
	log.Printf("[taskAuth] email invite resent: email=%s inviteID=%d by=%s method=%s", email, inviteID, userID, method)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "邀请邮件已发送",
		"email":      email,
		"expires_at": expiresAt.Format("2006-01-02 15:04:05.000000"),
		"delivered":  delivered,
		"queued":     queued,
	})
}

// --- Admin: cancel email invitation ---
