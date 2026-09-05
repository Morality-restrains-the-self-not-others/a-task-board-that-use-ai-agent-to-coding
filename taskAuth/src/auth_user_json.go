package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// buildLoginUserJSON constructs the user object returned from login/register/activate-session.
// Replaces the Django enrich-login serialization — no Django dependency.
func buildLoginUserJSON(userID string, token string) map[string]interface{} {
	detail, err := buildUserDetailJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] buildLoginUserJSON user=%s: %v", userID, err)
		// Fallback: minimal user JSON
		return map[string]interface{}{
			"id":              userID,
			"is_superuser":    false,
			"is_tenant":       false,
			"is_tester":       false,
			"is_active":       true,
			"companies":       []interface{}{},
			"current_company": nil,
			"login_methods":   []interface{}{},
			"username":        userID,
			"email":           "",
		}
	}
	// buildUserDetailJSON 已从 taskTenantService 拉取 companies/current_company/
	// current_workspace（此前曾在此处无条件覆盖为空，导致租户登录后公司列表丢失 —
	// 见 OPT-20260805 回归）。此处直接返回 detail 保留真实数据。

	return detail
}

// buildLoginResponse assembles the full login/register response JSON.
// redirect_url is determined by user role:
//   - superuser → /system-admin/
//   - regular user with company → /tenant/{company_id}/work-panel/
//   - regular user without company → /onboarding/
func buildLoginResponse(userID string, token string) map[string]interface{} {
	userJSON := buildLoginUserJSON(userID, token)
	redirectURL := resolveLoginRedirect(userJSON)
	return map[string]interface{}{
		"token":        token,
		"user":         userJSON,
		"redirect_url": redirectURL,
	}
}

// resolveLoginRedirect determines the post-login destination based on user role.
func resolveLoginRedirect(userJSON map[string]interface{}) string {
	// 平台角色（super_admin/employee）→ 系统管理。平台员工无公司，禁止落入
	// /onboarding/ 引导页（此前仅 superuser 走 /system-admin/，员工被误导）。
	if roles, ok := userJSON["platform_roles"].([]string); ok {
		for _, r := range roles {
			if r == "super_admin" || r == "employee" {
				return "/system-admin/"
			}
		}
	}
	// 过渡兜底：遗留 is_superuser 标志（platform_roles 缺失时）
	if isSuper, ok := userJSON["is_superuser"].(bool); ok && isSuper {
		return "/system-admin/"
	}

	// Regular user with company → work panel
	// companies 字段类型不统一：buildUserDetailJSON / profile 产出
	// []map[string]interface{}（auth_users.go / auth_user_profile.go），
	// 而 buildLoginUserJSON 的 fallback 产出 []interface{}。若只断言一种类型，
	// 另一种必落空 → 有公司用户登录/微信回调仍被导向 /onboarding/
	// （「第二次扫码登录依旧跳到公司名称设置页」的确定性根因）。
	if cid := firstCompanyID(userJSON); cid != "" && cid != "0" {
		return "/tenant/" + cid + "/work-panel/"
	}

	// Fallback: user without company → onboarding
	return "/onboarding/"
}

// impersonationLandingRedirect is the post-impersonation destination.
// Platform-admin login lands on /system-admin/; impersonation must enter the
// target user's frontend instead of keeping the operator on the console.
func impersonationLandingRedirect(userJSON map[string]interface{}) string {
	url := resolveLoginRedirect(userJSON)
	if strings.HasPrefix(url, "/system-admin") {
		if cid := firstCompanyID(userJSON); cid != "" && cid != "0" {
			return "/tenant/" + cid + "/work-panel/"
		}
		return "/onboarding/"
	}
	return url
}

// firstCompanyID 从 userJSON["companies"] 提取首个公司 id，兼容两种实际形状：
// []map[string]interface{}（buildUserDetailJSON/buildUserDetailJSON 产物）与
// []interface{}（遗留 fallback 产物）。取不到返回 ""。
func firstCompanyID(userJSON map[string]interface{}) string {
	switch cs := userJSON["companies"].(type) {
	case []map[string]interface{}:
		if len(cs) > 0 {
			if id, ok := cs[0]["id"]; ok {
				return fmt.Sprintf("%v", id)
			}
		}
	case []interface{}:
		for _, item := range cs {
			if c, ok := item.(map[string]interface{}); ok {
				if id, ok := c["id"]; ok {
					return fmt.Sprintf("%v", id)
				}
			}
		}
	}
	return ""
}

// buildActivateSessionResponse builds the activate-session response.
func buildActivateSessionResponse(userID string, token string) map[string]interface{} {
	return buildLoginResponse(userID, token)
}

// resolveTokenUserIDForAuth is a convenience wrapper that returns empty string on error.
func resolveTokenUserIDForAuth(tokenKey string) string {
	userID, err := resolveTokenUserID(tokenKey)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("[taskAuth] resolveTokenUserIDForAuth: %v", err)
		}
		return ""
	}
	return userID
}
