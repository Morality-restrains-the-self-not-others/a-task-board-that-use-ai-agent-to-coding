package notifications

import (
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
	"taskEvents/notifications/templates"
)

func (d *LocalDelivery) handleInvitationCreated(data map[string]interface{}) (domain.DispatchOutcome, error) {
	companyName := strField(data, "company_name")
	companyID := strField(data, "company_id")
	email := strField(data, "email")
	phone := strField(data, "phone")
	invitationURL := strField(data, "invitation_url")
	message := strField(data, "message")
	inviteMethod := strField(data, "invite_method")
	if inviteMethod == "" {
		inviteMethod = "email"
	}
	expirationDays := intField(data, "expiration_days", 7)

	// OPT-20260809-015 容错：company_name 缺失时以 company_id 兜底渲染公司名。
	// 旧发布方 / 存量事件可能只带 company_id（tenant 874176608758427648 死信根因），
	// 不得因缺公司名而永久卡「投递中」。
	if companyName == "" {
		if companyID != "" {
			companyName = companyID
		} else {
			companyName = "你的团队"
		}
	}

	switch inviteMethod {
	case "link":
		if invitationURL == "" {
			return permanentInviteFail(data, "invitation link missing fields")
		}
		if email == "" {
			log.Printf("[notifications] INVITATION_CREATED link-only skip")
			return domain.DispatchSuccess, nil
		}
	case "email":
		if email == "" || invitationURL == "" {
			return permanentInviteFail(data, "invitation email missing fields")
		}
	case "phone":
		if phone == "" || invitationURL == "" {
			return permanentInviteFail(data, "invitation phone missing fields")
		}
	default:
		return permanentInviteFail(data, "unknown invite_method %q", inviteMethod)
	}

	if inviteMethod == "email" {
		skippedPayload := false
		switch v := data["email_skipped"].(type) {
		case bool:
			skippedPayload = v
		case string:
			skippedPayload = strings.EqualFold(v, "true")
		}
		unsub, unsubURL := lookupInviteUnsubscription(email)
		if skippedPayload || unsub {
			log.Printf("[notifications] INVITATION_CREATED skipped unsubscribed")
			callbackTenantInvitationDeliveryStatus(data, "skipped_unsubscribed", "")
			return domain.DispatchSuccess, nil
		}
		if unsubURL == "" {
			unsubURL = strField(data, "unsubscribe_url")
		}
		data["_unsubscribe_url"] = unsubURL
	}

	if inviteMethod == "phone" {
		content := fmt.Sprintf("您被邀请加入%s，请通过以下链接加入（%d天内有效）：%s", companyName, expirationDays, invitationURL)
		if message != "" {
			content = content + "。附言：" + message
		}
		if err := d.SMS.SendNotification(phone, content, nil); err != nil {
			callbackTenantInvitationDeliveryStatus(data, "failed", err.Error())
			return domain.DispatchRetryable, err
		}
		callbackTenantInvitationDeliveryStatus(data, "delivered", "")
		return domain.DispatchSuccess, nil
	}

	textBody, htmlBody, err := d.Templates.RenderInvitation(templates.InvitationData{
		CompanyName:    companyName,
		InvitationURL:  invitationURL,
		ExpirationDays: expirationDays,
		Message:        message,
		UnsubscribeURL: strField(data, "_unsubscribe_url"),
	})
	if err != nil {
		return permanentInviteFail(data, "render invitation template: %v", err)
	}
	subject := fmt.Sprintf("邀请加入 %s - SaaS平台", companyName)
	unsubURL := strField(data, "_unsubscribe_url")
	if unsubURL == "" {
		unsubURL = strField(data, "unsubscribe_url")
	}
	if err := d.SMTP.SendWithHeaders(d.From, []string{email}, subject, textBody, htmlBody, listUnsubscribeHeaders(unsubURL)); err != nil {
		callbackTenantInvitationDeliveryStatus(data, "failed", err.Error())
		return smtpDispatchOutcome(err)
	}
	log.Printf("[notifications] INVITATION_CREATED email sent to %s", email)
	callbackTenantInvitationDeliveryStatus(data, "delivered", "")
	return domain.DispatchSuccess, nil
}

// permanentInviteFail reports a permanent INVITATION_CREATED failure to
// taskTenantService (delivery_status → failed) before returning DispatchPermanent.
// OPT-20260809-015: 永久失败不回调会让邀请永远显示「投递中」（真实死信根因），
// 前端无法区分「投递中」与「已死信」。
func permanentInviteFail(data map[string]interface{}, format string, args ...interface{}) (domain.DispatchOutcome, error) {
	err := fmt.Errorf(format, args...)
	callbackTenantInvitationDeliveryStatus(data, "failed", err.Error())
	return domain.DispatchPermanent, err
}
