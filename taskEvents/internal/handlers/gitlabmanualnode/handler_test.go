package gitlabmanualnode

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"taskEvents/domain"
)

type fakeSender struct {
	calls  int
	from   string
	to     []string
	subj   string
	text   string
	html   string
	sendFn func(from string, to []string, subject, textBody, htmlBody string) error
}

func (f *fakeSender) Send(from string, to []string, subject, textBody, htmlBody string) error {
	f.calls++
	f.from = from
	f.to = to
	f.subj = subject
	f.text = textBody
	f.html = htmlBody
	if f.sendFn != nil {
		return f.sendFn(from, to, subject, textBody, htmlBody)
	}
	return nil
}

func envelopeData(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseAlertMessage(t *testing.T) {
	msg, err := ParseAlertMessage(envelopeData(t, map[string]any{
		"tenant_id": "9300000201", "order_id": "9300000199",
		"region_slug": "aliyun-cn-hangzhou", "cloud_provider": "aliyun",
	}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msg.OrderID != "9300000199" || msg.RegionSlug != "aliyun-cn-hangzhou" {
		t.Fatalf("unexpected msg=%+v", msg)
	}
	if _, err := ParseAlertMessage(envelopeData(t, map[string]any{"region_slug": "x"})); err == nil {
		t.Fatal("want missing order_id error")
	}
	if _, err := ParseAlertMessage(envelopeData(t, map[string]any{"order_id": "1"})); err == nil {
		t.Fatal("want missing region_slug error")
	}
	if _, err := ParseAlertMessage(json.RawMessage(`{bad`)); err == nil {
		t.Fatal("want bad payload error")
	}
}

func TestAlertMessageRendersBody(t *testing.T) {
	msg := AlertMessage{TenantID: "9300000201", OrderID: "9300000199", RegionSlug: "aliyun-cn-hangzhou", CloudProvider: "aliyun"}
	if !strings.Contains(msg.Subject(), "9300000199") || !strings.Contains(msg.Subject(), "aliyun-cn-hangzhou") {
		t.Fatalf("subject=%q", msg.Subject())
	}
	if !strings.Contains(msg.TextBody(), "9300000199") || !strings.Contains(msg.TextBody(), "9300000201") {
		t.Fatalf("text=%q", msg.TextBody())
	}
	if !strings.Contains(msg.HTMLBody(), "9300000199") || !strings.Contains(msg.HTMLBody(), "<b>aliyun-cn-hangzhou</b>") {
		t.Fatalf("html=%q", msg.HTMLBody())
	}
}

func TestHandlerDispatchSendsOpsEmail(t *testing.T) {
	sender := &fakeSender{}
	h := &Handler{Sender: sender, From: "ops@daydaymoney.com", To: "contact@daydaymoney.com"}
	cmd := domain.DomainCommand{
		EventType: EventType,
		Envelope: domain.EventEnvelope{
			EventType: EventType,
			Data: envelopeData(t, map[string]any{
				"tenant_id": "9300000201", "order_id": "9300000199",
				"region_slug": "aliyun-cn-hangzhou", "cloud_provider": "aliyun",
			}),
			Key: "9300000199",
		},
	}
	out, err := h.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", out)
	}
	if sender.calls != 1 {
		t.Fatalf("calls=%d want 1", sender.calls)
	}
	if sender.to[0] != "contact@daydaymoney.com" || sender.from != "ops@daydaymoney.com" {
		t.Fatalf("to=%v from=%q", sender.to, sender.from)
	}
	if !strings.Contains(sender.subj, "9300000199") || !strings.Contains(sender.text, "9300000201") {
		t.Fatalf("subj=%q text=%q", sender.subj, sender.text)
	}
}

func TestHandlerDispatchDisabledWhenNoSender(t *testing.T) {
	h := &Handler{Sender: nil, To: ""}
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: EventType,
		Envelope: domain.EventEnvelope{
			EventType: EventType,
			Data:      envelopeData(t, map[string]any{"order_id": "9300000199", "region_slug": "aliyun-cn-hangzhou"}),
			Key:       "9300000199",
		},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", out)
	}
}

func TestHandlerDispatchSMTPOutcomeMapping(t *testing.T) {
	build := func(sendFn func(string, []string, string, string, string) error) *Handler {
		return &Handler{Sender: &fakeSender{sendFn: sendFn}, From: "ops@daydaymoney.com", To: "contact@daydaymoney.com"}
	}
	cmd := domain.DomainCommand{
		EventType: EventType,
		Envelope: domain.EventEnvelope{EventType: EventType,
			Data: envelopeData(t, map[string]any{"order_id": "9300000199", "region_slug": "aliyun-cn-hangzhou"}),
			Key:  "9300000199",
		},
	}
	// 535 login fail → permanent
	out, err := build(func(string, []string, string, string, string) error {
		return errors.New("535 Error: authentication failed")
	}).Dispatch(context.Background(), cmd)
	if err == nil || out != domain.DispatchPermanent {
		t.Fatalf("permanent err=%v out=%v", err, out)
	}
	// dial tcp timeout → retryable
	out, err = build(func(string, []string, string, string, string) error {
		return errors.New("dial tcp: i/o timeout")
	}).Dispatch(context.Background(), cmd)
	if err == nil || out != domain.DispatchRetryable {
		t.Fatalf("retryable err=%v out=%v", err, out)
	}
}

func TestHandlerDispatchRejectsWrongEvent(t *testing.T) {
	h := &Handler{Sender: &fakeSender{}, To: "contact@daydaymoney.com"}
	_, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "OTHER_EVENT",
		Envelope:  domain.EventEnvelope{EventType: "OTHER_EVENT", Data: envelopeData(t, map[string]any{})},
	})
	if err == nil {
		t.Fatal("want unsupported event error")
	}
}
