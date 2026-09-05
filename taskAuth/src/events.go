package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

// emitTrace writes a structured JSON log line with the request trace_id
// (OPT-20260901-016) so Loki {job="task-auth"} | json | trace_id="<id>" reaches
// EMAIL_SENT / kafka publish lines, not just http_request.
func emitTrace(ctx context.Context, level, msg string, fields map[string]string) {
	tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), level, msg, "events", fields)
}

// EMAIL_SENT → topic email-sent（与 Django KAFKA_TOPICS / taskEvents EventTopic 对齐）
var eventTopicMap = map[string]string{
	"EMAIL_SENT":                         "email-sent",
	"EMAIL_UNSUBSCRIBED":                 "email-unsubscribed",
	"EMAIL_RESUBSCRIBED":                 "email-resubscribed",
	"USER_CREATED":                       "user-created",
	"USER_ACTIVATED":                     "user-activated",
	"USER_LOGGED_IN":                     "user-logged-in",
	"REGISTRATION_INVITE_POLICY_UPDATED": "registration-invite-policy-updated",
	"REGISTRATION_INVITE_CODE_ISSUED":    "registration-invite-code-issued",
	"REGISTRATION_INVITE_CODE_REDEEMED":  "registration-invite-code-redeemed",
	"KYC_TIER_CHANGED":                   "kyc-tier-changed",
	"KYC_STATUS_CHANGED":                 "kyc-status-changed",
	"AML_SCREENING_RECORDED":             "aml-screening-recorded",
	// v64 wechat identity events
	"WECHAT_IDENTITY_LINKED":                  "wechat-identity-linked",
	"WECHAT_IDENTITY_CONFLICT":                "wechat-identity-conflict",
	"WECHAT_BOUND":                            "wechat-bound",
	"WECHAT_UNBOUND":                          "wechat-unbound",
	"WECHAT_MP_SUBSCRIBED":                    "wechat-mp-subscribed",
	"USER_ACCOUNT_DELETION_REQUESTED":         "user-account-deletion-requested",
	"USER_ACCOUNT_DELETION_CANCELLED":         "user-account-deletion-cancelled",
	"USER_ACCOUNT_DELETION_EXECUTION_STARTED": "user-account-deletion-execution-started",
	"USER_ACCOUNT_DELETION_COMPLETED":         "user-account-deletion-completed",
	"USER_ACCOUNT_DELETION_BLOCKED":           "user-account-deletion-blocked",
	// v72 / ADR-0003 RBAC authz cache invalidation (+ audit)
	"RoleChanged":                      "role-changed",
	"PlatformRoleChanged":              "platform-role-changed",
	"RoleResourceGroupsChanged":        "role-resource-groups-changed",
	"UserImpersonationStarted":         "user-impersonation-started",
	"UserImpersonationStopped":         "user-impersonation-stopped",
	"UserInboxMessageCreated":          "user-inbox-message-created",
	"UserTesterFlagChanged":            "user-tester-flag-changed",
	"TenantGitLabOidcSsoEnabled":       "tenant-gitlab-oidc-sso-enabled",
	"TenantGitLabOidcSsoSecretRotated": "tenant-gitlab-oidc-sso-secret-rotated",
	"TenantGitLabOidcSsoDisabled":      "tenant-gitlab-oidc-sso-disabled",
}

// --- Shared Kafka Writer (long-lived, auto-reconnect) ---

var (
	kafkaWriterMu sync.Mutex
	kafkaWriter   *kafka.Writer
)

func kafkaBootstrapServers() string {
	if v := strings.TrimSpace(os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_KAFKA_BOOTSTRAP_SERVERS")); v != "" {
		return v
	}
	return strings.TrimSpace(cfg.KafkaBootstrapServers)
}

// initKafkaWriter creates the shared kafka.Writer. Safe to call multiple times.
func initKafkaWriter() {
	kafkaWriterMu.Lock()
	defer kafkaWriterMu.Unlock()
	if kafkaWriter != nil {
		return
	}
	bootstrap := kafkaBootstrapServers()
	if bootstrap == "" {
		log.Printf("[taskAuth] kafka not configured — domain events disabled")
		return
	}
	kafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(bootstrap),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		BatchTimeout: 100 * time.Millisecond,
		// Transport with shorter dial timeout so failures are fast
		Transport: &kafka.Transport{
			Dial: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		},
	}
	log.Printf("[taskAuth] kafka writer initialized: bootstrap=%s", bootstrap)
}

// closeKafkaWriter flushes and closes the shared writer (call on shutdown).
func closeKafkaWriter() {
	kafkaWriterMu.Lock()
	defer kafkaWriterMu.Unlock()
	if kafkaWriter != nil {
		if err := kafkaWriter.Close(); err != nil {
			log.Printf("[taskAuth] kafka writer close error: %v", err)
		}
		kafkaWriter = nil
	}
}

// publishUserLoggedIn publishes USER_LOGGED_IN event asynchronously.
// providerApp（可选）标识微信登录来源应用（v64 provider_app 字段，向后兼容空值）。
func publishUserImpersonationStarted(ctx context.Context, actorID, targetID string, sessionID int64, reasonLen int) {
	key := strconv.FormatInt(sessionID, 10)
	data := map[string]interface{}{
		"actor_user_id":  actorID,
		"target_user_id": targetID,
		"session_id":     key,
		"reason_len":     reasonLen,
	}
	if err := publishDomainEventKafka(ctx, "UserImpersonationStarted", data, key); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish UserImpersonationStarted failed: %v", err)
		}
	}
}

func publishUserImpersonationStopped(ctx context.Context, actorID, targetID string, sessionID int64) {
	key := strconv.FormatInt(sessionID, 10)
	data := map[string]interface{}{
		"actor_user_id":  actorID,
		"target_user_id": targetID,
		"session_id":     key,
	}
	if err := publishDomainEventKafka(ctx, "UserImpersonationStopped", data, key); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish UserImpersonationStopped failed: %v", err)
		}
	}
}

func publishUserInboxMessageCreated(ctx context.Context, messageID int64, recipientID, actorID string, sessionID int64) {
	key := strconv.FormatInt(messageID, 10)
	data := map[string]interface{}{
		"message_id":               key,
		"recipient_user_id":        recipientID,
		"actor_user_id":            actorID,
		"kind":                     "impersonation_notice",
		"impersonation_session_id": strconv.FormatInt(sessionID, 10),
	}
	if err := publishDomainEventKafka(ctx, "UserInboxMessageCreated", data, key); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish UserInboxMessageCreated failed: %v", err)
		}
	}
}

func publishUserLoggedIn(ctx context.Context, userID, identifier, methodType, providerApp string) {
	publishUserLoggedInExtra(ctx, userID, identifier, methodType, providerApp, "", "", "")
}

// publishWechatIdentityLinked 跨应用身份别名收敛（绑定转移）事件。
func publishWechatIdentityLinked(ctx context.Context, userID, fromUserID, appKey, openID, unionID string) {
	data := map[string]interface{}{
		"user_id":      userID,
		"from_user_id": fromUserID,
		"app_key":      appKey,
		"openid":       openID,
		"unionid":      unionID,
	}
	if err := publishDomainEventKafka(ctx, "WECHAT_IDENTITY_LINKED", data, userID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish WECHAT_IDENTITY_LINKED failed: %v", err)
		}
	}
}

// publishWechatIdentityConflict 身份双射异常 / 绑定冲突事件（运维告警）。
func publishWechatIdentityConflict(ctx context.Context, ownerUserID, appKey, openID, unionID, ownerUnionID string) {
	data := map[string]interface{}{
		"owner_user_id": ownerUserID,
		"app_key":       appKey,
		"openid":        openID,
		"unionid":       unionID,
		"owner_unionid": ownerUnionID,
	}
	if err := publishDomainEventKafka(ctx, "WECHAT_IDENTITY_CONFLICT", data, ownerUserID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish WECHAT_IDENTITY_CONFLICT failed: %v", err)
		}
	}
}

// publishWechatBound 用户绑定微信到既有账号事件。
func publishWechatBound(ctx context.Context, userID, appKey, openID, unionID string) {
	data := map[string]interface{}{
		"user_id": userID,
		"app_key": appKey,
		"openid":  openID,
		"unionid": unionID,
	}
	if err := publishDomainEventKafka(ctx, "WECHAT_BOUND", data, userID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish WECHAT_BOUND failed: %v", err)
		}
	}
}

// publishWechatMpSubscribed 服务号关注：bound（锁定已有用户）或 pending（unionid 未命中）。
func publishWechatMpSubscribed(ctx context.Context, userID, openID, unionID, outcome string) {
	publishWechatMpSubscribedTicket(ctx, userID, openID, unionID, outcome, "")
}

func publishWechatMpSubscribedTicket(ctx context.Context, userID, openID, unionID, outcome, tempID string) {
	data := map[string]interface{}{
		"user_id": userID,
		"app_key": "mp",
		"openid":  openID,
		"unionid": unionID,
		"outcome": outcome,
		"temp_id": tempID,
	}
	key := openID
	if strings.EqualFold(outcome, "conflict") && tempID != "" {
		key = tempID
	}
	if key == "" {
		key = unionID
	}
	if err := publishDomainEventKafka(ctx, "WECHAT_MP_SUBSCRIBED", data, key); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish WECHAT_MP_SUBSCRIBED failed: %v", err)
		}
	}
}

// publishWechatUnbound 用户解绑微信事件。
func publishWechatUnbound(ctx context.Context, userID, appKey string) {
	data := map[string]interface{}{
		"user_id": userID,
		"app_key": appKey,
	}
	if err := publishDomainEventKafka(ctx, "WECHAT_UNBOUND", data, userID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish WECHAT_UNBOUND failed: %v", err)
		}
	}
}

func publishUserTesterFlagChanged(ctx context.Context, userID string, isTester bool) {
	data := map[string]interface{}{
		"user_id":   userID,
		"is_tester": isTester,
	}
	if err := publishDomainEventKafka(ctx, "UserTesterFlagChanged", data, userID); err != nil {
		if err != errKafkaNotConfigured {
			log.Printf("[taskAuth] publish UserTesterFlagChanged failed: %v", err)
		}
	}
}

// publishUserCreated publishes USER_CREATED event.
func publishUserCreated(ctx context.Context, userID, phone, email, username string) {
	data := map[string]interface{}{
		"user_id":  userID,
		"phone":    phone,
		"email":    email,
		"username": username,
	}
	if err := publishDomainEventKafka(ctx, "USER_CREATED", data, userID); err != nil {
		if err == errKafkaNotConfigured {
			log.Printf("[taskAuth] USER_CREATED event skipped: kafka not configured")
			return
		}
		log.Printf("[taskAuth] publish USER_CREATED failed: %v", err)
	}
}

// publishUserCreatedAsync fires USER_CREATED in a goroutine with its own
// background context so the Kafka publish is not killed when the HTTP handler
// returns (unlike r.Context() which is cancelled after the response is sent).
func publishUserCreatedAsync(userID, phone, email, username string) {
	go func() {
		publishUserCreated(context.Background(), userID, phone, email, username)
	}()
}

// publishUserActivated publishes USER_ACTIVATED event.
func publishUserActivated(ctx context.Context, userID, methodType, identifier string) {
	data := map[string]interface{}{
		"user_id":               userID,
		"activated_method_type": methodType,
		"activated_identifier":  identifier,
	}
	if err := publishDomainEventKafka(ctx, "USER_ACTIVATED", data, userID); err != nil {
		if err == errKafkaNotConfigured {
			return
		}
		log.Printf("[taskAuth] publish USER_ACTIVATED failed: %v", err)
	}
}

// emailDeliveryMethod indicates how an email was sent.
// "kafka" = published to message queue (async, actual delivery pending)
// "smtp"  = sent directly via SMTP (synchronous, delivery confirmed)
// ""      = neither succeeded
type emailDeliveryMethod string

const (
	emailDeliveryKafka emailDeliveryMethod = "kafka"
	emailDeliverySMTP  emailDeliveryMethod = "smtp"
)

// publishEmailSent 发送邮件。verification_code（OTP）同步 SMTP，成功才返回；
// 其它模板仍 Kafka 优先、SMTP 回退。Kafka 入队成功不等于已投递。
func publishEmailSent(ctx context.Context, toEmail, subject, templateName string, contextData map[string]interface{}) (emailDeliveryMethod, error) {
	smtpCfg := loadSMTPConfig()
	from := smtpCfg.DefaultFrom
	data := map[string]interface{}{
		"subject":        subject,
		"template_name":  templateName,
		"context":        contextData,
		"from_email":     from,
		"recipient_list": []string{toEmail},
	}
	textBody := buildFallbackEmailText(subject, templateName, contextData)
	htmlBody := buildFallbackEmailHTML(subject, templateName, contextData)
	// OTP 5 分钟 TTL：必须同步 SMTP。Kafka 入队成功曾让前端显示「验证码已发送」而 QQ SMTP 535 从未投递。
	if templateName == "verification_code" {
		if smtpErr := sendEmailSMTP(from, []string{toEmail}, subject, textBody, htmlBody); smtpErr != nil {
			emitTrace(ctx, "error", "EMAIL_SENT: verification OTP SMTP failed", map[string]string{"to": toEmail, "err": smtpErr.Error()})
			return "", fmt.Errorf("EMAIL_SENT failed: smtp=%w", smtpErr)
		}
		emitTrace(ctx, "info", "EMAIL_SENT: verification OTP delivered via SMTP", map[string]string{"to": toEmail})
		data["message"] = textBody
		data["html_message"] = htmlBody
		data["pre_delivered"] = true
		if kerr := publishDomainEventKafka(ctx, "EMAIL_SENT", data, toEmail); kerr != nil && kerr != errKafkaNotConfigured {
			emitTrace(ctx, "warn", "EMAIL_SENT: OTP SMTP ok but kafka audit failed", map[string]string{"to": toEmail, "err": kerr.Error()})
		}
		return emailDeliverySMTP, nil
	}
	// 优先 Kafka，失败时回退到直接 SMTP 发送（确保用户能收到邮件）
	err := publishDomainEventKafka(ctx, "EMAIL_SENT", data, toEmail)
	if err != nil {
		if err == errKafkaNotConfigured {
			emitTrace(ctx, "info", "EMAIL_SENT: kafka not configured, falling back to direct SMTP", map[string]string{"to": toEmail})
		} else {
			emitTrace(ctx, "warn", "EMAIL_SENT: kafka publish failed — falling back to direct SMTP", map[string]string{"to": toEmail, "err": err.Error()})
		}
		if smtpErr := sendEmailSMTP(from, []string{toEmail}, subject, textBody, htmlBody); smtpErr != nil {
			emitTrace(ctx, "error", "EMAIL_SENT: SMTP fallback also failed", map[string]string{"to": toEmail, "err": smtpErr.Error()})
			return "", fmt.Errorf("EMAIL_SENT failed: kafka=%w, smtp=%v", err, smtpErr)
		}
		emitTrace(ctx, "info", "EMAIL_SENT: delivered via SMTP fallback", map[string]string{"to": toEmail})
		return emailDeliverySMTP, nil
	}
	emitTrace(ctx, "info", "EMAIL_SENT: kafka published", map[string]string{"to": toEmail})
	return emailDeliveryKafka, nil
}

var errKafkaNotConfigured = fmt.Errorf("kafka not configured")

func publishInviteEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) {
	if err := publishDomainEventKafka(ctx, eventType, data, key); err != nil {
		if err == errKafkaNotConfigured {
			log.Printf("[taskAuth] registration invite event %s skipped: kafka not configured", eventType)
			return
		}
		log.Printf("[taskAuth] registration invite event %s failed: %v", eventType, err)
	}
}

func publishDomainEventKafka(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	bootstrap := kafkaBootstrapServers()
	if bootstrap == "" {
		return errKafkaNotConfigured
	}
	topic, ok := eventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}

	msg := kafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}

	// 使用共享 writer（长连接，比每次新建 writer 更可靠）
	kafkaWriterMu.Lock()
	w := kafkaWriter
	kafkaWriterMu.Unlock()

	if w == nil {
		// 回退：writer 未初始化时创建临时 writer
		emitTrace(ctx, "warn", "kafka shared writer not initialized, using one-shot writer", map[string]string{"event": eventType})
		w = &kafka.Writer{
			Addr:         kafka.TCP(bootstrap),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		}
		defer w.Close()
	} else {
		// 共享 writer 使用目标 topic（覆盖 writer 默认 topic）
		w = &kafka.Writer{
			Addr:         kafka.TCP(bootstrap),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
			Transport:    w.Transport,
		}
		defer w.Close()
	}

	// 带重试的写入（最多 3 次，指数退避）
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
			emitTrace(ctx, "info", "kafka publish retry", map[string]string{"event": eventType, "topic": topic, "attempt": strconv.Itoa(attempt), "max_retries": strconv.Itoa(maxRetries - 1)})
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		// 使用带超时的 context（避免无限阻塞）
		writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.WriteMessages(writeCtx, msg)
		cancel()

		if err == nil {
			emitTrace(ctx, "info", "kafka published", map[string]string{"event": eventType, "topic": topic, "key": key, "attempt": strconv.Itoa(attempt + 1)})
			return nil
		}
		lastErr = err
		emitTrace(ctx, "warn", "kafka publish attempt failed", map[string]string{"event": eventType, "topic": topic, "attempt": strconv.Itoa(attempt + 1), "err": err.Error()})
	}
	emitTrace(ctx, "error", "kafka publish all attempts failed", map[string]string{"event": eventType, "topic": topic, "max_retries": strconv.Itoa(maxRetries), "err": lastErr.Error()})
	return lastErr
}
