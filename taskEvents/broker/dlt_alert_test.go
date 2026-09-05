package broker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type stubMailSender struct {
	mu    sync.Mutex
	calls []mailCall
	err   error
}

type mailCall struct {
	from    string
	to      []string
	subject string
	text    string
}

func (s *stubMailSender) Send(from string, to []string, subject, textBody, htmlBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, mailCall{from: from, to: append([]string{}, to...), subject: subject, text: textBody})
	return s.err
}

func (s *stubMailSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *stubMailSender) last() mailCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return mailCall{}
	}
	return s.calls[len(s.calls)-1]
}

func sampleDLT() DLTMessage {
	return DLTMessage{
		OriginalEventType: "USER_CREATED",
		OriginalTopic:     "user-created",
		OriginalKey:       "user-1",
		Error:             "handler boom",
		DeadLetteredAt:    "2026-08-10T11:00:00Z",
		FailureReason:     FailureReasonPermanent,
		RetryCount:        1,
	}
}

func TestThrottledDLTAlerter_FirstNotifySends(t *testing.T) {
	sender := &stubMailSender{}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: 5 * time.Minute,
		Sender:   sender,
	})
	alerter.now = func() time.Time { return now }

	alerter.Notify(sampleDLT())

	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1", sender.count())
	}
	c := sender.last()
	if len(c.to) != 1 || c.to[0] != "contact@daydaymoney.com" {
		t.Fatalf("to = %v, want [contact@daydaymoney.com]", c.to)
	}
	if !strings.Contains(c.text, "USER_CREATED") {
		t.Fatalf("body missing event type: %q", c.text)
	}
	if !strings.Contains(c.text, FailureReasonPermanent) {
		t.Fatalf("body missing failure reason: %q", c.text)
	}
	if !strings.Contains(c.text, "handler boom") {
		t.Fatalf("body missing error: %q", c.text)
	}
	if !strings.Contains(c.text, "user-created-dlt") && !strings.Contains(c.text, "user-created") {
		t.Fatalf("body missing topic hint: %q", c.text)
	}
}

func TestThrottledDLTAlerter_SuppressWithinCooldown(t *testing.T) {
	sender := &stubMailSender{}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: 5 * time.Minute,
		Sender:   sender,
	})
	alerter.now = func() time.Time { return now }

	alerter.Notify(sampleDLT())
	alerter.Notify(sampleDLT())

	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 (second suppressed)", sender.count())
	}
}

func TestThrottledDLTAlerter_ResendAfterCooldown(t *testing.T) {
	sender := &stubMailSender{}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: 5 * time.Minute,
		Sender:   sender,
	})
	alerter.now = func() time.Time { return now }

	alerter.Notify(sampleDLT())
	now = now.Add(5 * time.Minute)
	alerter.Notify(sampleDLT())

	if sender.count() != 2 {
		t.Fatalf("Send calls = %d, want 2 after cooldown", sender.count())
	}
}

func TestThrottledDLTAlerter_SenderErrorStillAdvancesCooldown(t *testing.T) {
	sender := &stubMailSender{err: errors.New("smtp down")}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: 5 * time.Minute,
		Sender:   sender,
	})
	alerter.now = func() time.Time { return now }

	alerter.Notify(sampleDLT())
	alerter.Notify(sampleDLT())

	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 (cooldown advanced despite error)", sender.count())
	}
}

func TestResolveDLTAlertEmail_DefaultAndEnv(t *testing.T) {
	t.Setenv("DLT_ALERT_EMAIL", "")
	if got := ResolveDLTAlertEmail(); got != DefaultDLTAlertEmail {
		t.Fatalf("default email = %q, want %q", got, DefaultDLTAlertEmail)
	}
	t.Setenv("DLT_ALERT_EMAIL", "ops@example.com")
	if got := ResolveDLTAlertEmail(); got != "ops@example.com" {
		t.Fatalf("env email = %q, want ops@example.com", got)
	}
}

func TestResolveDLTAlertCooldown_DefaultAndEnv(t *testing.T) {
	t.Setenv("DLT_ALERT_COOLDOWN", "")
	if got := ResolveDLTAlertCooldown(); got != DefaultDLTAlertCooldown {
		t.Fatalf("default cooldown = %v, want %v", got, DefaultDLTAlertCooldown)
	}
	t.Setenv("DLT_ALERT_COOLDOWN", "2m")
	if got := ResolveDLTAlertCooldown(); got != 2*time.Minute {
		t.Fatalf("env cooldown = %v, want 2m", got)
	}
	t.Setenv("DLT_ALERT_COOLDOWN", "not-a-duration")
	if got := ResolveDLTAlertCooldown(); got != DefaultDLTAlertCooldown {
		t.Fatalf("invalid cooldown = %v, want default", got)
	}
}

func TestMaybeAlertDLT_NoopWithoutAlerter(t *testing.T) {
	SetDLTAlerter(nil)
	t.Cleanup(func() { SetDLTAlerter(nil) })
	maybeAlertDLT(sampleDLT()) // must not panic
}

func TestMaybeAlertDLT_InvokesConfiguredAlerter(t *testing.T) {
	sender := &stubMailSender{}
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: time.Minute,
		Sender:   sender,
	})
	SetDLTAlerter(alerter)
	t.Cleanup(func() { SetDLTAlerter(nil) })

	maybeAlertDLT(sampleDLT())

	// maybeAlertDLT is sync for testability when called directly
	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1", sender.count())
	}
}

func TestThrottledDLTAlerter_SuppressedCountInNextReport(t *testing.T) {
	sender := &stubMailSender{}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:       "contact@daydaymoney.com",
		From:     "noreply@example.com",
		Cooldown: 5 * time.Minute,
		Sender:   sender,
	})
	alerter.now = func() time.Time { return now }

	// First event sends a report with suppressed_since_last_email: 0.
	alerter.Notify(sampleDLT())
	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1", sender.count())
	}
	if !strings.Contains(sender.last().text, "suppressed_since_last_email: 0") {
		t.Fatalf("first report should show 0 suppressed: %q", sender.last().text)
	}

	// Two suppressed events within the window.
	now = now.Add(1 * time.Minute)
	alerter.Notify(sampleDLT())
	now = now.Add(2 * time.Minute)
	alerter.Notify(sampleDLT())
	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 (both suppressed)", sender.count())
	}

	// After the cooldown the next report carries the accumulated count and resets.
	now = now.Add(5 * time.Minute)
	alerter.Notify(sampleDLT())
	if sender.count() != 2 {
		t.Fatalf("Send calls = %d, want 2 after cooldown", sender.count())
	}
	if !strings.Contains(sender.last().text, "suppressed_since_last_email: 2") {
		t.Fatalf("second report should show 2 suppressed: %q", sender.last().text)
	}

	// Counter resets: the next suppressed+report cycle starts from 0 again.
	now = now.Add(1 * time.Minute)
	alerter.Notify(sampleDLT()) // suppressed, count=1
	now = now.Add(5 * time.Minute)
	alerter.Notify(sampleDLT()) // sends, count resets and reports 1
	if sender.count() != 3 {
		t.Fatalf("Send calls = %d, want 3", sender.count())
	}
	if !strings.Contains(sender.last().text, "suppressed_since_last_email: 1") {
		t.Fatalf("third report should show 1 suppressed (reset): %q", sender.last().text)
	}
}

// stubCooldownStore is a deterministic DLTCooldownStore for unit tests.
type stubCooldownStore struct {
	claim bool
	err   error
	calls int
}

func (s *stubCooldownStore) Acquire(_ context.Context, _ time.Duration) (bool, error) {
	s.calls++
	return s.claim, s.err
}

func TestThrottledDLTAlerter_CrossProcessStoreWinsSends(t *testing.T) {
	sender := &stubMailSender{}
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:            "contact@daydaymoney.com",
		From:          "noreply@example.com",
		Cooldown:      5 * time.Minute,
		Sender:        sender,
		CooldownStore: &stubCooldownStore{claim: true},
	})

	alerter.Notify(sampleDLT())

	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 (won cross-process window)", sender.count())
	}
}

func TestThrottledDLTAlerter_CrossProcessStoreSuppresses(t *testing.T) {
	sender := &stubMailSender{}
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:            "contact@daydaymoney.com",
		From:          "noreply@example.com",
		Cooldown:      5 * time.Minute,
		Sender:        sender,
		CooldownStore: &stubCooldownStore{claim: false},
	})

	alerter.Notify(sampleDLT())

	if sender.count() != 0 {
		t.Fatalf("Send calls = %d, want 0 (another process claimed the window)", sender.count())
	}
}

func TestThrottledDLTAlerter_CrossProcessStoreErrorFallsBackToLocal(t *testing.T) {
	sender := &stubMailSender{}
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	alerter := NewThrottledDLTAlerter(DLTAlertConfig{
		To:            "contact@daydaymoney.com",
		From:          "noreply@example.com",
		Cooldown:      5 * time.Minute,
		Sender:        sender,
		CooldownStore: &stubCooldownStore{err: errors.New("redis down")},
	})
	alerter.now = func() time.Time { return now }

	// Store error → process-local window still throttles.
	alerter.Notify(sampleDLT())
	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 after store fallback", sender.count())
	}
	alerter.Notify(sampleDLT())
	if sender.count() != 1 {
		t.Fatalf("Send calls = %d, want 1 (local cooldown suppresses)", sender.count())
	}
}

func TestNewRedisDLTCooldownStore_KeyDefaults(t *testing.T) {
	// Key defaulting must not panic and store must be non-nil.
	s := NewRedisDLTCooldownStore("127.0.0.1", 6379, 0, "")
	if s == nil {
		t.Fatal("store is nil")
	}
}
