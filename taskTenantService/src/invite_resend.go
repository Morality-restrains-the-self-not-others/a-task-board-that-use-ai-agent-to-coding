package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func handleResendInvitation(w http.ResponseWriter, r *http.Request, tenantID, inviteID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if _, ok := requireCompanyMember(w, r, userID, tenantID); !ok {
		return
	}
	body, _ := readJSONBody(r)
	expirationDays, expErr := normalizeInviteExpirationDays(intField(body, "expiration_days", defaultInviteExpirationDays), false)
	if expErr != nil {
		writeError(w, r, http.StatusBadRequest, expErr.Error())
		return
	}
	var method, target, memberName, workspaceID string
	var isAdmin, accepted int
	err := db.QueryRow(`
		SELECT invite_method, invite_target, COALESCE(company_member_name,''), workspace_id, is_admin, is_accepted
		FROM tenant_invitation WHERE id=? AND company_id=?`, inviteID, tenantID,
	).Scan(&method, &target, &memberName, &workspaceID, &isAdmin, &accepted)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "邀请记录不存在")
		return
	}
	if accepted == 1 {
		writeError(w, r, http.StatusBadRequest, "邀请已被接受，无法重新发送")
		return
	}
	token := genInviteToken()
	expires := time.Now().UTC().Add(time.Duration(expirationDays) * 24 * time.Hour)
	// 重发即重新投递：重置为 queued 等待 taskEvents 消费者回写结果
	_, _ = db.Exec(`UPDATE tenant_invitation SET invitation_token=?, invitation_token_expires_at=?,
		delivery_status='queued', delivery_error=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		token, expires.Format("2006-01-02 15:04:05"), inviteID)
	role := "member"
	if isAdmin == 1 {
		role = "admin"
	}
	email, phone := "", ""
	if method == "email" {
		email = target
	} else if method == "phone" {
		phone = target
	}
	invitationURL := fmt.Sprintf("%s/tenant/%s/people/join/?token=%s", cfg.FrontendBase, tenantID, token)
	emailSkipped := false
	unsubscribeURL := ""
	if method == "email" {
		var unsub bool
		unsub, unsubscribeURL = checkEmailUnsubscribedFn(email)
		if unsub {
			emailSkipped = true
			_, _ = db.Exec(`UPDATE tenant_invitation SET delivery_status='skipped_unsubscribed' WHERE id=?`, inviteID)
			slog.Info("invite_email_skipped", "level", "info", "reason", "unsubscribed")
		}
	}
	payload := map[string]interface{}{
		"invitation_id": inviteID, "company_id": tenantID, "company_name": companyNameForEvent(tenantID),
		"email": email, "phone": phone,
		"company_member_name": memberName, "invite_method": method, "role": role,
		"workspace_id": workspaceID, "invitation_token": token, "invitation_url": invitationURL,
		"expiration_days": expirationDays, "expires_at": expires.Format("2006-01-02 15:04:05"),
		"message":       strField(body, "message"),
		"email_skipped": emailSkipped,
	}
	if unsubscribeURL != "" {
		payload["unsubscribe_url"] = unsubscribeURL
	}
	publishEvent("INVITATION_CREATED", payload)
	resp := map[string]interface{}{
		"message":      "邀请链接已重新生成并重新发送",
		"invite_token": token, "invite_url": invitationURL,
		"expires_at": expires.Format("2006-01-02 15:04:05"), "expiration_days": expirationDays,
	}
	if emailSkipped {
		resp["message"] = emailUnsubscribedHint
		resp["code"] = "email_unsubscribed"
		resp["email_skipped"] = true
		resp["invitation_url"] = invitationURL
	}
	writeJSON(w, http.StatusOK, resp)
}
