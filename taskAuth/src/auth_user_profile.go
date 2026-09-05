package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// clearResidualAuthCookies 清理残留的 HttpOnly userId/token cookie
// （OPT-20260807-006：DB 清库重建后前端 JS 无法删除，借 profile 401 服务端清理；
// 样式与 oidc_handlers.go end-session 的清理一致）。
func clearResidualAuthCookies(w http.ResponseWriter) {
	// 双变体（域 + host-only 旧影子）：与 oidc end-session / logout 的清理同构
	clearAuthCookies(w)
}

// handleGetUserProfile returns the authenticated user's profile data.
// Replaces Django saas-backend GET /api/accounts/users/profile/ (retired 2026-07-30).
// GET /api/accounts/users/profile/
func handleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// 不直接复用 requireAuthenticatedUser：它在内部先 writeJSON 写 401 响应体
	// （header 已 flush），之后设置 Set-Cookie 将成为 no-op（Go http 服务器只认
	// 首段 flush 前的响应头）。必须先解析（只读不写响应）→ 清理残留 cookie →
	// 再写 401。
	userID, ok := resolveUserIDFromRequest(r)
	if !ok || userID == "" {
		// OPT-20260807-006: 登录页加载会先调 profile → 401。DB 清库重建后浏览器仍持有
		// 30 天 HttpOnly userId/token cookie（前端 JS 无法删除），服务端已拒绝其认证
		// 但 cookie 残留直到自然过期；此处借 profile 401 附带 Set-Cookie Max-Age=-1
		// 清理残留（须在写 401 响应体之前设置响应头），登录页加载即清理。
		clearResidualAuthCookies(w)
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile for user %s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// handlePatchUserProfile updates the authenticated user's profile fields.
// Replaces Django saas-backend PATCH /api/accounts/users/profile/ (retired 2026-07-30).
// PATCH /api/accounts/users/profile/
func handlePatchUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	// Update personal_nickname if provided
	if nickname, ok := body["personal_nickname"]; ok {
		username := ""
		if s, ok := nickname.(string); ok {
			username = strings.TrimSpace(s)
		}
		if err := upsertUserProfile(userID, username); err != nil {
			log.Printf("[taskAuth] patch profile nickname for user %s: %v", userID, err)
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
	}

	// Update company_nicknames if provided (member_name per company)
	if nicknamesRaw, ok := body["company_nicknames"]; ok {
		if nicknames, ok := nicknamesRaw.([]interface{}); ok {
			for _, entry := range nicknames {
				if m, ok := entry.(map[string]interface{}); ok {
					cid := strField(m, "company_id")
					name := strField(m, "member_name")
					if cid != "" && name != "" {
						if err := updateMemberName(userID, cid, name); err != nil {
							log.Printf("[taskAuth] patch company_nicknames update for user %s company %s: %v", userID, cid, err)
							// Non-fatal: continue updating other companies
						}
					}
				}
			}
		}
	}

	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile after patch for user %s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// buildUserProfileJSON constructs the profile response matching the Django
// saas-backend shape that the frontend UserProfile.vue expects.
func buildUserProfileJSON(userID string) (map[string]interface{}, error) {
	// Fetch user basic info
	var isActive, isArchived bool
	err := db.QueryRow(
		`SELECT is_active, COALESCE(is_archived, 0) FROM auth_user WHERE id = ?`,
		userID,
	).Scan(&isActive, &isArchived)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	// Fetch profile (username + avatar)
	var username, avatar string
	err = db.QueryRow(
		`SELECT COALESCE(username, ''), COALESCE(avatar, '') FROM auth_user_profile WHERE user_id = ?`,
		userID,
	).Scan(&username, &avatar)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Fetch login methods (email + phone)
	rows, err := db.Query(
		`SELECT method_type, identifier, is_verified, COALESCE(phone_country_calling_code, '')
		 FROM auth_login_method
		 WHERE object_id = ? AND binding_voided_at IS NULL
		 ORDER BY method_type`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	email := ""
	var phoneIdentifier string
	var phoneCountryCode string
	for rows.Next() {
		var methodType, identifier, countryCallingCode string
		var verified bool
		if err := rows.Scan(&methodType, &identifier, &verified, &countryCallingCode); err != nil {
			return nil, err
		}
		if methodType == "email" && email == "" {
			email = identifier
		}
		if methodType == "phone" && phoneIdentifier == "" {
			phoneIdentifier = identifier
			phoneCountryCode = countryCallingCode
		}
	}

	hasPhone := phoneIdentifier != ""
	phoneMasked := maskPhoneNumber(phoneIdentifier)

	// v64: 微信绑定状态（wechat_identity app 列表；openid 别名仅存于 wechat_identity，不暴露裸值）
	wechatApps := []string{}
	wrows, err := db.Query(`
		SELECT app_key FROM wechat_identity
		WHERE user_id = ? ORDER BY app_key`, userID)
	if err == nil {
		for wrows.Next() {
			var ak string
			if wrows.Scan(&ak) == nil {
				wechatApps = append(wechatApps, ak)
			}
		}
		wrows.Close()
	}
	hasWechat := len(wechatApps) > 0
	// wechat_bind_available: 平台微信 web 应用凭据（appId/appSecret/redirectUri）是否齐全。
	// 绑定流程固定走 /api/auth/wechat/bind/?app=web，凭据缺失时服务端必然拒绝；
	// 前端据此隐藏「绑定其他微信应用」入口（缺配置时该按钮无意义）。
	wechatBindAvailable := wechatAppByKey("web") != nil

	// Build company_nicknames from taskTenantService internal API
	companyNicknames := fetchCompanyNicknames(userID)

	_ = phoneCountryCode // reserved for future display

	// Build response matching Django saas-backend shape
	// Map company_nicknames to companies shape expected by frontend router guard
	// (resolveUnauthorizedTenantRedirect expects [{id, name, ...}] not [{company_id, company_name, ...}])
	companies := buildCompaniesFromNicknames(companyNicknames)
	var currentCompany map[string]interface{}
	if len(companies) > 0 {
		currentCompany = companies[0]
	}
	return map[string]interface{}{
		"user_id":           userID,
		"personal_nickname": username,
		"email":             email,
		// OPT-20260806-066: 前端邮箱绑定面板据此渲染（无邮箱显示绑定表单）
		"has_email":             email != "",
		"has_phone":             hasPhone,
		"phone_masked":          phoneMasked,
		"has_wechat":            hasWechat,
		"wechat_apps":           wechatApps,
		"wechat_bind_available": wechatBindAvailable,
		"avatar_url":            avatar,
		"company_nicknames":     companyNicknames,
		"companies":             companies,
		"current_company":       currentCompany,
	}, nil
}

// fetchCompanyNicknames calls taskTenantService internal API to get the user's
// company memberships. Returns an empty slice on any error (graceful degradation).
//
// OPT-20260806-044: 登录/微信回调/activate-session 每次同步拉取 members 计算落点，
// 属无谓外部往返 — 按 user_id 短 TTL（5s）缓存成功结果，命中时免外部调用。
// 缓存键含 base URL，测试中不同 mock server 天然隔离。
// OPT-20260806-042: 回调路径 members 拉取对 tenant 服务瞬态失败（重启窗口/网络
// 抖动）无重试，有公司用户会误落 /onboarding/ — 首次失败后 500ms 重试一次，
// 两次均失败才返回空（落点兜底，且不缓存失败结果）。
func fetchCompanyNicknames(userID string) []interface{} {
	base := strings.TrimRight(cfg.TenantServiceURL, "/")
	if base == "" {
		log.Printf("[taskAuth] fetchCompanyNicknames user=%s: cfg.TenantServiceURL is empty", userID)
		return []interface{}{}
	}
	key := base + "|" + userID
	tenantMembersCacheMu.Lock()
	if ent, ok := tenantMembersCache[key]; ok && time.Since(ent.fetchedAt) < tenantMembersCacheTTL {
		tenantMembersCacheMu.Unlock()
		return ent.result
	}
	tenantMembersCacheMu.Unlock()

	log.Printf("[taskAuth] fetchCompanyNicknames user=%s base=%s", userID, base)
	result, err := fetchTenantMembersOnce(base, userID)
	if err != nil {
		// 瞬态失败重试一次（OPT-20260806-042）
		log.Printf("[taskAuth] fetchCompanyNicknames call for user %s: %v (retrying once)", userID, err)
		time.Sleep(tenantMembersRetryDelay)
		result, err = fetchTenantMembersOnce(base, userID)
		if err != nil {
			log.Printf("[taskAuth] fetchCompanyNicknames retry for user %s: %v (fail-closed empty)", userID, err)
			return []interface{}{}
		}
	}
	// 仅缓存成功结果（不缓存失败，避免落点被冻结）
	tenantMembersCacheMu.Lock()
	tenantMembersCache[key] = tenantMembersCacheEntry{result: result, fetchedAt: time.Now()}
	tenantMembersCacheMu.Unlock()
	return result
}

// ── tenant members 短 TTL 缓存 + 瞬态失败重试（OPT-20260806-044/042）──
const tenantMembersCacheTTL = 5 * time.Second

// tenantMembersRetryDelay 为瞬态失败重试间隔；测试可覆盖为极小值。
var tenantMembersRetryDelay = 500 * time.Millisecond

type tenantMembersCacheEntry struct {
	result    []interface{}
	fetchedAt time.Time
}

var (
	tenantMembersCacheMu sync.Mutex
	tenantMembersCache   = map[string]tenantMembersCacheEntry{}
)

func fetchTenantMembersOnce(base, userID string) ([]interface{}, error) {
	url := fmt.Sprintf("%s/api/internal/tenant/members?user_id=%s", base, userID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var members []map[string]interface{}
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, err
	}
	result := make([]interface{}, 0, len(members))
	for _, m := range members {
		result = append(result, map[string]interface{}{
			"company_id":        m["company_id"],
			"company_name":      m["company_name"],
			"member_name":       m["member_name"],
			"member_avatar_url": m["member_avatar_url"],
			"is_admin":          m["is_admin"],
			"is_creator":        m["is_creator"],
			"is_active":         m["is_active"],
			"workspace_id":      m["workspace_id"],
		})
	}
	return result, nil
}

// updateMemberName calls taskTenantService to update a member's display name.
// Uses the existing PATCH /api/internal/tenant/members/update endpoint.
func updateMemberName(userID, companyID, memberName string) error {
	base := strings.TrimRight(cfg.TenantServiceURL, "/")
	if base == "" {
		return fmt.Errorf("tenant service URL not configured")
	}
	body := map[string]string{
		"user_id":     userID,
		"company_id":  companyID,
		"member_name": memberName,
	}
	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/internal/tenant/members/update", base)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// maskPhoneNumber masks a phone number for display (e.g. "138****1234").
// Returns empty string for empty input.
func maskPhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	runes := []rune(phone)
	if len(runes) <= 4 {
		return phone
	}
	// Show first 3 and last 4 digits
	prefix := string(runes[:3])
	suffix := string(runes[len(runes)-4:])
	return prefix + "****" + suffix
}

// buildCompaniesFromNicknames converts company_nicknames [{company_id, company_name, ...}]
// to the companies shape [{id, name, ...}] expected by the frontend router guard.
func buildCompaniesFromNicknames(nicknames []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(nicknames))
	for _, n := range nicknames {
		if m, ok := n.(map[string]interface{}); ok {
			entry := map[string]interface{}{
				"id":                m["company_id"],
				"name":              m["company_name"],
				"member_name":       m["member_name"],
				"member_avatar_url": m["member_avatar_url"],
				"is_admin":          m["is_admin"],
				"is_creator":        m["is_creator"],
				"is_active":         m["is_active"],
			}
			result = append(result, entry)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Avatar endpoints
// ---------------------------------------------------------------------------

// avatarUploadDisabled — 临时安全开关：图片上传漏洞修复前，禁用所有头像上传路径
// （全局头像 POST /profile/avatar/、公司头像 POST /profile/company-avatar/、
// 以及 copy-personal-to-company 的全局头像复制）。
// 删除头像（DELETE）不受影响。
// OPT-20260806-046: 安全加固已完成（sniffAndValidateAvatarImage — magic bytes +
// 真实解码 + 尺寸/压缩比限制），开关已还原为 false（上传恢复）。若再发现漏洞，
// 可改回 true 快速熔断。
var avatarUploadDisabled = false

const maxAvatarSize = 2 * 1024 * 1024 // 2 MB

var allowedAvatarTypes = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// handleUploadAvatar handles multipart avatar file upload.
// POST /api/accounts/users/profile/avatar/
func handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	// OPT-20260806-046: 安全开关（默认关闭 = 上传开启）。若再发现图片上传漏洞，
	// 将 avatarUploadDisabled 改回 true 即可立即熔断所有上传路径（含公司头像）。
	if avatarUploadDisabled {
		writeError(w, r, http.StatusForbidden, "头像上传已暂时禁用（安全维护中），请稍后再试")
		return
	}

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+1024) // small headroom for multipart overhead
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		writeError(w, r, http.StatusBadRequest, "文件过大或格式无效，最大支持 2MB")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "未找到上传的文件")
		return
	}
	defer file.Close()

	// Read file into buffer (for magic-byte sniffing + decode validation)
	buf := &bytes.Buffer{}
	written, err := io.Copy(buf, file)
	if err != nil {
		log.Printf("[taskAuth] avatar read error for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "读取文件失败")
		return
	}
	if written > maxAvatarSize {
		writeError(w, r, http.StatusBadRequest, "文件过大，最大支持 2MB")
		return
	}

	// OPT-20260806-046: 安全加固 — magic bytes 嗅探 + 真实解码验证 + 尺寸/压缩比限制
	// （替代此前仅信 Content-Type 头；拒绝 SVG/HTML 伪装与解压炸弹）
	mimeType, _, err := sniffAndValidateAvatarImage(header.Header.Get("Content-Type"), buf.Bytes())
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Encode as base64 data URI
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(buf.Bytes()))

	// Upsert avatar into auth_user_profile
	if err := upsertUserAvatar(userID, dataURI); err != nil {
		log.Printf("[taskAuth] avatar save error for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "保存头像失败")
		return
	}

	// Return updated profile
	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile after avatar upload for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleDeleteAvatar removes the user's avatar.
// DELETE /api/accounts/users/profile/avatar/
func handleDeleteAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	if err := upsertUserAvatar(userID, ""); err != nil {
		log.Printf("[taskAuth] avatar delete error for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "移除头像失败")
		return
	}

	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile after avatar delete for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// upsertUserAvatar updates the avatar field, creating the profile row if it doesn't exist.
func upsertUserAvatar(userID, avatar string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`
		INSERT INTO auth_user_profile (user_id, username, avatar, updated_at)
		VALUES (?, '', ?, ?)
		ON DUPLICATE KEY UPDATE avatar = VALUES(avatar), updated_at = VALUES(updated_at)`,
		userID, avatar, now)
	return err
}
