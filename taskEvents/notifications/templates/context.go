package templates

import (
	"fmt"
	"strings"
)

// InvitationData is template context for invitation emails.
type InvitationData struct {
	CompanyName     string
	InvitationURL   string
	ExpirationDays  int
	Message         string
	UnsubscribeURL  string
}

type PasswordResetData struct {
	ResetURL string
}

type ActivationData struct {
	ActivationURL string
}

type VerificationCodeData struct {
	Code string
}

type EmailRegistrationInviteData struct {
	InviteURL      string
	Email          string
	UnsubscribeURL string
}

type WelcomeData struct{}

func mapContext(templateName string, raw map[string]interface{}) (any, error) {
	switch templateName {
	case "password_reset":
		resetURL := strField(raw, "reset_url")
		if resetURL == "" {
			return nil, fmt.Errorf("password_reset template missing reset_url")
		}
		return PasswordResetData{ResetURL: resetURL}, nil
	case "activation":
		activationURL := strField(raw, "activation_url")
		if activationURL == "" {
			return nil, fmt.Errorf("activation template missing activation_url")
		}
		return ActivationData{ActivationURL: activationURL}, nil
	case "verification_code":
		code := strField(raw, "code")
		if code == "" {
			return nil, fmt.Errorf("verification_code template missing code")
		}
		return VerificationCodeData{Code: code}, nil
	case "welcome":
		return WelcomeData{}, nil
	case "email_registration_invite":
		inviteURL := strField(raw, "invite_url")
		email := strField(raw, "email")
		if inviteURL == "" || email == "" {
			return nil, fmt.Errorf("email_registration_invite template missing invite_url or email")
		}
		return EmailRegistrationInviteData{
			InviteURL:      inviteURL,
			Email:          email,
			UnsubscribeURL: strField(raw, "unsubscribe_url"),
		}, nil
	case "invitation":
		companyName := strField(raw, "company_name")
		invitationURL := strField(raw, "invitation_url")
		if invitationURL == "" {
			return nil, fmt.Errorf("invitation template missing invitation_url")
		}
		// OPT-20260809-015: company_name 缺失时兜底渲染，避免旧事件缺字段卡「投递中」。
		if companyName == "" {
			companyName = "你的团队"
		}
		return InvitationData{
			CompanyName:    companyName,
			InvitationURL:  invitationURL,
			ExpirationDays: intField(raw, "expiration_days", 7),
			Message:        strField(raw, "message"),
			UnsubscribeURL: strField(raw, "unsubscribe_url"),
		}, nil
	default:
		return nil, fmt.Errorf("unknown template %q", templateName)
	}
}

func strField(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func intField(data map[string]interface{}, key string, def int) int {
	v, ok := data[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return def
	}
}
