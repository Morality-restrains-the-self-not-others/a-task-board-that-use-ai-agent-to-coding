package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"taskEvents/domain"
	"taskEvents/notifications/cfg"
	"taskEvents/notifications/sms"
	"taskEvents/notifications/smtp"
	"taskEvents/notifications/templates"
)

// LocalDelivery implements domain.DomainCommandPort for in-process SMTP/SMS.
type LocalDelivery struct {
	SMTP      *smtp.Sender
	SMS       *sms.AliyunSender
	Templates *templates.Engine
	From      string
}

func NewLocalDelivery(settings cfg.Settings) (*LocalDelivery, error) {
	engine, err := templates.NewEngine()
	if err != nil {
		return nil, err
	}
	return &LocalDelivery{
		SMTP:      smtp.NewSender(settings.Email),
		SMS:       sms.NewAliyunSender(settings.SMS.Aliyun),
		Templates: engine,
		From:      settings.Email.DefaultFrom,
	}, nil
}

func (d *LocalDelivery) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	switch cmd.EventType {
	case "EMAIL_SENT":
		return d.handleEmailSent(data)
	case "INVITATION_CREATED":
		return d.handleInvitationCreated(data)
	case "USER_ACTIVATED":
		return d.handleUserActivated(data)
	case "WECHAT_IDENTITY_CONFLICT":
		return d.handleWechatIdentityConflict(data)
	default:
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
}

// handleWechatIdentityConflict 微信身份双射异常/绑定冲突审计告警（v64）。
// 输出结构化 JSON 审计行（Loki 可检索）+ ERROR 级告警日志。
func (d *LocalDelivery) handleWechatIdentityConflict(data map[string]interface{}) (domain.DispatchOutcome, error) {
	ownerUserID := strField(data, "owner_user_id")
	appKey := strField(data, "app_key")
	openID := strField(data, "openid")
	unionID := strField(data, "unionid")
	ownerUnionID := strField(data, "owner_unionid")
	if ownerUserID == "" || openID == "" {
		return domain.DispatchPermanent, fmt.Errorf("WECHAT_IDENTITY_CONFLICT missing owner_user_id/openid")
	}
	// openid/unionid 为半公开标识，审计日志保留以便人工对账
	audit := map[string]interface{}{
		"event":         "wechat_identity_conflict",
		"owner_user_id": ownerUserID,
		"app_key":       appKey,
		"openid":        openID,
		"unionid":       unionID,
		"owner_unionid": ownerUnionID,
	}
	raw, _ := json.Marshal(audit)
	log.Printf("[audit] %s", string(raw))
	log.Printf("[alert] WECHAT_IDENTITY_CONFLICT owner_user_id=%s app_key=%s openid=%s unionid=%s owner_unionid=%s — 需人工核查身份归属",
		ownerUserID, appKey, openID, unionID, ownerUnionID)
	return domain.DispatchSuccess, nil
}

func (d *LocalDelivery) handleEmailSent(data map[string]interface{}) (domain.DispatchOutcome, error) {
	subject := strField(data, "subject")
	recipients := strSliceField(data, "recipient_list")
	if subject == "" || len(recipients) == 0 {
		log.Printf("[notifications] EMAIL_SENT missing subject/recipients: %+v", data)
		return domain.DispatchPermanent, fmt.Errorf("missing subject or recipients")
	}
	if boolField(data, "pre_delivered") {
		log.Printf("[notifications] EMAIL_SENT skip smtp pre_delivered recipients=%v", recipients)
		callbackEmailDeliveryStatus(data, "delivered", "")
		return domain.DispatchSuccess, nil
	}
	from := strField(data, "from_email")
	if from == "" {
		from = d.From
	}
	textBody := strField(data, "message")
	htmlBody := strField(data, "html_message")
	templateName := strField(data, "template_name")
	if templateName != "" && (textBody == "" || htmlBody == "") {
		renderedText, renderedHTML, err := d.Templates.Render(templateName, eventContextMap(data))
		if err != nil {
			log.Printf("[notifications] EMAIL_SENT template render error template=%s: %v", templateName, err)
			return domain.DispatchPermanent, err
		}
		if textBody == "" {
			textBody = renderedText
		}
		if htmlBody == "" {
			htmlBody = renderedHTML
		}
	}
	if textBody == "" && htmlBody == "" {
		if templateName == "" {
			log.Printf("[notifications] EMAIL_SENT empty body and no template_name: %+v", data)
			return domain.DispatchPermanent, fmt.Errorf("EMAIL_SENT empty body and no template_name")
		}
		log.Printf("[notifications] EMAIL_SENT rendered empty body template=%s", templateName)
		return domain.DispatchPermanent, fmt.Errorf("EMAIL_SENT rendered empty body")
	}
	unsubHeaders := map[string]string(nil)
	if templateName == "email_registration_invite" || templateName == "invitation" {
		if len(recipients) > 0 {
			unsub, link := lookupInviteUnsubscription(recipients[0])
			if unsub {
				log.Printf("[notifications] EMAIL_SENT skipped unsubscribed recipient")
				callbackEmailDeliveryStatus(data, "skipped_unsubscribed", "")
				return domain.DispatchSuccess, nil
			}
			if link == "" {
				link = strField(eventContextMap(data), "unsubscribe_url")
			}
			unsubHeaders = listUnsubscribeHeaders(link)
		}
	}
	if err := d.SMTP.SendWithHeaders(from, recipients, subject, textBody, htmlBody, unsubHeaders); err != nil {
		log.Printf("[notifications] EMAIL_SENT smtp error: %v", err)
		return smtpDispatchOutcome(err)
	}
	log.Printf("[notifications] EMAIL_SENT delivered to %v", recipients)
	// Callback to taskAuth for registration invite delivery confirmation
	callbackEmailDeliveryStatus(data, "delivered", "")
	return domain.DispatchSuccess, nil
}

func (d *LocalDelivery) handleUserActivated(data map[string]interface{}) (domain.DispatchOutcome, error) {
	userID := data["user_id"]
	if userID == nil {
		return domain.DispatchPermanent, fmt.Errorf("USER_ACTIVATED missing user_id")
	}
	methodType := strField(data, "activated_method_type")
	identifier := strField(data, "activated_identifier")
	username := strField(data, "username")

	if methodType == "phone" && identifier != "" {
		display := username
		if display == "" {
			display = identifier
		}
		content := fmt.Sprintf("欢迎加入SaaS平台，%s！您的账号已激活成功。", display)
		if err := d.SMS.SendNotification(identifier, content, map[string]string{"content": content}); err != nil {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchSuccess, nil
	}

	if methodType == "email" && identifier != "" {
		display := username
		if display == "" {
			display = strings.Split(identifier, "@")[0]
		}
		subject := "欢迎使用SaaS平台"
		textBody := fmt.Sprintf("欢迎 %s！\n\n您的账号已激活成功，欢迎使用SaaS平台。\n\n如有任何问题，请联系我们。", display)
		htmlBody := fmt.Sprintf("<h1>欢迎 %s！</h1><p>您的账号已激活成功，欢迎使用SaaS平台。</p><p>如有任何问题，请联系我们。</p>", display)
		if err := d.SMTP.Send(d.From, []string{identifier}, subject, textBody, htmlBody); err != nil {
			return smtpDispatchOutcome(err)
		}
		log.Printf("[notifications] USER_ACTIVATED welcome email to %s", identifier)
		return domain.DispatchSuccess, nil
	}

	log.Printf("[notifications] USER_ACTIVATED no contact for user_id=%v", userID)
	return domain.DispatchSuccess, nil
}

func smtpDispatchOutcome(err error) (domain.DispatchOutcome, error) {
	if smtp.IsPermanent(err) {
		return domain.DispatchPermanent, err
	}
	return domain.DispatchRetryable, err
}

func strField(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func boolField(data map[string]interface{}, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		s := strings.TrimSpace(strings.ToLower(b))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func eventContextMap(data map[string]interface{}) map[string]interface{} {
	raw, ok := data["context"].(map[string]interface{})
	if ok && raw != nil {
		return raw
	}
	return map[string]interface{}{}
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

func strSliceField(data map[string]interface{}, key string) []string {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return arr
	case string:
		s := strings.TrimSpace(arr)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

// emailTokenFromContext extracts a per-email business token from the event context:
// the last path segment of the *_url link fields, or the verification code.
// It distinguishes distinct emails of the same template to the same recipient
// (e.g. two password-reset clicks with different tokens), while staying stable
// across Kafka redeliveries of the same event (same token).
func emailTokenFromContext(data map[string]interface{}) string {
	ctx, ok := data["context"].(map[string]interface{})
	if !ok || ctx == nil {
		return ""
	}
	for _, key := range []string{"reset_url", "activation_url", "invite_url", "invitation_url"} {
		v, ok := ctx[key].(string)
		if !ok || v == "" {
			continue
		}
		if i := strings.LastIndex(v, "/"); i >= 0 && i < len(v)-1 {
			return v[i+1:]
		}
		return v
	}
	if code, ok := ctx["code"].(string); ok && code != "" {
		return code
	}
	return ""
}

// IdempotencyKeyForEnvelope includes recipient/channel in the dedupe key.
func IdempotencyKeyForEnvelope(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	recipient := ""
	switch env.EventType {
	case "EMAIL_SENT":
		if rs := strSliceField(data, "recipient_list"); len(rs) > 0 {
			recipient = rs[0]
		}
		// OPT-20260807-018 修复: 发布方以收件人邮箱作为 Kafka 消息 key，此前幂等键
		// = recipient + ":" + 消息key，同一收件人的所有邮件（验证码/重置/激活/欢迎）
		// 共用一个键 → 首封 Mark 后，后续邮件全部被静默去重丢弃（线上事件：绑定验证码
		// 邮件送达后，密码重置邮件两次尝试均无投递日志）。键改为
		// template_name + 业务 token + recipient：同一事件重投仍去重，不同邮件互不干扰。
		business := strField(data, "template_name")
		if tok := emailTokenFromContext(data); tok != "" {
			business += ":" + tok
		}
		if recipient != "" {
			business += ":" + recipient
		}
		return domain.IdempotencyKeyForEvent(env.EventType, business)
	case "INVITATION_CREATED":
		if m := strField(data, "invite_method"); m == "phone" {
			recipient = strField(data, "phone")
		} else {
			recipient = strField(data, "email")
		}
	case "USER_ACTIVATED":
		recipient = strField(data, "activated_identifier")
	}
	suffix := env.Key
	if recipient != "" {
		suffix = recipient + ":" + suffix
	}
	return domain.IdempotencyKeyForEvent(env.EventType, suffix)
}

// --- Delivery status callback to taskAuth ---

// taskAuthInternalURL returns the taskAuth internal API base URL.
func taskAuthInternalURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8003"
}

// callbackEmailDeliveryStatus notifies taskAuth about email delivery outcome
// for email_registration_invite template events. This closes the loop on
// async Kafka delivery — the consumer here has the actual SMTP result.
func callbackEmailDeliveryStatus(data map[string]interface{}, status, errMsg string) {
	templateName := strField(data, "template_name")
	// Only callback for registration invite emails
	if templateName != "email_registration_invite" {
		return
	}

	// Extract email and invitation_id from the event data
	email := ""
	if recipients := strSliceField(data, "recipient_list"); len(recipients) > 0 {
		email = recipients[0]
	}
	if email == "" {
		log.Printf("[notifications] callback: skip - no email in EMAIL_SENT event")
		return
	}

	invitationID := ""
	ctx := eventContextMap(data)
	if ctx != nil {
		if id, ok := ctx["invitation_id"].(string); ok {
			invitationID = id
		}
	}

	// Build callback payload
	payload := map[string]interface{}{
		"email":         email,
		"status":        status,
		"invitation_id": invitationID,
		"error_message": errMsg,
	}
	body, _ := json.Marshal(payload)

	url := taskAuthInternalURL() + "/api/internal/email-delivery-callback/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[notifications] callback: build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[notifications] callback: POST %s failed: %v", url, err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("[notifications] callback: taskAuth returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		return
	}
	log.Printf("[notifications] callback: email=%s status=%s → taskAuth OK", email, status)
}

// tenantInvitationInternalURL returns the taskTenantService internal API base URL.
func tenantInvitationInternalURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_TENANT_INTERNAL_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8020"
}

// callbackTenantInvitationDeliveryStatus notifies taskTenantService about the
// delivery outcome of a tenant invitation (email/SMS). Closes the loop on async
// Kafka delivery — this consumer has the actual SMTP/SMS result, which
// taskTenantService persists on tenant_invitation.delivery_status for the
// people/manage pending-invitations list (投递状态列).
func callbackTenantInvitationDeliveryStatus(data map[string]interface{}, status, errMsg string) {
	invitationID := strField(data, "invitation_id")
	if invitationID == "" {
		// 非租户邀请事件（如 taskAuth 注册邀请）无 invitation_id — 跳过
		return
	}
	payload := map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}
	body, _ := json.Marshal(payload)

	url := tenantInvitationInternalURL() + "/api/internal/tenant/invitations/delivery-callback/" + invitationID + "/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[notifications] tenant-invite callback: build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[notifications] tenant-invite callback: POST %s failed: %v", url, err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("[notifications] tenant-invite callback: taskTenantService returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		return
	}
	log.Printf("[notifications] tenant-invite callback: invitation_id=%s status=%s → taskTenantService OK", invitationID, status)
}
