package domain

import "strings"

const (
	LoginEntryCustomer        = "customer"
	LoginEntryAdmin           = "admin"
	LoginEntryWechat          = "wechat"
	LoginEntryAccessToken     = "access_token"
	LoginEntryRegister        = "register"
	LoginEntrySessionActivate = "session_activate"

	LoginOutcomeSuccess           = "success"
	LoginOutcomePasswordMismatch  = "password_mismatch"
	LoginOutcomeEmailUnverified   = "email_unverified"
	LoginOutcomeNotActive         = "not_active"
	LoginOutcomeAdminEntryMismatch = "admin_entry_mismatch"
	LoginOutcomeNotAdminStaff     = "not_admin_staff"

	MaxLoginHistoryUserAgentRunes = 512
)

var loginEntryLabels = map[string]string{
	LoginEntryCustomer:        "用户入口",
	LoginEntryAdmin:           "管理员入口",
	LoginEntryWechat:          "微信",
	LoginEntryAccessToken:     "访问令牌",
	LoginEntryRegister:        "注册",
	LoginEntrySessionActivate: "切换会话",
}

var loginMethodLabels = map[string]string{
	"email":    "邮箱",
	"phone":    "手机",
	"username": "用户名",
	"wechat":   "微信",
	"token":    "访问令牌",
	"session":  "会话",
}

var loginOutcomeLabels = map[string]string{
	LoginOutcomeSuccess:            "成功",
	LoginOutcomePasswordMismatch:   "密码错误",
	LoginOutcomeEmailUnverified:    "邮箱未验证",
	LoginOutcomeNotActive:          "账号未激活",
	LoginOutcomeAdminEntryMismatch: "入口不匹配",
	LoginOutcomeNotAdminStaff:      "非管理员账号",
}

// LoginOutcomeLabel is the Chinese label for a stored login outcome.
func LoginOutcomeLabel(outcome string) string {
	if label, ok := loginOutcomeLabels[outcome]; ok {
		return label
	}
	return outcome
}

// NormalizeLoginEntry returns a known entry or empty if invalid.
func NormalizeLoginEntry(entry string) string {
	e := strings.TrimSpace(entry)
	if _, ok := loginEntryLabels[e]; !ok {
		return ""
	}
	return e
}

// LoginEntryLabel is the Chinese label for a stored entry.
func LoginEntryLabel(entry string) string {
	if label, ok := loginEntryLabels[entry]; ok {
		return label
	}
	return entry
}

// LoginMethodLabel is the Chinese label for a stored method_type.
func LoginMethodLabel(method string) string {
	if label, ok := loginMethodLabels[method]; ok {
		return label
	}
	return method
}

// TruncateLoginUserAgent caps stored User-Agent length.
func TruncateLoginUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	runes := []rune(ua)
	if len(runes) <= MaxLoginHistoryUserAgentRunes {
		return ua
	}
	return string(runes[:MaxLoginHistoryUserAgentRunes])
}
