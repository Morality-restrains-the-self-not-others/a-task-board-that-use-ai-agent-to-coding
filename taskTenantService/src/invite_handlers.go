package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"snowflake"
	"strings"
	"time"
)

func genInviteToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// companyNameForEvent returns the tenant_company display name for the INVITATION_CREATED
// event payload, falling back to the company id when the row is missing.
// OPT-20260809-015: taskEvents 消费者渲染邮件需要 company_name；此前发布端只带 company_id
// 导致邮件永远卡「投递中」并死信到 invitation-created-dlt。
func companyNameForEvent(companyID string) string {
	if c, err := getCompanyByID(companyID); err == nil && c != nil && strings.TrimSpace(c.Name) != "" {
		return strings.TrimSpace(c.Name)
	}
	return companyID
}

func handleInvite(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	if !requireCompanyAdmin(w, r, userID, tenantID) {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	inviteMethod := strField(body, "invite_method")
	if inviteMethod == "" {
		inviteMethod = "link"
	}
	email := strField(body, "email")
	phone := strField(body, "phone")
	memberName := strField(body, "company_member_name")
	role := strField(body, "role")
	if role == "" {
		role = "member"
	}
	message := strField(body, "message")
	workspaceID := strField(body, "workspace_id")
	expirationDays, expErr := normalizeInviteExpirationDays(intField(body, "expiration_days", defaultInviteExpirationDays), true)
	if expErr != nil {
		writeError(w, r, http.StatusBadRequest, expErr.Error())
		return
	}
	linkKind, maxUses, kindErr := parseInviteLinkKind(body, inviteMethod)
	if kindErr != nil {
		msg := kindErr.Error()
		if msg == "open_invite_link_only" {
			msg = "开放式邀请仅支持复制邀请链接"
		}
		writeError(w, r, http.StatusBadRequest, msg)
		return
	}

	switch inviteMethod {
	case "email", "phone", "link":
	default:
		writeError(w, r, http.StatusBadRequest, "invite_method 仅支持 email、phone、link")
		return
	}
	if workspaceID == "" {
		writeError(w, r, http.StatusBadRequest, "发送邀请前必须指定工作空间")
		return
	}
	if inviteMethod == "email" && email == "" {
		writeError(w, r, http.StatusBadRequest, "邮箱不能为空")
		return
	}
	if inviteMethod == "phone" && phone == "" {
		writeError(w, r, http.StatusBadRequest, "电话号码不能为空")
		return
	}

	grants, err := parseInviteGrants(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	// 管理员入职走 tenant_admin，忽略预授
	if role == "admin" {
		grants = nil
	}
	pendingJSON, err := encodePendingGrantsJSON(grants)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "grants 序列化失败")
		return
	}

	// v75 角色优先: 邀请可选绑可复用角色（管理员走 tenant_admin，忽略）
	roleNames, err := parseInviteRoleNames(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if role == "admin" {
		roleNames = nil
	}
	if len(roleNames) > 0 {
		if vErr := validateRolesExist(tenantID, roleNames); vErr != nil {
			writeError(w, r, http.StatusBadRequest, vErr.Error())
			return
		}
	}
	pendingRoleJSON, err := encodePendingRoleNamesJSON(roleNames)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "role_names 序列化失败")
		return
	}

	inviteTarget := ""
	if inviteMethod == "email" {
		inviteTarget = email
	} else if inviteMethod == "phone" {
		inviteTarget = phone
	}

	token := genInviteToken()
	expires := time.Now().UTC().Add(time.Duration(expirationDays) * 24 * time.Hour)
	id := snowflake.GenerateIDString()
	isAdmin := 0
	if role == "admin" {
		isAdmin = 1
	}
	// 投递状态初始值：link 渠道无投递动作（none）；email/phone 事件已发布待异步投递（queued）
	deliveryStatus := "none"
	emailSkipped := false
	unsubscribeURL := ""
	if inviteMethod == "email" || inviteMethod == "phone" {
		deliveryStatus = "queued"
	}
	if inviteMethod == "email" {
		var unsub bool
		unsub, unsubscribeURL = checkEmailUnsubscribedFn(email)
		if unsub {
			emailSkipped = true
			deliveryStatus = "skipped_unsubscribed"
		}
	}
	_, err = db.Exec(`
		INSERT INTO tenant_invitation
		(id, company_id, is_admin, workspace_id, invite_method, invite_target,
		 invitation_token, invitation_token_expires_at, is_accepted, company_member_name,
		 delivery_status, pending_grants, pending_role_names, link_kind, max_uses, use_count,
		 created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,0,?,?,?,?,?,?,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		id, tenantID, isAdmin, workspaceID, inviteMethod, inviteTarget, token, expires.Format("2006-01-02 15:04:05"), memberName,
		deliveryStatus, pendingJSON, pendingRoleJSON, linkKind, maxUses,
	)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "创建邀请失败")
		return
	}

	invitationURL := fmt.Sprintf("%s/tenant/%s/people/join/?token=%s", cfg.FrontendBase, tenantID, token)
	if emailSkipped {
		slog.Info("invite_email_skipped", "level", "info", "reason", "unsubscribed")
	}
	eventPayload := map[string]interface{}{
		"invitation_id":       id,
		"company_id":          tenantID,
		"company_name":        companyNameForEvent(tenantID),
		"email":               email,
		"phone":               phone,
		"company_member_name": memberName,
		"invite_method":       inviteMethod,
		"role":                role,
		"workspace_id":        workspaceID,
		"invitation_url":      invitationURL,
		"expiration_days":     expirationDays,
		"expires_at":          expires.Format("2006-01-02 15:04:05"),
		"message":             message,
		"email_skipped":       emailSkipped,
		"link_kind":           linkKind,
		"max_uses":            maxUses,
	}
	if unsubscribeURL != "" {
		eventPayload["unsubscribe_url"] = unsubscribeURL
	}
	if len(grants) > 0 {
		eventPayload["grants"] = grants
	}
	if len(roleNames) > 0 {
		eventPayload["role_names"] = roleNames
	}
	publishEvent("INVITATION_CREATED", eventPayload)

	slog.Info("invite_created",
		"level", "info",
		"link_kind", linkKind,
		"max_uses", maxUses,
		"invite_method", inviteMethod,
		"expiration_days", expirationDays,
		"token_fp", inviteTokenFingerprint(token),
	)
	resp := map[string]interface{}{
		"message":         "邀请链接已生成",
		"invite_token":    token,
		"expires_at":      expires.Format("2006-01-02 15:04:05"),
		"expiration_days": expirationDays,
		"invitation_url":  invitationURL,
		"link_kind":       linkKind,
		"max_uses":        maxUses,
		"use_count":       0,
	}
	if emailSkipped {
		resp["message"] = emailUnsubscribedHint
		resp["code"] = "email_unsubscribed"
		resp["email_skipped"] = true
	}
	writeJSON(w, http.StatusCreated, resp)
}

func handleValidateInvite(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{"valid": false, "error": "邀请链接无效"})
		return
	}
	var expires, linkKind string
	var maxUses, useCount, isAccepted int
	err := db.QueryRow(`
		SELECT invitation_token_expires_at, COALESCE(link_kind,'single'), max_uses, use_count, is_accepted
		FROM tenant_invitation
		WHERE invitation_token=? AND company_id=? AND invitation_token_expires_at > CURRENT_TIMESTAMP`,
		token, tenantID,
	).Scan(&expires, &linkKind, &maxUses, &useCount, &isAccepted)
	if err != nil || isAccepted == 1 || !inviteHasRemainingUses(maxUses, useCount) {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{"valid": false, "error": "邀请链接无效或已过期"})
		return
	}
	resp := map[string]interface{}{
		"valid": true, "message": "邀请链接有效",
		"link_kind": linkKind, "max_uses": maxUses, "use_count": useCount,
		"remaining_uses": remainingInviteUses(maxUses, useCount),
	}
	writeJSON(w, http.StatusOK, resp)
}

func handlePendingInvitations(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
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
	rows, err := db.Query(`
		SELECT id, invitation_token, invitation_token_expires_at, is_admin, created_at, is_accepted,
		       workspace_id, COALESCE(company_member_name,''), invite_method, invite_target,
		       COALESCE(delivery_status,'none'), COALESCE(delivery_error,''),
		       COALESCE(DATE_FORMAT(email_sent_at, '%Y-%m-%d %H:%i:%s'),''),
		       COALESCE(link_kind,'single'), max_uses, use_count
		FROM tenant_invitation
		WHERE company_id=? AND invitation_token IS NOT NULL AND invitation_token != ''
		  AND invitation_token_expires_at > CURRENT_TIMESTAMP AND is_accepted=0
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "查询失败")
		return
	}
	defer rows.Close()
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, token, expires, created, wsID, name, method, target, deliveryStatus, deliveryError, emailSentAt, linkKind string
		var isAdmin, accepted, maxUses, useCount int
		if err := rows.Scan(&id, &token, &expires, &isAdmin, &created, &accepted, &wsID, &name, &method, &target, &deliveryStatus, &deliveryError, &emailSentAt, &linkKind, &maxUses, &useCount); err != nil {
			continue
		}
		item := map[string]interface{}{
			"id": id, "token": token, "expires_at": expires, "is_admin": isAdmin == 1,
			"created_at": created, "is_accepted": accepted == 1,
			"workspace_name":      workspaceName(r.Context(), tenantID, wsID),
			"company_member_name": name, "invite_method": method, "invite_target": target,
			"delivery_status": deliveryStatus, "delivery_error": deliveryError,
			"email_sent_at": emailSentAt,
			"link_kind":     linkKind, "max_uses": maxUses, "use_count": useCount,
		}
		// 兼容：老数据无投递状态列（迁移前创建的邀请）→ 按渠道回退语义
		if deliveryStatus == "none" && method != "link" {
			item["delivery_status"] = "queued"
		}
		list = append(list, item)
	}
	writeJSON(w, http.StatusOK, list)
}

func handleRevokeInvitation(w http.ResponseWriter, r *http.Request, tenantID, inviteID string) {
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
	var token string
	err := db.QueryRow(`SELECT COALESCE(invitation_token,'') FROM tenant_invitation WHERE id=? AND company_id=?`, inviteID, tenantID).Scan(&token)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "邀请记录不存在")
		return
	}
	if token == "" {
		writeError(w, r, http.StatusBadRequest, "此记录不是邀请")
		return
	}
	_, _ = db.Exec(`UPDATE tenant_invitation SET invitation_token=NULL, invitation_token_expires_at=NULL, updated_at=CURRENT_TIMESTAMP WHERE id=?`, inviteID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "邀请已撤销"})
}
