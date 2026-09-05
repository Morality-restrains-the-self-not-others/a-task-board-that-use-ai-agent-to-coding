package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gatewaycors"
	"tracelog"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// traceIDForError resolves a trace id for error-body injection:
// X-Trace-Id header first (gateway-bridged), then the context trace id.
func traceIDForError(r *http.Request) string {
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		return tid
	}
	return strings.TrimSpace(tracelog.TraceIDFromContext(r.Context()))
}

// writeError writes a uniform {status, error, message, trace_id?} error body.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]interface{}{"status": "error", "error": message, "message": message}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorDetail writes a {status, error, detail, message, trace_id?} error body.
// Frontend reads data.detail first (taskAuth FE error contract), so detail-first.
func writeErrorDetail(w http.ResponseWriter, r *http.Request, status int, detail string) {
	body := map[string]interface{}{
		"status":  "error",
		"error":   detail,
		"detail":  detail,
		"message": detail,
	}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorMap writes a composite error body, injecting trace_id and filling
// in status/message (from error or detail) when absent while preserving any
// extra keys the caller provided.
func writeErrorMap(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	if _, ok := body["status"]; !ok {
		body["status"] = "error"
	}
	if _, ok := body["message"]; !ok {
		if msg, ok := body["error"]; ok {
			body["message"] = msg
		} else if detail, ok := body["detail"]; ok {
			body["message"] = detail
		}
	}
	writeJSON(w, status, body)
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func strField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			// Expose trace headers so frontend JS can read X-Trace-Id for data-traceId binding
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskAuth"})
}

// handleMetrics exposes Prometheus-compatible metrics.
// Includes OIDC health gauges so monitoring can alert on issuer/template/key issues,
// plus HTTP RED metrics (rate/errors/duration) from tracelog.MetricsMiddleware.
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	iss := issuerURL()
	issuerOK := 0
	if iss != "" && (strings.HasPrefix(iss, "http://") || strings.HasPrefix(iss, "https://")) && !strings.Contains(iss, "${") {
		issuerOK = 1
	}
	keyOK := 0
	if signingKey != nil {
		keyOK = 1
	}
	clients := len(cfg.OidcBootstrapClients)

	body := fmt.Sprintf(`# HELP taskauth_oidc_issuer_valid Whether the OIDC issuer URL is well-formed (1=ok, 0=bad).
# TYPE taskauth_oidc_issuer_valid gauge
taskauth_oidc_issuer_valid %d
# HELP taskauth_oidc_signing_key_valid Whether the OIDC signing key is initialized (1=ok, 0=missing).
# TYPE taskauth_oidc_signing_key_valid gauge
taskauth_oidc_signing_key_valid %d
# HELP taskauth_oidc_bootstrap_clients Number of registered OIDC bootstrap clients.
# TYPE taskauth_oidc_bootstrap_clients gauge
taskauth_oidc_bootstrap_clients %d
# HELP taskauth_info Service metadata.
# TYPE taskauth_info gauge
taskauth_info{issuer="%s"} 1
`, issuerOK, keyOK, clients, iss)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))

	// Append HTTP RED metrics (rate / errors / duration) from tracelog middleware.
	tracelog.WriteMetricsBody(w)
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/schema/", handleOpenAPISchema)
	mux.HandleFunc("GET /api/schema", handleOpenAPISchema)
	mux.HandleFunc("GET /api/swagger/", handleSwaggerUI)
	mux.HandleFunc("GET /api/swagger", handleSwaggerUI)
	// 运维用内部/OIDC 规范：不进入网关浏览器门户（upstream docs 仅指向公开 schema）
	mux.HandleFunc("GET /api/schema-internal/", handleOpenAPIInternalSchema)
	mux.HandleFunc("GET /api/schema-internal", handleOpenAPIInternalSchema)
	mux.HandleFunc("GET /api/swagger-internal/", handleSwaggerInternalUI)
	mux.HandleFunc("GET /api/swagger-internal", handleSwaggerInternalUI)
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("GET /api/accounts/users/{user_id}/", handleGetUser)
	// OPT-049: SSO bridge for AI provider (vendor/admin) — migrated from Django 2026-07-30
	mux.HandleFunc("GET /api/accounts/sso/", handleSSOBridge)

	// Legacy Django path bridge — frontend still calls /api/user/{user_id}/accounts/users/me/
	// from 13+ locations. OPT-049: Django saas-backend decommissioned 2026-07-30.
	mux.HandleFunc("GET /api/user/{user_id}/accounts/users/me/", handleGetUserLegacyPath)
	// User profile (migrated from Django saas-backend, retired 2026-07-30)
	mux.HandleFunc("GET /api/accounts/users/profile/", handleGetUserProfile)
	mux.HandleFunc("PATCH /api/accounts/users/profile/", handlePatchUserProfile)
	mux.HandleFunc("POST /api/accounts/users/profile/avatar/", handleUploadAvatar)
	mux.HandleFunc("DELETE /api/accounts/users/profile/avatar/", handleDeleteAvatar)
	// 各公司的设置页：成员头像/复制操作 → 桥接 taskTenantService 内部 API
	mux.HandleFunc("POST /api/accounts/users/profile/company-avatar/", handleUploadCompanyAvatar)
	mux.HandleFunc("DELETE /api/accounts/users/profile/company-avatar/", handleDeleteCompanyAvatar)
	mux.HandleFunc("POST /api/accounts/users/profile/copy-personal-to-company/", handleCopyPersonalToCompany)
	// 资料页绑定/换绑：必须比 POST /profile/{$} 更具体。Go ServeMux 尾斜杠是子树匹配，
	// 否则 POST /profile/bind-phone/ 会落到 handlePublicUpsertProfile → 200 {ok:true} 且不写 login_method。
	mux.HandleFunc("POST /api/accounts/users/profile/bind-phone/", handleBindPhone)
	mux.HandleFunc("POST /api/accounts/users/profile/replace-phone/", handleBindPhone)
	mux.HandleFunc("POST /api/accounts/users/profile/{$}", handlePublicUpsertProfile)
	mux.HandleFunc("POST /api/accounts/users/profile", handlePublicUpsertProfile)
	// Method-less patterns match all methods and conflict with GET /users/{user_id}/
	// (Go 1.22+ ServeMux: more specific path cannot match more methods).
	mux.HandleFunc("GET /api/accounts/users/me/account-deletion/", handleAccountDeletionRouter)
	mux.HandleFunc("POST /api/accounts/users/me/account-deletion/", handleAccountDeletionRouter)
	mux.HandleFunc("GET /api/accounts/users/me/account-deletion", handleAccountDeletionRouter)
	mux.HandleFunc("POST /api/accounts/users/me/account-deletion", handleAccountDeletionRouter)
	// 个人信息导出（PIPL「导出权」；GET+POST 子树显式注册，同注销防 ServeMux 冲突）
	mux.HandleFunc("GET /api/accounts/users/me/personal-data-export/", handlePersonalDataExportRouter)
	mux.HandleFunc("POST /api/accounts/users/me/personal-data-export/", handlePersonalDataExportRouter)
	mux.HandleFunc("GET /api/accounts/users/me/personal-data-export", handlePersonalDataExportRouter)
	mux.HandleFunc("POST /api/accounts/users/me/personal-data-export", handlePersonalDataExportRouter)
	mux.HandleFunc("POST /api/internal/taskauth/account-deletion/execute-due/", handleInternalAccountDeletionExecuteDue)
	mux.HandleFunc("POST /api/internal/taskauth/account-deletion/execute-due", handleInternalAccountDeletionExecuteDue)
	mux.HandleFunc("POST /api/internal/taskauth/login-methods/void-orphans/", handleInternalVoidOrphanLoginMethods)
	mux.HandleFunc("POST /api/internal/taskauth/login-methods/void-orphans", handleInternalVoidOrphanLoginMethods)
	mux.HandleFunc("POST /api/internal/taskauth/oidc-sso-idempotency/cleanup/", handleInternalOidcSsoIdempotencyCleanup)
	mux.HandleFunc("POST /api/internal/taskauth/oidc-sso-idempotency/cleanup", handleInternalOidcSsoIdempotencyCleanup)
	mux.HandleFunc("POST /api/internal/taskauth/wechat-mp-cleanup/", handleInternalWechatMpCleanup)
	mux.HandleFunc("POST /api/internal/taskauth/wechat-mp-cleanup", handleInternalWechatMpCleanup)
	mux.HandleFunc("POST "+emailInvitesExpireDuePath, handleInternalEmailInvitesExpireDue)
	mux.HandleFunc("POST /api/internal/taskauth/email-invites/expire-due", handleInternalEmailInvitesExpireDue)
	mux.HandleFunc("GET /api/internal/users/", handleListUsers)
	mux.HandleFunc("GET /api/internal/users/id/{user_id}/platform-roles/", handleInternalUserPlatformRoles)
	mux.HandleFunc("GET /api/internal/users/id/{user_id}/wechat-pay-openid/", handleInternalWechatPayOpenID)
	mux.HandleFunc("GET /api/internal/users/wechat-linked-account/", handleInternalWechatLinkedAccount)
	mux.HandleFunc("GET /api/internal/user-count/", handleCountUsers)
	mux.HandleFunc("PATCH /api/internal/users/id/{user_id}/", handlePatchUser)
	mux.HandleFunc("POST /api/internal/users/batch/resolve/", handleBatchResolveUsers)
	mux.HandleFunc("POST /api/internal/users/batch/details/", handleBatchUserDetails)
	mux.HandleFunc("POST /api/internal/token/resolve/", handleResolveToken)
	mux.HandleFunc("GET /api/internal/gateway/forward-auth/", handleGatewayForwardAuth)
	mux.HandleFunc("POST /api/internal/gateway/forward-auth/", handleGatewayForwardAuth)
	// ── v63 RBAC: PDP + 角色管理 ──
	mux.HandleFunc("POST /api/internal/authz/check", handlePDPCheck)
	mux.HandleFunc("GET /api/internal/authz/role-exists", handleRoleExists)
	mux.HandleFunc("POST /api/internal/authz/apply-member-grants/", handleApplyMemberGrants)
	mux.HandleFunc("POST /api/auth/roles/", handleCreateRole)
	mux.HandleFunc("PUT /api/auth/roles/role_id/{rid}/", handleUpdateRole)
	mux.HandleFunc("DELETE /api/auth/roles/role_id/{rid}/", handleDeleteRole)
	mux.HandleFunc("GET /api/auth/roles/company_id/{cid}/", handleListRoles)
	mux.HandleFunc("GET /api/auth/resource-groups/", handleListResourceGroups)
	mux.HandleFunc("GET /api/auth/roles/role_id/{rid}/resource-groups/", handleRoleResourceGroups)
	mux.HandleFunc("PUT /api/auth/roles/role_id/{rid}/resource-groups/", handleRoleResourceGroups)
	mux.HandleFunc("POST /api/auth/user-roles/user_id/{uid}/", handleAssignPlatformRole)
	mux.HandleFunc("DELETE /api/auth/user-roles/user_id/{uid}/role_name/{name}/", handleRevokePlatformRole)
	mux.HandleFunc("GET /api/auth/role-users/role_name/{name}/", handleRoleUsers)
	mux.HandleFunc("GET /api/auth/user-roles/", handleMyUserRoles)
	mux.HandleFunc("GET /api/auth/user-permissions/", handleMyPermissions)
	mux.HandleFunc("POST /api/internal/container-gateway/validate-session/", handleContainerGatewayValidateSession)
	mux.HandleFunc("POST /api/internal/container-gateway/validate-session", handleContainerGatewayValidateSession)
	mux.HandleFunc("GET /api/internal/users/id/{user_id}/super-admin/", handleGetSuperAdmin)
	mux.HandleFunc("POST /api/internal/users/lookup-by-email/", handleLookupUserByEmail)
	mux.HandleFunc("POST /api/internal/email-delivery-callback/", handleEmailDeliveryCallback)
	mux.HandleFunc("POST /api/internal/login-methods/username-taken/", handleUsernameTaken)
	mux.HandleFunc("PATCH /api/internal/users/id/{user_id}/username-login-method/", handleUpsertUsernameLoginMethod)
	mux.HandleFunc("POST /api/internal/login-methods/phone-taken/", handlePhoneTaken)
	mux.HandleFunc("PATCH /api/internal/users/id/{user_id}/phone-login-method/", handleUpsertPhoneLoginMethod)
	mux.HandleFunc("POST /api/internal/users/id/{user_id}/profile/", handleUpsertUserProfile)
	mux.HandleFunc("POST /api/auth/", handleLogin)
	mux.HandleFunc("POST /api/auth/admin-login/", handleAdminLogin)
	mux.HandleFunc("GET /api/auth/verify", handleAuthVerify)
	mux.HandleFunc("GET /api/auth/tenant-memberships/", handleTenantMemberships)
	mux.HandleFunc("POST /api/accounts/users/login/", handleLogin)
	mux.HandleFunc("POST /api/accounts/users/activate-session/", handleActivateSession)
	mux.HandleFunc("GET /api/accounts/users/client-ip/", handleClientIP)
	mux.HandleFunc("POST /api/accounts/users/logout/", handleLogout)
	mux.HandleFunc("POST /api/accounts/users/email_register/", handleEmailRegister)
	mux.HandleFunc("POST /api/accounts/users/phone_register/", handlePhoneRegister)
	mux.HandleFunc("POST /api/accounts/users/bind_phone/", handleBindPhone)
	// OPT-20260806-066: 个人资料页邮箱绑定（厂商门户前置，镜像 handleBindPhone 语义）
	mux.HandleFunc("POST /api/accounts/users/bind_email/", handleBindEmail)
	// OPT-049: system feature policy — migrated from Django 2026-07-30
	mux.HandleFunc("GET /api/public/system-feature-policy/", handlePublicSystemFeaturePolicy)
	mux.HandleFunc("/api/system-admin/system-feature-policy/", handleAdminSystemFeaturePolicy)

	mux.HandleFunc("GET /api/public/registration-invite-policy/", handlePublicRegistrationInvitePolicy)
	mux.HandleFunc("GET /api/system-admin/registration-invite-policy/", handleSystemAdminRegistrationInvitePolicy)
	mux.HandleFunc("PUT /api/system-admin/registration-invite-policy/", handleSystemAdminRegistrationInvitePolicy)
	mux.HandleFunc("GET /api/system-admin/registration-invite-relations/", handleSystemAdminRegistrationInviteRelations)
	mux.HandleFunc("POST /api/accounts/users/registration-invite-codes/apply/", handleApplyRegistrationInviteCode)
	mux.HandleFunc("GET /api/accounts/users/registration-invite-codes/", handleListRegistrationInviteCodes)

	// OPT-036: system-admin dashboard + user management
	mux.HandleFunc("GET /api/system-admin/dashboard/", handleSystemAdminDashboard)
	mux.HandleFunc("/api/system-admin/users/", handleSystemAdminUsers)
	mux.HandleFunc("/api/system-admin/users", handleSystemAdminUsers)
	mux.HandleFunc("POST /api/auth/impersonation/stop/", handleStopImpersonation)
	mux.HandleFunc("GET /api/auth/impersonation/status/", handleImpersonationStatus)
	mux.HandleFunc("GET /api/auth/login-history/", handleListOwnLoginHistory)
	mux.HandleFunc("GET /api/auth/inbox/", handleListInbox)
	mux.HandleFunc("PATCH /api/auth/inbox/{id}/read/", handleMarkInboxRead)
	mux.HandleFunc("POST /api/auth/inbox/{id}/read/", handleMarkInboxRead)
	// OPT-20260808-026: 插件 OIDC 白名单管理（GET 现状 / PUT 更新）
	mux.HandleFunc("GET /api/system-admin/oidc-extension/", handleSystemAdminOidcExtension)
	mux.HandleFunc("PUT /api/system-admin/oidc-extension/", handleSystemAdminOidcExtension)

	mux.HandleFunc("POST /api/accounts/users/confirm_activation/", handleConfirmActivationPrefix)
	mux.HandleFunc("POST /api/accounts/users/resend_activation_email/", handleResendActivation)
	mux.HandleFunc("POST /api/accounts/users/send_password_reset_link/", handleSendPasswordResetLink)
	mux.HandleFunc("POST /api/accounts/users/reset-password-with-link/", handleResetPasswordWithLinkPrefix)
	mux.HandleFunc("GET /api/accounts/users/get-reset-user-info/", handleGetResetUserInfoPrefix)
	mux.HandleFunc("POST /api/accounts/users/send_password_reset_code/", handleSendPasswordResetCode)
	mux.HandleFunc("POST /api/accounts/users/reset_password_with_code/", handleResetPasswordWithCode)
	mux.HandleFunc("POST /api/accounts/users/send_verification_code/", handleSendVerificationCode)
	mux.HandleFunc("POST /api/internal/verification-code/verify/", handleInternalVerifyVerificationCode)
	mux.HandleFunc("GET /api/internal/recharge-sms-gate/", handleInternalRechargeSMSGate)
	mux.HandleFunc("POST /api/internal/recharge-sms-gate/", handleInternalRechargeSMSGate)
	mux.HandleFunc("DELETE /api/internal/recharge-sms-gate/", handleInternalRechargeSMSGate)

	mux.HandleFunc("GET /api/internal/kyc/profile/", handleInternalKycProfile)
	mux.HandleFunc("POST /api/internal/kyc/evaluate/", handleInternalKycEvaluate)
	mux.HandleFunc("POST /api/internal/kyc/admin-override/", handleInternalKycAdminOverride)
	mux.HandleFunc("GET /api/internal/kyc/audit/", handleInternalKycAudit)
	mux.HandleFunc("GET /api/internal/kyc/aml-screening/", handleInternalKycAmlScreening)
	mux.HandleFunc("POST /api/internal/kyc/aml-screening/", handleInternalKycAmlScreening)
	mux.HandleFunc("GET /api/internal/kyc/limit-policy/", handleInternalKycLimitPolicy)
	mux.HandleFunc("PUT /api/internal/kyc/limit-policy/", handleInternalKycLimitPolicy)
	mux.HandleFunc("GET /api/internal/kyc/recharge-gate/", handleInternalKycRechargeGate)

	// Public KYC endpoints (authenticated via session/cookie/token, not internal secret)
	mux.HandleFunc("GET /api/kyc/admin/users/{user_id}/", handlePublicKycAdminGet)
	mux.HandleFunc("POST /api/kyc/admin/users/{user_id}/override/", handlePublicKycAdminOverride)
	mux.HandleFunc("POST /api/kyc/admin/users/{user_id}/aml/", handlePublicKycAdminAml)
	mux.HandleFunc("POST /api/kyc/admin/users/{user_id}/evaluate/", handlePublicKycAdminEvaluate)
	mux.HandleFunc("GET /api/kyc/me/", handlePublicKycMe)

	// Access token management
	mux.HandleFunc("POST /api/accounts/users/access-tokens/", handleCreateAccessToken)
	mux.HandleFunc("GET /api/accounts/users/access-tokens/", handleListAccessTokens)
	mux.HandleFunc("DELETE /api/accounts/users/access-tokens/{token_id}/", handleRevokeAccessToken)
	mux.HandleFunc("POST /api/accounts/users/login-with-access-token/", handleLoginWithAccessToken)
	mux.HandleFunc("POST /api/internal/access-token/resolve/", handleInternalResolveAccessToken)

	// OIDC health check (validates issuer, templates, signing key)
	mux.HandleFunc("GET /api/health/oidc", handleOidcHealth)

	// Prometheus metrics (lightweight — no external library dependency)
	mux.HandleFunc("GET /metrics", handleMetrics)

	// 邮箱注册邀请（管理员发起）
	mux.HandleFunc("POST /api/system-admin/email-invitations/", handleCreateEmailInvitation)
	mux.HandleFunc("GET /api/system-admin/email-invitations/", handleListEmailInvitations)
	mux.HandleFunc("POST /api/system-admin/email-invitations/resend/", handleResendEmailInvitation)
	mux.HandleFunc("DELETE /api/system-admin/email-invitations/{id}/", handleCancelEmailInvitation)
	mux.HandleFunc("POST /api/system-admin/email-invitations/bulk-resend/", handleBulkResendEmailInvitations)
	// 公开：验证邮箱邀请 token
	mux.HandleFunc("GET /api/public/email-invitation/", handleValidateEmailInvite)
	mux.HandleFunc("GET /api/public/email-unsubscribe/", handlePublicEmailUnsubscribe)
	mux.HandleFunc("POST /api/public/email-unsubscribe/", handlePublicEmailUnsubscribe)
	// OPT-20260829-002: 退订确认页「重新接收邀请邮件」入口
	mux.HandleFunc("POST /api/public/email-resubscribe/", handlePublicEmailResubscribe)
	mux.HandleFunc("GET /api/internal/email-unsubscription/", handleInternalEmailUnsubscription)

	// 微信扫码登录 (WeChat OAuth, v64 多应用: web扫码 / inapp公众号授权)
	mux.HandleFunc("GET /api/auth/wechat/login/", handleWeChatLogin)
	mux.HandleFunc("GET /api/auth/wechat/callback/", handleWeChatCallback) // 旧路径别名 (web)
	mux.HandleFunc("GET /api/auth/wechat/web/callback/", handleWeChatCallback)
	mux.HandleFunc("GET /api/auth/wechat/inapp/callback/", handleWeChatCallback)
	// 微信绑定/解绑 (登录态)
	mux.HandleFunc("GET /api/auth/wechat/bind/", handleWeChatBind)
	mux.HandleFunc("GET /api/auth/wechat/bind/callback/", handleWeChatCallback)
	mux.HandleFunc("DELETE /api/auth/wechat/unbind/", handleWeChatUnbind)
	// 服务号关注回调（公众平台服务器 URL；独立于扫码 OAuth，禁止与 login 限流共用）
	mux.HandleFunc("GET /api/auth/wechat/mp/callback/", handleWeChatMPCallback)
	mux.HandleFunc("POST /api/auth/wechat/mp/callback/", handleWeChatMPCallback)
	mux.HandleFunc("GET /api/auth/wechat/mp/follow-status/", handleWeChatMPFollowStatus)
	mux.HandleFunc("POST /api/auth/wechat/mp/follow-qr/", handleWeChatMPFollowQR)

	// OIDC Provider endpoints
	mux.HandleFunc("GET /.well-known/openid-configuration", handleOidcDiscovery)
	mux.HandleFunc("GET /api/oidc/authorize", handleOidcAuthorize)
	mux.HandleFunc("POST /api/oidc/token", handleOidcToken)
	mux.HandleFunc("GET /api/oidc/userinfo", handleOidcUserInfo)
	mux.HandleFunc("POST /api/oidc/userinfo", handleOidcUserInfo)
	mux.HandleFunc("GET /api/oidc/jwks", handleOidcJWKS)
	mux.HandleFunc("GET /api/oidc/endsession", handleOidcEndSession)
	mux.HandleFunc("GET /api/oidc/{tenant_id}/authorize", handleOidcAuthorize)
	mux.HandleFunc("POST /api/oidc/{tenant_id}/token", handleOidcToken)
	mux.HandleFunc("GET /api/oidc/{tenant_id}/userinfo", handleOidcUserInfo)
	mux.HandleFunc("POST /api/oidc/{tenant_id}/userinfo", handleOidcUserInfo)
	mux.HandleFunc("GET /api/oidc/{tenant_id}/jwks", handleOidcJWKS)

	mux.HandleFunc("GET /api/tenant/{tenant_id}/gitlab-oidc-sso/", handleTenantGitLabOidcSsoGet)
	mux.HandleFunc("GET /api/tenant/{tenant_id}/gitlab-oidc-sso", handleTenantGitLabOidcSsoGet)
	mux.HandleFunc("PUT /api/tenant/{tenant_id}/gitlab-oidc-sso/", handleTenantGitLabOidcSsoPut)
	mux.HandleFunc("PUT /api/tenant/{tenant_id}/gitlab-oidc-sso", handleTenantGitLabOidcSsoPut)
	mux.HandleFunc("POST /api/tenant/{tenant_id}/gitlab-oidc-sso/rotate/", handleTenantGitLabOidcSsoRotate)
	mux.HandleFunc("POST /api/tenant/{tenant_id}/gitlab-oidc-sso/rotate", handleTenantGitLabOidcSsoRotate)
	mux.HandleFunc("DELETE /api/tenant/{tenant_id}/gitlab-oidc-sso/", handleTenantGitLabOidcSsoDelete)
	mux.HandleFunc("DELETE /api/tenant/{tenant_id}/gitlab-oidc-sso", handleTenantGitLabOidcSsoDelete)
}

func handleConfirmActivationPrefix(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/users/confirm_activation/")
	token := strings.Trim(path, "/")
	handleConfirmActivation(w, r, token)
}

func handleConfirmActivation(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if token == "" {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "激活链接无效",
		})
		return
	}
	valid, lm := isActivationTokenValid(token)
	if !valid || lm == nil {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "激活链接无效或已过期",
		})
		return
	}
	if err := activateLoginMethod(lm.ID); err != nil {
		log.Printf("[taskAuth] activate error: %v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "activate failed")
		return
	}
	// Publish USER_ACTIVATED event (async, non-blocking)
	go publishUserActivated(r.Context(), lm.ObjectID, lm.MethodType, lm.Identifier)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "账号激活成功",
		"email":   lm.Identifier,
	})
}

func handleResendActivation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	email := strField(body, "email")
	if email == "" {
		writeError(w, r, http.StatusBadRequest, "邮箱不能为空")
		return
	}
	lm, err := findLoginMethodByEmail(email)
	if err != nil || lm == nil {
		writeError(w, r, http.StatusNotFound, "该邮箱未注册")
		return
	}
	if lm.IsVerified {
		writeError(w, r, http.StatusBadRequest, "该邮箱已激活，请直接登录")
		return
	}
	token, err := generateActivationToken()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "token generation failed")
		return
	}
	now := timeNowUTC()
	_, err = db.Exec(`
		UPDATE auth_login_method SET activation_token = ?, activation_token_expires_at = ?, updated_at = ?
		WHERE id = ?`, token, expiresAdd24h(), now, lm.ID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "update failed")
		return
	}
	// Publish EMAIL_SENT event for activation email — 包含 Kafka 优先 + SMTP 回退
	// OPT-20260807-018: activation_url 与 reset_url 同源问题——裸相对路径在邮件客户端
	// 无正确基址，点击 404；与邀请/重置邮件一致拼上 FrontendBase。
	if _, err := publishEmailSent(r.Context(), email, "账号激活 - SaaS平台", "activation", map[string]interface{}{
		"activation_url": buildUserFacingURL(fmt.Sprintf("/auth/activate/%s/", token)),
	}); err != nil {
		log.Printf("[taskAuth] resend activation: delivery failed for %s (both Kafka and SMTP): %v", email, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "激活邮件发送失败，请稍后重试"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "激活邮件已重新发送，请查收"})
}

func timeNowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.000000")
}

func expiresAdd24h() string {
	return time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02 15:04:05.000000")
}
