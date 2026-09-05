package domain

// MaxSharedPhoneBindings 是同一规范化手机号上允许并存的活跃绑定上限。
// 仅超级用户 / 员工 / 测试账号可共享；普通客户仍一号一户。
const MaxSharedPhoneBindings = 5

// PhoneShareDecision 是绑定占用策略的结论。
type PhoneShareDecision string

const (
	PhoneShareAllow PhoneShareDecision = "allow"
	PhoneShareTaken PhoneShareDecision = "taken"
	PhoneShareLimit PhoneShareDecision = "limit"
)

// EvaluatePhoneShareBind 根据绑定者是否特权、其他活跃占用者是否均为特权、占用人数做裁决。
// otherHoldersPrivileged 不含绑定者自己；长度为其他活跃占用数。
func EvaluatePhoneShareBind(actorPrivileged bool, otherHoldersPrivileged []bool) PhoneShareDecision {
	if len(otherHoldersPrivileged) == 0 {
		return PhoneShareAllow
	}
	if !actorPrivileged {
		return PhoneShareTaken
	}
	for _, p := range otherHoldersPrivileged {
		if !p {
			return PhoneShareTaken
		}
	}
	if len(otherHoldersPrivileged) >= MaxSharedPhoneBindings {
		return PhoneShareLimit
	}
	return PhoneShareAllow
}

// IsPhoneLoginAmbiguous 表示同一手机号上密码命中了多个账号，不能 LIMIT 1 登录。
func IsPhoneLoginAmbiguous(passwordMatchCount int) bool {
	return passwordMatchCount >= 2
}
