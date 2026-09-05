package gitlabmanualnode

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"tracelog"

	"taskEvents/domain"
	"taskEvents/notifications/smtp"
)

// MailSender is the SMTP surface the ops alert needs (satisfied by smtp.Sender).
type MailSender interface {
	Send(from string, to []string, subject, textBody, htmlBody string) error
}

// EventType is the taskBill-published domain event this intent consumes.
const EventType = "GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED"

// AlertMessage is the payload of a GitLab manual-node-fulfillment queued event.
type AlertMessage struct {
	TenantID      string
	OrderID       string
	RegionSlug    string
	CloudProvider string
}

// ParseAlertMessage decodes the envelope payload. order_id/region_slug are
// required; tenant_id/cloud_provider are informational.
func fieldString(raw map[string]interface{}, key string) string {
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}

func ParseAlertMessage(data json.RawMessage) (AlertMessage, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return AlertMessage{}, fmt.Errorf("bad payload: %w", err)
	}
	msg := AlertMessage{
		TenantID:      fieldString(raw, "tenant_id"),
		OrderID:       fieldString(raw, "order_id"),
		RegionSlug:    fieldString(raw, "region_slug"),
		CloudProvider: fieldString(raw, "cloud_provider"),
	}
	if msg.OrderID == "" {
		return AlertMessage{}, fmt.Errorf("missing order_id")
	}
	if msg.RegionSlug == "" {
		return AlertMessage{}, fmt.Errorf("missing region_slug")
	}
	return msg, nil
}

// Subject renders the ops alert subject (no tenant-scoped fields to keep it greppable).
func (m AlertMessage) Subject() string {
	return fmt.Sprintf("[运维告警] GitLab 待手动建节点 order=%s region=%s", m.OrderID, m.RegionSlug)
}

// TextBody renders the plain-text alert body.
func (m AlertMessage) TextBody() string {
	var b strings.Builder
	b.WriteString("GitLab 区域待人工建节点，请及时处理：\n\n")
	fmt.Fprintf(&b, "order_id:      %s\n", m.OrderID)
	fmt.Fprintf(&b, "tenant_id:     %s\n", m.TenantID)
	fmt.Fprintf(&b, "region_slug:   %s\n", m.RegionSlug)
	fmt.Fprintf(&b, "cloud_provider:%s\n", m.CloudProvider)
	b.WriteString("\n请登录系统管理后台完成手动建节点（infra_status: pending_node -> ready）。\n")
	return b.String()
}

// HTMLBody renders a simple HTML variant.
func (m AlertMessage) HTMLBody() string {
	return fmt.Sprintf(
		"<p>GitLab 区域待人工建节点，请及时处理：</p><ul>"+
			"<li>order_id: <b>%s</b></li>"+
			"<li>tenant_id: %s</li>"+
			"<li>region_slug: <b>%s</b></li>"+
			"<li>cloud_provider: %s</li></ul>"+
			"<p>请登录系统管理后台完成手动建节点（infra_status: pending_node -&gt; ready）。</p>",
		htmlEscape(m.OrderID), htmlEscape(m.TenantID), htmlEscape(m.RegionSlug), htmlEscape(m.CloudProvider),
	)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// Handler is a read-only ops alert consumer for GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED.
// It sends one email per order_id; replaying the same order_id is deduped upstream
// by the idempotency key (consumer.IdempotencyKeyFromEnvelope → order_id).
type Handler struct {
	Sender MailSender
	From   string
	To     string // ops 收件人（DLT_ALERT_EMAIL 同一收件箱）
}

// Dispatch implements domain.DomainCommandPort.
func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != EventType {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	msg, err := ParseAlertMessage(cmd.Envelope.Data)
	if err != nil {
		tracelog.LogEventConsume(ctx, "gitlab_manual_node_bad_payload", cmd.EventType, map[string]any{
			"error": err.Error(),
		})
		return domain.DispatchPermanent, err
	}

	corr := tracelog.CorrelationFromEnvelopeData(cmd.Envelope.Data)
	tracelog.EmitComponent("info", "gitlab manual node fulfillment queued", "consumer", corr.TraceID, map[string]string{
		"order_id":       msg.OrderID,
		"tenant_id":      msg.TenantID,
		"region_slug":    msg.RegionSlug,
		"cloud_provider": msg.CloudProvider,
	})

	recipient := strings.TrimSpace(h.To)
	if h.Sender == nil || recipient == "" {
		// SMTP/收件人未配置：仅保留审计日志，避免事件永久卡在重试（生产由 main 启动日志暴露）。
		log.Printf("[gitlab-manual-node] ops alert disabled order_id=%s region=%s (no sender/recipient)", msg.OrderID, msg.RegionSlug)
		return domain.DispatchSuccess, nil
	}

	if err := h.Sender.Send(strings.TrimSpace(h.From), []string{recipient}, msg.Subject(), msg.TextBody(), msg.HTMLBody()); err != nil {
		if smtp.IsPermanent(err) {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	log.Printf("[gitlab-manual-node] ops alert sent order_id=%s region=%s to=%s", msg.OrderID, msg.RegionSlug, recipient)
	return domain.DispatchSuccess, nil
}
