package saas

import "strings"

const personalCompanyNameSuffix = "的公司"

// DefaultPersonalCompanyName 是空展示名时的中性默认公司名。
// 与前端 onboardingCompanyName.FALLBACK_PERSONAL_COMPANY_NAME 对齐。
const DefaultPersonalCompanyName = "我的公司"

// PersonalCompanyName 将用户展示名格式化为默认公司名「{user}的公司」。
// 空值或已是中性占位时返回 DefaultPersonalCompanyName，避免「的公司」或「我的公司的公司」。
// 已以「的公司」结尾则原样返回。
func PersonalCompanyName(displayName string) string {
	name := strings.TrimSpace(displayName)
	if name == "" || name == DefaultPersonalCompanyName {
		return DefaultPersonalCompanyName
	}
	if strings.HasSuffix(name, personalCompanyNameSuffix) {
		return name
	}
	return name + personalCompanyNameSuffix
}

// ResolvePersonalCompanyName 选择展示名再套用 PersonalCompanyName。
// 优先事件 username；若为空或仅为中性占位，则用个人昵称。
func ResolvePersonalCompanyName(username, nickname string) string {
	display := strings.TrimSpace(username)
	if display == "" || display == DefaultPersonalCompanyName {
		if n := strings.TrimSpace(nickname); n != "" && n != DefaultPersonalCompanyName {
			display = n
		} else {
			display = ""
		}
	}
	return PersonalCompanyName(display)
}
