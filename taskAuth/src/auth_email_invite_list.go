package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// handleListEmailInvitations lists email registration invitations.
// Admin-only.
// GET /api/system-admin/email-invitations/
func handleListEmailInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		writeError(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	isSuper, err := isSuperAdminUser(userID)
	if err != nil || !isSuper {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}

	if _, err := cleanupExpiredEmailInvites(); err != nil {
		log.Printf("[taskAuth] email invite list expire: %v", err)
	}

	rows, err := db.Query(`
		SELECT i.id, i.email, i.token, i.inviter_user_id, i.status, i.expires_at,
		       COALESCE(i.accepted_user_id, ''),
		       COALESCE(i.email_sent_at, ''),
		       COALESCE(i.email_send_attempts, 0),
		       COALESCE(i.delivery_status, 'pending'),
		       COALESCE(i.delivery_error, ''),
		       COALESCE(i.invite_reason, ''),
		       COALESCE(i.account_expires_at, ''),
		       COALESCE(i.assigned_role, ''),
		       COALESCE(p.username, '') AS inviter_name
		FROM auth_email_registration_invite i
		LEFT JOIN auth_user_profile p ON p.user_id = i.inviter_user_id
		ORDER BY i.created_at DESC LIMIT 100`,
	)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var email, token, inviterID, status, acceptedUID, emailSentAt, deliveryStatus, deliveryError string
		var emailSendAttempts int
		var expiresAt time.Time
		var inviteReason, accountExpiresAt, assignedRole, inviterName string
		if err := rows.Scan(&id, &email, &token, &inviterID, &status, &expiresAt, &acceptedUID, &emailSentAt, &emailSendAttempts, &deliveryStatus, &deliveryError, &inviteReason, &accountExpiresAt, &assignedRole, &inviterName); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id":             fmt.Sprintf("%d", id),
			"email":          email,
			"status":         status,
			"inviterUserId":  inviterID,
			"expiresAt":      expiresAt.Format(time.RFC3339),
			"canResend":      status == "pending",
			"deliveryStatus": deliveryStatus,
		}
		if inviterName != "" {
			entry["inviterName"] = inviterName
		}
		if acceptedUID != "" {
			entry["acceptedUserId"] = acceptedUID
		}
		if emailSentAt != "" {
			entry["emailSentAt"] = emailSentAt
		}
		if deliveryError != "" {
			entry["deliveryError"] = deliveryError
		}
		entry["emailSendAttempts"] = emailSendAttempts
		entry["deliveryAttempts"] = getDeliveryAttempts(id)
		if inviteReason != "" {
			entry["inviteReason"] = inviteReason
		}
		if accountExpiresAt != "" {
			entry["accountExpiresAt"] = accountExpiresAt
		}
		if assignedRole != "" {
			entry["assignedRole"] = assignedRole
		}
		results = append(results, entry)
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"invitations": results})
}
