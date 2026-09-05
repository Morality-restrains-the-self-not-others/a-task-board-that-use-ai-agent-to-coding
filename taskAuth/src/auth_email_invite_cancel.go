package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// handleCancelEmailInvitation cancels a pending email invitation.
// Admin-only: requires superuser token.
// DELETE /api/system-admin/email-invitations/{id}/
func handleCancelEmailInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	// Extract invitation ID from path: /api/system-admin/email-invitations/{id}/
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		writeError(w, r, http.StatusBadRequest, "invalid path")
		return
	}
	inviteIDStr := parts[len(parts)-1]
	inviteID := strings.TrimSpace(inviteIDStr)
	if inviteID == "" {
		writeError(w, r, http.StatusBadRequest, "缺少邀请ID")
		return
	}

	// Verify the invitation exists and is still pending
	var email, status string
	err = db.QueryRow(`
		SELECT email, status FROM auth_email_registration_invite
		WHERE id = ?`, inviteID,
	).Scan(&email, &status)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "邀请不存在")
		return
	}
	if err != nil {
		log.Printf("[taskAuth] cancel email invite lookup error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if status != "pending" {
		writeError(w, r, http.StatusBadRequest, "该邀请已不是待处理状态，无法取消")
		return
	}

	// Cancel the invitation
	_, err = db.Exec(`
		UPDATE auth_email_registration_invite
		SET status = 'cancelled'
		WHERE id = ? AND status = 'pending'`, inviteID)
	if err != nil {
		log.Printf("[taskAuth] cancel email invite update error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "取消失败")
		return
	}

	log.Printf("[taskAuth] email invite cancelled: inviteID=%s email=%s by=%s", inviteID, email, userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "邀请已取消",
		"invitation_id": inviteID,
		"email":         email,
	})
}
