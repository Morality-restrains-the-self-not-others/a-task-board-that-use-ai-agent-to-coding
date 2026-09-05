package broker

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultDLTAlertEmail is the ops report recipient when DLT_ALERT_EMAIL is unset.
const DefaultDLTAlertEmail = "contact@daydaymoney.com"

// DefaultDLTAlertCooldown is the suppress window for repeated DLT report emails.
const DefaultDLTAlertCooldown = 5 * time.Minute

// MailSender sends a single email message (SMTP adapter injected by consumer).
type MailSender interface {
	Send(from string, to []string, subject, textBody, htmlBody string) error
}

// DLTAlertNotifier is invoked after a successful dead-letter publish.
type DLTAlertNotifier interface {
	Notify(msg DLTMessage)
}

// DLTCooldownStore atomically claims the DLT report cooldown window across
// consumer processes. Return true when this process won the claim (and may send
// the report email), false when another process already claimed it, and a
// non-nil error when the store is unavailable (caller falls back to
// process-local cooldown).
type DLTCooldownStore interface {
	Acquire(ctx context.Context, ttl time.Duration) (bool, error)
}

// RedisDLTCooldownStore is a DLTCooldownStore backed by Redis SET NX + TTL, so
// the "at most one email per cooldown" guarantee holds across all task-events-*
// consumer processes that share the Redis.
type RedisDLTCooldownStore struct {
	client *redis.Client
	key    string
}

// NewRedisDLTCooldownStore builds a store keyed at key (default "dlt-alert:lock").
func NewRedisDLTCooldownStore(host string, port, db int, key string) *RedisDLTCooldownStore {
	if key == "" {
		key = "dlt-alert:lock"
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	if port == 0 {
		addr = fmt.Sprintf("%s:6379", host)
	}
	return &RedisDLTCooldownStore{
		client: redis.NewClient(&redis.Options{Addr: addr, DB: db}),
		key:    key,
	}
}

// Acquire claims the cooldown window via SET NX with TTL = cooldown. A non-nil
// error means Redis is unreachable and the caller should degrade to its
// process-local window.
func (s *RedisDLTCooldownStore) Acquire(ctx context.Context, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, s.key, "1", ttl).Result()
}

// Del removes the cooldown key (used by tests and manual reset).
func (s *RedisDLTCooldownStore) Del(ctx context.Context) error {
	return s.client.Del(ctx, s.key).Err()
}

// DLTAlertConfig configures throttled DLT report emails.
type DLTAlertConfig struct {
	To            string
	From          string
	Cooldown      time.Duration
	Sender        MailSender
	CooldownStore DLTCooldownStore // optional cross-process cooldown store
}

// ThrottledDLTAlerter sends at most one DLT report email per cooldown window.
// With a CooldownStore the window is shared across consumer processes; without
// one it degrades to a process-local window.
type ThrottledDLTAlerter struct {
	cfg        DLTAlertConfig
	mu         sync.Mutex
	lastSent   time.Time
	suppressed int // DLT events suppressed since the last sent report
	now        func() time.Time
}

// NewThrottledDLTAlerter builds an alerter; invalid/zero cooldown falls back to DefaultDLTAlertCooldown.
func NewThrottledDLTAlerter(cfg DLTAlertConfig) *ThrottledDLTAlerter {
	if strings.TrimSpace(cfg.To) == "" {
		cfg.To = DefaultDLTAlertEmail
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = DefaultDLTAlertCooldown
	}
	return &ThrottledDLTAlerter{
		cfg: cfg,
		now: time.Now,
	}
}

// Notify sends a report email unless still inside the cooldown window.
func (a *ThrottledDLTAlerter) Notify(msg DLTMessage) {
	if a == nil || a.cfg.Sender == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// Cross-process cooldown: when a shared store is configured it is the source
	// of truth for the window. On store error we degrade to the process-local
	// window below (the store is optional and must never block alerting).
	if a.cfg.CooldownStore != nil {
		ok, err := a.cfg.CooldownStore.Acquire(context.Background(), a.cfg.Cooldown)
		if err == nil {
			if !ok {
				a.suppressed++
				log.Printf("[dlt-alert] suppressed email (cross-process) event=%s reason=%s cooldown=%s",
					msg.OriginalEventType, msg.FailureReason, a.cfg.Cooldown)
				return
			}
			// Advance cooldown even on send failure to avoid SMTP hammering on bad config.
			a.lastSent = a.now()
			a.sendLocked(msg)
			return
		}
		log.Printf("[dlt-alert] cooldown store unavailable (%v), falling back to process-local", err)
	}

	now := a.now()
	if !a.lastSent.IsZero() && now.Sub(a.lastSent) < a.cfg.Cooldown {
		a.suppressed++
		log.Printf("[dlt-alert] suppressed email event=%s reason=%s cooldown=%s",
			msg.OriginalEventType, msg.FailureReason, a.cfg.Cooldown)
		return
	}

	// Advance cooldown even on send failure to avoid SMTP hammering on bad config.
	a.lastSent = now
	a.sendLocked(msg)
}

// sendLocked sends the report email. suppressed is snapshotted into the body and
// reset so the next report reflects only the events suppressed since this one.
func (a *ThrottledDLTAlerter) sendLocked(msg DLTMessage) {
	suppressed := a.suppressed
	a.suppressed = 0

	subject := fmt.Sprintf("[taskEvents DLT] %s (%s)", msg.OriginalEventType, msg.FailureReason)
	body := formatDLTAlertBody(msg, a.cfg.Cooldown, suppressed)
	from := a.cfg.From
	if err := a.cfg.Sender.Send(from, []string{a.cfg.To}, subject, body, ""); err != nil {
		log.Printf("[dlt-alert] send failed to=%s event=%s: %v", a.cfg.To, msg.OriginalEventType, err)
		return
	}
	log.Printf("[dlt-alert] sent report to=%s event=%s reason=%s topic=%s suppressed_since_last_email=%d",
		a.cfg.To, msg.OriginalEventType, msg.FailureReason, msg.OriginalTopic, suppressed)
}

func formatDLTAlertBody(msg DLTMessage, cooldown time.Duration, suppressed int) string {
	dltTopic := msg.OriginalTopic
	if dltTopic != "" && !strings.HasSuffix(dltTopic, "-dlt") {
		dltTopic = dltTopic + "-dlt"
	}
	var b strings.Builder
	b.WriteString("Kafka consumer dead-lettered an event.\n\n")
	b.WriteString(fmt.Sprintf("event_type: %s\n", msg.OriginalEventType))
	b.WriteString(fmt.Sprintf("original_topic: %s\n", msg.OriginalTopic))
	b.WriteString(fmt.Sprintf("dlt_topic: %s\n", dltTopic))
	b.WriteString(fmt.Sprintf("failure_reason: %s\n", msg.FailureReason))
	b.WriteString(fmt.Sprintf("retry_count: %d\n", msg.RetryCount))
	b.WriteString(fmt.Sprintf("key: %s\n", msg.OriginalKey))
	b.WriteString(fmt.Sprintf("dead_lettered_at: %s\n", msg.DeadLetteredAt))
	if tid := strings.TrimSpace(msg.TraceID); tid != "" {
		b.WriteString(fmt.Sprintf("trace_id: %s\n", tid))
	} else {
		b.WriteString("trace_id: (missing)\n")
	}
	b.WriteString(fmt.Sprintf("error: %s\n", msg.Error))
	if suppressed > 0 {
		b.WriteString(fmt.Sprintf("suppressed_since_last_email: %d\n", suppressed))
		b.WriteString("(additional DLT events were suppressed by the cooldown window)\n")
	} else {
		b.WriteString("suppressed_since_last_email: 0\n")
	}
	b.WriteString(fmt.Sprintf("\nFurther DLT emails from this process are suppressed for %s.\n", cooldown))
	b.WriteString("Inspect the DLT topic in Kafka UI for full payload.\n")
	return b.String()
}

var (
	dltAlerterMu sync.RWMutex
	dltAlerter   DLTAlertNotifier
)

// SetDLTAlerter registers the process-wide DLT email alerter. Pass nil to disable.
func SetDLTAlerter(n DLTAlertNotifier) {
	dltAlerterMu.Lock()
	defer dltAlerterMu.Unlock()
	dltAlerter = n
}

func getDLTAlerter() DLTAlertNotifier {
	dltAlerterMu.RLock()
	defer dltAlerterMu.RUnlock()
	return dltAlerter
}

// maybeAlertDLT invokes the configured alerter synchronously (tests + internal).
func maybeAlertDLT(msg DLTMessage) {
	if n := getDLTAlerter(); n != nil {
		n.Notify(msg)
	}
}

// alertDLTAsync notifies in a goroutine so DLT publish / Ack paths stay unblocked.
func alertDLTAsync(msg DLTMessage) {
	n := getDLTAlerter()
	if n == nil {
		return
	}
	go n.Notify(msg)
}

// ResolveDLTAlertEmail returns DLT_ALERT_EMAIL or the default ops address.
func ResolveDLTAlertEmail() string {
	if v := strings.TrimSpace(os.Getenv("DLT_ALERT_EMAIL")); v != "" {
		return v
	}
	return DefaultDLTAlertEmail
}

// ResolveDLTAlertCooldown returns DLT_ALERT_COOLDOWN (Go duration) or default 5m.
func ResolveDLTAlertCooldown() time.Duration {
	if v := strings.TrimSpace(os.Getenv("DLT_ALERT_COOLDOWN")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return DefaultDLTAlertCooldown
}
