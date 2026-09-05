package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"taskAuth/domain"
)

// --- DB types ---

type emailInviteRow struct {
	ID                int64
	Email             string
	Token             string
	InviterUserID     string
	InviterName       string
	Status            string
	ExpiresAt         time.Time
	AcceptedUserID    sql.NullString
	EmailSentAt       sql.NullString
	EmailSendAttempts int
	DeliveryStatus    string
	DeliveryError     sql.NullString
	InviteReason      sql.NullString
	AccountExpiresAt  sql.NullString
	AssignedRole      sql.NullString
}

// --- Delivery attempt history ---
// --- Token generation ---

func generateEmailInviteToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// --- Admin: create email invitation ---

// handleCreateEmailInvitation creates an email registration invitation.
// Admin-only: requires superuser token.
// POST /api/system-admin/email-invitations/
func handleCreateEmailInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Auth check: must be superuser
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
		writeError(w, r, http.StatusForbidden, "仅管理员可发送邀请")
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

	inviteReason := strings.TrimSpace(strField(body, "invite_reason"))
	accountExpiresAt := strings.TrimSpace(strField(body, "account_expires_at"))
	assignedRole := strings.TrimSpace(strField(body, "assigned_role"))

	// Validate assigned_role
	if assignedRole != "" {
		validRoles := map[string]bool{"superuser": true, "staff": true, "tenant": true, "member": true}
		if !validRoles[assignedRole] {
			writeError(w, r, http.StatusBadRequest, "无效的角色类型，可选值: superuser, staff, tenant, member")
			return
		}
	}

	// Check if email already registered
	existingLM, err := findLoginMethodByEmail(email)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if existingLM != nil {
		writeError(w, r, http.StatusBadRequest, "该邮箱已注册")
		return
	}

	// Check for existing pending invite
	var existingInviteID int64
	var existingExpiresAt string
	err = db.QueryRow(`
		SELECT id, expires_at FROM auth_email_registration_invite
		WHERE email = ? AND status = 'pending' AND expires_at > ?
		LIMIT 1`, email, timeNowUTC(),
	).Scan(&existingInviteID, &existingExpiresAt)
	if err == nil {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":            "该邮箱已有待接受的邀请",
			"invitation_id":    fmt.Sprintf("%d", existingInviteID),
			"invitation_email": email,
			"expires_at":       existingExpiresAt,
			"can_resend":       true,
		})
		return
	}

	token, err := generateEmailInviteToken()
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "token generation failed")
		return
	}

	id := generateSnowflakeID()
	now := timeNowUTC()
	expires := time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05.000000")

	// Convert ISO 8601 datetime (e.g. "2027-12-31T00:00:00Z") to MySQL format if needed
	if accountExpiresAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, accountExpiresAt)
		if parseErr == nil {
			accountExpiresAt = parsed.Format("2006-01-02 15:04:05.000000")
		}
		// If parse fails, pass as-is (MySQL may accept other formats)
	}

	// Convert empty strings to nil for SQL NULL insertion
	var inviteReasonParam, accountExpiresAtParam, assignedRoleParam interface{}
	if inviteReason != "" {
		inviteReasonParam = inviteReason
	}
	if accountExpiresAt != "" {
		accountExpiresAtParam = accountExpiresAt
	}
	if assignedRole != "" {
		assignedRoleParam = assignedRole
	}

	// Insert the invitation record first (delivery_status initially pending)
	_, err = db.Exec(`
		INSERT INTO auth_email_registration_invite
		(id, email, token, inviter_user_id, status, expires_at, created_at, email_sent_at, email_send_attempts, delivery_status,
		 invite_reason, account_expires_at, assigned_role)
		VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, 1, 'pending', ?, ?, ?)`,
		id, email, token, userID, expires, now, now, inviteReasonParam, accountExpiresAtParam, assignedRoleParam,
	)
	if err != nil {
		log.Printf("[taskAuth] create email invite: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "创建邀请失败")
		return
	}

	// Send invitation email — publishEmailSent 内部包含 Kafka 优先 + SMTP 回退
	frontendBase := cfg.FrontendBase
	if frontendBase == "" {
		frontendBase = "http://localhost:4000"
	}
	inviteURL := fmt.Sprintf("%s/auth/register/?invite_token=%s", frontendBase, token)
	if domain.ShouldSkipInviteEmail(isEmailUnsubscribed(email)) {
		_, _ = db.Exec(`UPDATE auth_email_registration_invite SET delivery_status = ?, email_send_attempts = 0 WHERE id = ?`,
			"skipped_unsubscribed", id)
		slog.InfoContext(r.Context(), "invite_email_skipped", "level", "info", "reason", "unsubscribed")
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"message":        domain.EmailUnsubscribedHint,
			"code":           "email_unsubscribed",
			"email_skipped":  true,
			"email":          email,
			"expires":        expires,
			"invite_token":   token,
			"invitation_url": inviteURL,
			"delivered":      false,
		})
		return
	}
	inviteCtx := map[string]interface{}{
		"invite_url":         inviteURL,
		"email":              email,
		"invitation_id":      fmt.Sprintf("%d", id),
		"invite_reason":      inviteReason,
		"account_expires_at": accountExpiresAt,
		"assigned_role":      assignedRole,
	}
	attachUnsubscribeURL(inviteCtx, email)
	// deliveryStatus: "delivered"=SMTP同步确认送达, "queued"=Kafka入队待投递, "failed"=投递失败
	deliveryStatus := "queued" // default to queued (Kafka async)
	var deliveryError string
	method, sendErr := publishEmailSent(r.Context(), email, "您已获得SaaS平台注册邀请", "email_registration_invite", inviteCtx)
	if sendErr != nil {
		log.Printf("[taskAuth] email invite created but delivery failed for %s: %v", email, sendErr)
		deliveryStatus = "failed"
		deliveryError = sendErr.Error()
	} else if method == "smtp" {
		// SMTP direct send — delivery is confirmed
		deliveryStatus = "delivered"
	}
	// else: method == "kafka" → deliveryStatus stays "queued" (message published, consumer will send)

	// Record delivery attempt history
	methodStr := string(method)
	if sendErr != nil {
		methodStr = "fallback"
	}
	recordDeliveryAttempt(id, 1, methodStr, deliveryStatus, deliveryError)

	// Update delivery status and error based on actual send result
	if deliveryError != "" {
		_, _ = db.Exec(`UPDATE auth_email_registration_invite SET delivery_status = ?, delivery_error = ? WHERE id = ?`, deliveryStatus, deliveryError, id)
	} else {
		_, _ = db.Exec(`UPDATE auth_email_registration_invite SET delivery_status = ? WHERE id = ?`, deliveryStatus, id)
	}

	delivered := deliveryStatus == "delivered"
	log.Printf("[taskAuth] email invite created: email=%s token=%s by=%s method=%s delivered=%v", email, token[:8]+"...", userID, method, delivered)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "邀请已发送",
		"email":     email,
		"expires":   expires,
		"delivered": delivered,
	})
}

// --- Admin: list email invitations ---

// --- Public: validate invitation token ---

// handleValidateEmailInvite validates an email invitation token.
// GET /api/public/email-invitation/{token}/
func handleValidateEmailInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := strings.TrimPrefix(r.URL.Path, "/api/public/email-invitation/")
	token = strings.Trim(strings.TrimSuffix(token, "/"), "/")
	if token == "" {
		writeError(w, r, http.StatusBadRequest, "invalid token")
		return
	}

	var email, status string
	var expiresAt time.Time
	err := db.QueryRow(`
		SELECT email, status, expires_at FROM auth_email_registration_invite
		WHERE token = ?`, token,
	).Scan(&email, &status, &expiresAt)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "邀请链接无效")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if status != "pending" {
		writeError(w, r, http.StatusBadRequest, "邀请链接已失效")
		return
	}
	if time.Now().UTC().After(expiresAt) {
		writeError(w, r, http.StatusBadRequest, "邀请链接已过期")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"email":     email,
		"valid":     true,
		"expiresAt": expiresAt.Format(time.RFC3339),
	})
}
