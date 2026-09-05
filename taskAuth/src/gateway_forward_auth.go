package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// handleGatewayForwardAuth supports APISIX forward-auth: 2xx allows the client request.
// Token resolution intentionally skips IP binding — the caller is the trusted gateway
// (authenticated via X-TaskAuth-Internal-Secret), whose internal IP differs from the
// end-user's public IP.  IP binding is enforced at the edge (direct API calls).
// unauthorizedJSON 输出带 detail + trace_id 的 401（OPT-20260806-005）：
// 此前 bare WriteHeader(401) 空 body，前端只能显示泛化「服务器错误 (401)」；
// X-Trace-Id 响应头由 tracelog middleware 注入，body 内 trace_id 供直连排障。
func unauthorizedJSON(w http.ResponseWriter, r *http.Request, detail string) {
	log.Printf("[taskAuth] event=forward_auth_deny detail=%s", detail)
	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"detail":   detail,
		"trace_id": tracelog.NormalizeTraceID(r.Header.Get(tracelog.Header)),
	})
}

func handleGatewayForwardAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	cacheKey := credentialFingerprintForForwardAuth(r)
	if ent, ok := getForwardAuthCache(cacheKey); ok {
		if !ent.isActive {
			unauthorizedJSON(w, r, "登录状态无效或已失效，请重新登录")
			return
		}
		ent.cacheStatus = "hit"
		if phoneWriteGateBlocks(r, ent) {
			writePhoneGateForbidden(w, r)
			return
		}
		writeForwardAuthHeaders(w, ent)
		return
	}

	userID, err := resolveUserIDForForwardAuth(r)
	if err != nil {
		unauthorizedJSON(w, r, "无法解析登录凭据，请重新登录")
		return
	}
	isActive, isSuperuser, isStaff, isArchived, err := loadUserAuthFlags(userID)
	if err != nil || !isActive || isArchived {
		unauthorizedJSON(w, r, "账号不可用或已归档，请联系管理员")
		return
	}
	tenantClaimsJWT := fetchAndSignTenantClaims(userID)

	// v63 单一来源: X-User-Roles 由 loadPlatformRoles（RBAC 表优先，遗留标志兜底）
	// 推导，替代过渡期的 platformRolesFromFlags。修复反向双写缺口：
	// RBAC 指派但未同步标志的用户此前在网关层缺失平台权限。
	// 查询失败时降级为标志推导，避免网关鉴权整体失败。
	platformRoles, rolesErr := loadPlatformRoles(userID)
	if rolesErr != nil {
		log.Printf("[taskAuth] event=rbac_platform_roles status=degraded user_id=%s err=%v", userID, rolesErr)
		platformRoles = platformRolesFromFlags(isSuperuser, isStaff)
	}
	tenantPerms, permErr := computePermSets(userID)
	if permErr != nil {
		log.Printf("[taskAuth] event=rbac_perm_sets status=degraded user_id=%s err=%v", userID, permErr)
		tenantPerms = nil
	}
	// 主邮箱随 forward-auth 注入 X-User-Email，供下游服务邮箱匹配（厂商门户资格判断等）。
	// 复用 resolveUserEmail（auth_login_method method_type='email'），无邮箱登录方式的
	// 用户得到确定性合成邮箱（sso-<id>@sso.invalid），不落日志、不参与匹配。
	userEmail := resolveUserEmail(userID)
	imp := impersonationFromRequest(r)
	isTester, testerErr := loadUserIsTester(userID)
	if testerErr != nil {
		log.Printf("[taskAuth] event=forward_auth_tester status=degraded user_id=%s err=%v", userID, testerErr)
		isTester = false
	}
	// OPT-20260825-005：未验证手机号业务写门禁 —— 查询已验证 phone login_method。
	// 查询失败按未验证处理（fail-closed），仅在写路径判定，读请求不受影响。
	phoneVerified, phoneErr := userHasVerifiedPhone(userID)
	if phoneErr != nil {
		log.Printf("[taskAuth] event=forward_auth_phone status=degraded user_id=%s err=%v", userID, phoneErr)
		phoneVerified = false
	}
	putForwardAuthCache(cacheKey, userID, userEmail, isActive, isSuperuser, isStaff, isTester, phoneVerified, tenantClaimsJWT, platformRoles, tenantPerms, imp.actorID, imp.sessionID)
	log.Printf("[taskAuth] event=forward_auth_cache_store user_id=%s ttl=%s perms=%d tester=%v phone_verified=%v", userID, forwardAuthCacheTTL(), len(tenantPerms), isTester, phoneVerified)

	ent := forwardAuthCacheEntry{
		userID: userID, userEmail: userEmail, isActive: isActive, isSuperuser: isSuperuser, isStaff: isStaff,
		isTester: isTester, phoneVerified: phoneVerified,
		tenantClaimsJWT: tenantClaimsJWT, platformRoles: platformRoles, tenantPerms: tenantPerms,
		impersonatorID: imp.actorID, impersonationSessionID: imp.sessionID, cacheStatus: "miss",
	}
	if phoneWriteGateBlocks(r, ent) {
		writePhoneGateForbidden(w, r)
		return
	}
	writeForwardAuthHeaders(w, ent)
}

type impersonationForwardHeaders struct {
	actorID   string
	sessionID string
}

func impersonationFromRequest(r *http.Request) impersonationForwardHeaders {
	tok := requestAuthToken(r)
	if !isImpersonationToken(tok) {
		return impersonationForwardHeaders{}
	}
	sess, err := lookupOpenImpersonationByToken(tok)
	if err != nil || sess == nil {
		return impersonationForwardHeaders{}
	}
	return impersonationForwardHeaders{
		actorID:   sess.ActorUserID,
		sessionID: strconv.FormatInt(sess.ID, 10),
	}
}

// ---- OPT-20260825-005 未验证手机号业务写门禁 ----
// 前端已硬门禁未验证手机用户；直打 /api/tenant/... 写请求仍可绕过 SPA。
// APISIX forward-auth 子请求固定 GET，但注入 X-Forwarded-Method / X-Forwarded-Uri
// 携带原始方法与路径（forward-auth.lua），据此对客户业务写路径做服务端 403 兜底。
var phoneGateWriteMethods = map[string]bool{
	"POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

// phoneGateBusinessPrefixes 为客户业务路由族（routes.yaml auth_mode: token 的业务侧），
// 与 SPA 手机号门禁覆盖的工作面板/任务详情/项目等页面对应。认证自身路由
// （/api/accounts/、/api/auth/ 等）与机器/内部路由不在此列，天然豁免。
var phoneGateBusinessPrefixes = []string{
	"/api/tenant/",
	"/api/projects/",
	"/api/cloud/",
	"/api/tasks/",
	"/api/ai-comment/",
	"/api/v1/tenants/",
	"/api/switch-workspace/",
}

// isPhoneGateExemptWritePath 放行 SPA 对未验证手机豁免页面对应的写路径
// （people/join、people/invite）：成员加入与邀请是未验证用户合法的首个业务写操作。
func isPhoneGateExemptWritePath(u string) bool {
	return strings.Contains(u, "/accounts/members/join") ||
		strings.Contains(u, "/accounts/members/invite") ||
		strings.Contains(u, "/accounts/members/validate-invite")
}

func isPhoneGateBusinessPath(u string) bool {
	for _, p := range phoneGateBusinessPrefixes {
		if strings.HasPrefix(u, p) {
			return true
		}
	}
	return false
}

func userHasVerifiedPhone(userID string) (bool, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(1) FROM auth_login_method
		WHERE object_id = ? AND method_type = 'phone' AND is_verified = 1
		  AND binding_voided_at IS NULL`, userID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (ent forwardAuthCacheEntry) isPlatformStaff() bool {
	for _, r := range ent.platformRoles {
		if r == "super_admin" || r == "employee" {
			return true
		}
	}
	return false
}

// phoneWriteGateBlocks 判定是否应 403：仅客户业务写路径 + 未验证手机 + 非员工/模拟登录。
func phoneWriteGateBlocks(r *http.Request, ent forwardAuthCacheEntry) bool {
	m := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-Forwarded-Method")))
	if m == "" {
		m = strings.ToUpper(r.Method)
	}
	if !phoneGateWriteMethods[m] {
		return false
	}
	u := strings.TrimSpace(r.Header.Get("X-Forwarded-Uri"))
	if u == "" {
		u = r.URL.Path
	}
	if u == "" || !isPhoneGateBusinessPath(u) {
		return false
	}
	if isPhoneGateExemptWritePath(u) {
		return false
	}
	// 员工/超管/模拟登录豁免（与前端 isPhoneVerifyStaffBypass / isImpersonating 对齐）
	if ent.isSuperuser || ent.isStaff || ent.isPlatformStaff() {
		return false
	}
	if ent.impersonatorID != "" {
		return false
	}
	if ent.phoneVerified {
		return false
	}
	return true
}

// writePhoneGateForbidden 输出 403 + 机器可读 error code（前端可据此引导绑定手机号）。
func writePhoneGateForbidden(w http.ResponseWriter, r *http.Request) {
	log.Printf("[taskAuth] event=forward_auth_phone_gate_deny method=%s uri=%s",
		r.Header.Get("X-Forwarded-Method"), r.Header.Get("X-Forwarded-Uri"))
	writeJSON(w, http.StatusForbidden, map[string]interface{}{
		"status":   "error",
		"code":     "phone_verification_required",
		"error":    "手机号未验证，无法执行该操作",
		"detail":   "请先完成手机号验证后再执行该操作",
		"message":  "手机号未验证，无法执行该操作",
		"trace_id": tracelog.NormalizeTraceID(r.Header.Get(tracelog.Header)),
	})
}

// writeForwardAuthHeaders 统一注入 forward-auth 响应头（v63 RBAC 角色/权限码）。
// 遗留 X-Auth-Superuser/X-Auth-Staff 已移除：平台判定统一走 X-User-Roles
// （shareLib authz.IsPlatformStaff），消费方已全部迁移。
func writeForwardAuthHeaders(w http.ResponseWriter, ent forwardAuthCacheEntry) {
	w.Header().Set("X-User-Id", ent.userID)
	if ent.userEmail != "" {
		w.Header().Set("X-User-Email", ent.userEmail)
	}
	w.Header().Set("X-Gateway-Auth-Verified", "1")
	if ent.cacheStatus != "" {
		w.Header().Set("X-Auth-Cache", ent.cacheStatus)
	}
	if ent.tenantClaimsJWT != "" {
		w.Header().Set("X-Auth-Tenant-Claims", ent.tenantClaimsJWT)
	}
	// v63 RBAC 注入
	if len(ent.platformRoles) > 0 {
		w.Header().Set("X-User-Roles", strings.Join(ent.platformRoles, ","))
	}
	if perms := serializeTenantPerms(ent.tenantPerms); perms != "" {
		w.Header().Set("X-Tenant-Perms", perms)
	}
	if ent.isTester {
		w.Header().Set("X-User-Is-Tester", "1")
	}
	// OPT-20260825-005：已验证手机号信号随 forward-auth 注入，供下游/网关侧后续判定。
	if ent.phoneVerified {
		w.Header().Set("X-User-Phone-Verified", "1")
	} else {
		w.Header().Set("X-User-Phone-Verified", "0")
	}
	if ent.impersonatorID != "" {
		w.Header().Set("X-Impersonator-Id", ent.impersonatorID)
		w.Header().Set("X-Impersonating", "1")
	}
	if ent.impersonationSessionID != "" {
		w.Header().Set("X-Impersonation-Session-Id", ent.impersonationSessionID)
	}
	w.WriteHeader(http.StatusOK)
}

// resolveUserIDForForwardAuth resolves the user ID from a forward-auth request
// WITHOUT IP binding.  The caller is the trusted gateway (authenticated via
// X-TaskAuth-Internal-Secret), whose internal IP differs from the end-user's
// public IP.  IP binding for direct API calls is enforced separately in
// resolveTokenUserIDFromRequestStrict.
func resolveUserIDForForwardAuth(r *http.Request) (string, error) {
	// 无 IP 绑定：调用方是已用 internal secret 鉴权的网关。
	// 显式 Authorization（含 Bearer JWT）优先于网页 Cookie，见 resolveUserIDFromAuthSources。
	return resolveUserIDFromAuthSources(r, userIDAuthOpts{withIP: false, allowUserIDCookie: true})
}
