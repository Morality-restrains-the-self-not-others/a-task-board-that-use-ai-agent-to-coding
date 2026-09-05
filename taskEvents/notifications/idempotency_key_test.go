package notifications_test

import (
	"encoding/json"
	"testing"

	"taskEvents/domain"
	"taskEvents/notifications"
)

// makeEmailEnvelope 构造一条 EMAIL_SENT 事件，Key 为 Kafka 消息 key（发布方用收件人邮箱分区）。
func makeEmailEnvelope(key string, data map[string]interface{}) domain.EventEnvelope {
	raw, _ := json.Marshal(data)
	return domain.EventEnvelope{EventType: "EMAIL_SENT", Data: raw, Key: key}
}

// OPT-20260807-018 回归护栏: 同一收件人的不同业务邮件（验证码 / 重置 / 激活…）
// 必须产生不同的幂等键。此前幂等键 = recipient + ":" + 消息key（= 收件人邮箱），
// 同一收件人的首封邮件 Mark 后，后续所有邮件被静默去重丢弃（本事故根因）。
func TestIdempotencyKeyEmailSentDistinguishesTemplates(t *testing.T) {
	email := "contact@daydaymoney.com"
	verification := makeEmailEnvelope(email, map[string]interface{}{
		"template_name":  "verification_code",
		"recipient_list": []string{email},
		"context":        map[string]interface{}{"code": "589001"},
	})
	reset := makeEmailEnvelope(email, map[string]interface{}{
		"template_name":  "password_reset",
		"recipient_list": []string{email},
		"context":        map[string]interface{}{"reset_url": "/auth/reset-password/token-A/"},
	})
	k1 := notifications.IdempotencyKeyForEnvelope(verification)
	k2 := notifications.IdempotencyKeyForEnvelope(reset)
	if k1 == k2 {
		t.Fatalf("verification and password_reset emails to same recipient share key %q", k1)
	}
}

// 同一收件人两次密码重置（两次点击产生不同 token）必须产生不同幂等键，
// 否则用户第二次点击重置仍收不到邮件。
func TestIdempotencyKeyEmailSentDistinguishesTokens(t *testing.T) {
	email := "contact@daydaymoney.com"
	mk := func(token string) domain.EventEnvelope {
		return makeEmailEnvelope(email, map[string]interface{}{
			"template_name":  "password_reset",
			"recipient_list": []string{email},
			"context":        map[string]interface{}{"reset_url": "/auth/reset-password/" + token + "/"},
		})
	}
	if a, b := notifications.IdempotencyKeyForEnvelope(mk("token-A")), notifications.IdempotencyKeyForEnvelope(mk("token-B")); a == b {
		t.Fatalf("two reset emails with different tokens share key %q", a)
	}
}

// 同一事件 Kafka 重投（同 token 同收件人）必须产生相同幂等键，保持 at-least-once 去重。
func TestIdempotencyKeyEmailSentDedupesRedelivery(t *testing.T) {
	email := "user@example.com"
	ev := makeEmailEnvelope(email, map[string]interface{}{
		"template_name":  "password_reset",
		"recipient_list": []string{email},
		"context":        map[string]interface{}{"reset_url": "/auth/reset-password/token-X/"},
	})
	if a, b := notifications.IdempotencyKeyForEnvelope(ev), notifications.IdempotencyKeyForEnvelope(ev); a != b {
		t.Fatalf("redelivered same event must share key, got %q vs %q", a, b)
	}
}

// 无 token 模板（welcome）: 不同收件人必须产生不同幂等键，
// 否则用户 A 的欢迎邮件发出后用户 B 的欢迎邮件会被吞。
func TestIdempotencyKeyEmailSentWelcomePerRecipient(t *testing.T) {
	mk := func(email string) domain.EventEnvelope {
		return makeEmailEnvelope(email, map[string]interface{}{
			"template_name":  "welcome",
			"recipient_list": []string{email},
		})
	}
	if a, b := notifications.IdempotencyKeyForEnvelope(mk("a@example.com")), notifications.IdempotencyKeyForEnvelope(mk("b@example.com")); a == b {
		t.Fatalf("welcome emails to different recipients share key %q", a)
	}
}

// 验证码模板: 不同验证码（重新发送）必须产生不同幂等键。
func TestIdempotencyKeyEmailSentVerificationCodeReroll(t *testing.T) {
	email := "user@example.com"
	mk := func(code string) domain.EventEnvelope {
		return makeEmailEnvelope(email, map[string]interface{}{
			"template_name":  "verification_code",
			"recipient_list": []string{email},
			"context":        map[string]interface{}{"code": code},
		})
	}
	if a, b := notifications.IdempotencyKeyForEnvelope(mk("111111")), notifications.IdempotencyKeyForEnvelope(mk("222222")); a == b {
		t.Fatalf("different verification codes share key %q", a)
	}
}
