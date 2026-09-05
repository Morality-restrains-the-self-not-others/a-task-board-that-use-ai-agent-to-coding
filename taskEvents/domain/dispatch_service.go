package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"tracelog"
)

// IdempotentDispatchService applies domain commands at-most-once per IdempotencyKey.
type IdempotentDispatchService struct {
	Commands    DomainCommandPort
	Idempotency IdempotencyStorePort
	KeyFor      func(EventEnvelope) IdempotencyKey
}

// Handle processes one broker message; caller Ack's on DispatchSuccess.
func (s *IdempotentDispatchService) Handle(ctx context.Context, cmd DomainCommand) (DispatchOutcome, error) {
	key := s.KeyFor(cmd.Envelope)
	if s.Idempotency.Seen(key) {
		// OPT-20260807-018 加固: 幂等命中此前静默跳过 — 线上 EMAIL_SENT 键碰撞事故
		// （同收件人全部邮件共用一个幂等键）期间日志无迹可查，仅能靠 Kafka 对账发现。
		// 命中即 warn 输出（含 event_type + 键指纹），使"事件被去重"成为可检索事实。
		// 注意: 幂等键内含密码重置/激活 token 明文（IdempotencyKeyForEvent 明文拼接），
		// 完整键写日志即泄漏可重放凭据 — 只能输出 SHA-256 指纹（security-and-hardening）。
		corr := tracelog.CorrelationFromEnvelopeData(cmd.Envelope.Data)
		tracelog.EmitComponent("warn", "idempotency skip", "consumer", corr.TraceID, map[string]string{
			"event_type":      cmd.Envelope.EventType,
			"idempotency_key": idempotencyKeyFingerprint(string(key)),
		})
		return DispatchSuccess, nil
	}
	out, err := s.Commands.Dispatch(ctx, cmd)
	if err != nil {
		return out, err
	}
	if out == DispatchSuccess {
		s.Idempotency.Mark(key)
	}
	return out, nil
}

// idempotencyKeyFingerprint 把幂等键转换为 SHA-256 截断指纹（128-bit）。
// 幂等键由 IdempotencyKeyForEvent 明文拼接业务标识（密码重置 token、激活 token 等），
// 完整键进日志/追踪即泄漏可重放凭据 — 可观测性字段一律输出指纹。
func idempotencyKeyFingerprint(key string) string {
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16])
}
