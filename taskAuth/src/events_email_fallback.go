package main

import (
	"fmt"
	"strings"
)

// buildFallbackEmailText builds a plain-text email body for the SMTP fallback / OTP sync path.
func buildFallbackEmailText(subject, templateName string, contextData map[string]interface{}) string {
	switch templateName {
	case "email_registration_invite":
		url := strVal(contextData, "invite_url")
		unsub := strVal(contextData, "unsubscribe_url")
		body := fmt.Sprintf("%s\n\n点击以下链接完成注册（7天内有效）：\n%s\n\n如未发起注册请求，请忽略此邮件。", subject, url)
		if unsub != "" {
			body += "\n\n退订邮件邀请：\n" + unsub
		}
		return body
	case "password_reset":
		resetURL := strVal(contextData, "reset_url")
		code := strVal(contextData, "code")
		if resetURL != "" {
			return fmt.Sprintf("%s\n\n点击以下链接重置密码：\n%s\n\n如未发起密码重置请求，请忽略此邮件。", subject, resetURL)
		}
		if code != "" {
			return fmt.Sprintf("%s\n\n您的验证码为：%s\n\n如未发起密码重置请求，请忽略此邮件。", subject, code)
		}
		return fmt.Sprintf("%s\n\n请使用此邮件中的信息完成密码重置。\n\n如未发起密码重置请求，请忽略此邮件。", subject)
	case "verification_code":
		code := strVal(contextData, "code")
		if code == "" {
			return subject
		}
		return fmt.Sprintf("%s\n\n您的验证码为：%s\n\n该验证码有效期为5分钟，请不要将验证码透露给他人。\n如未发起此请求，请忽略此邮件。", subject, code)
	default:
		return subject
	}
}

// buildFallbackEmailHTML builds an HTML email body for the SMTP fallback / OTP sync path.
func buildFallbackEmailHTML(subject, templateName string, contextData map[string]interface{}) string {
	switch templateName {
	case "email_registration_invite":
		url := strVal(contextData, "invite_url")
		unsub := strVal(contextData, "unsubscribe_url")
		footer := ""
		if unsub != "" {
			footer = fmt.Sprintf(`<p style="color:#888;font-size:13px"><a href="%s">退订邮件邀请</a></p>`, unsub)
		}
		return fmt.Sprintf(`<div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif">
<h2>%s</h2>
<p>您已获得SaaS平台注册邀请，点击下方按钮完成注册：</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background-color:#16a34a;color:white;text-decoration:none;border-radius:6px">接受邀请并注册</a></p>
<p style="color:#888;font-size:13px">或复制链接到浏览器：<br><a href="%s">%s</a></p>
<p style="color:#888;font-size:13px">此邀请 7 天内有效。如未发起注册请求，请忽略此邮件。</p>
%s
</div>`, subject, url, url, url, footer)
	case "password_reset":
		resetURL := strVal(contextData, "reset_url")
		code := strVal(contextData, "code")
		body := ""
		if resetURL != "" {
			body = fmt.Sprintf(`<p>点击下方按钮重置密码：</p>
<p><a href="%s" style="display:inline-block;padding:12px 24px;background-color:#16a34a;color:white;text-decoration:none;border-radius:6px">重置密码</a></p>
<p style="color:#888;font-size:13px">或复制链接到浏览器：<br><a href="%s">%s</a></p>`, resetURL, resetURL, resetURL)
		} else if code != "" {
			body = fmt.Sprintf(`<p>您的密码重置验证码为：</p>
<p style="font-size:24px;font-weight:bold;letter-spacing:4px;color:#16a34a">%s</p>`, code)
		}
		return fmt.Sprintf(`<div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif">
<h2>%s</h2>
%s
<p style="color:#888;font-size:13px">如未发起密码重置请求，请忽略此邮件。</p>
</div>`, subject, body)
	case "verification_code":
		code := strVal(contextData, "code")
		if code == "" {
			return fmt.Sprintf("<p>%s</p>", subject)
		}
		return fmt.Sprintf(`<div style="max-width:600px;margin:0 auto;font-family:Arial,sans-serif">
<h2>%s</h2>
<p>您的验证码为：</p>
<p style="font-size:24px;font-weight:bold;letter-spacing:4px;color:#16a34a">%s</p>
<p style="color:#888;font-size:13px">该验证码有效期为5分钟，请不要将验证码透露给他人。</p>
<p style="color:#888;font-size:13px">如未发起此请求，请忽略此邮件。</p>
</div>`, subject, code)
	default:
		return fmt.Sprintf("<p>%s</p>", subject)
	}
}

func strVal(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
