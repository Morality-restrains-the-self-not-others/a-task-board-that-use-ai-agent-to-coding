package broker

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"taskEvents/config"
	"taskEvents/domain"
	"tracelog"
)

// DLT metrics counters (atomic for lock-free reads from health endpoint).
var (
	dltPublished  atomic.Int64
	dltFailed     atomic.Int64
	dltRetryRepub atomic.Int64
)

// DLTCounters returns the current DLT metric counters.
func DLTCounters() map[string]int64 {
	return map[string]int64{
		"dlt_published":       dltPublished.Load(),
		"dlt_publish_failed":  dltFailed.Load(),
		"dlt_retry_republish": dltRetryRepub.Load(),
	}
}

// DLTMessage is the envelope written to a dead-letter topic.
type DLTMessage struct {
	OriginalEventType string                 `json:"original_event_type"`
	OriginalTopic     string                 `json:"original_topic"`
	OriginalData      map[string]interface{} `json:"original_data"`
	OriginalKey       string                 `json:"original_key"`
	Error             string                 `json:"error"`
	DeadLetteredAt    string                 `json:"dead_lettered_at"`
	FailureReason     string                 `json:"failure_reason"`
	RetryCount        int                    `json:"retry_count"`
	// TraceID / OtelTraceID correlate DLT payloads and alert emails with Loki logs.
	TraceID     string `json:"trace_id,omitempty"`
	OtelTraceID string `json:"otel_trace_id,omitempty"`
}

const (
	// FailureReasonPermanent is used when the handler returned DispatchPermanent.
	FailureReasonPermanent = "permanent"
	// FailureReasonRetryExhausted is used when max retries were exhausted.
	FailureReasonRetryExhausted = "retry_exhausted"
)

// PublishDeadLetter sends an unprocessable envelope to the dead-letter topic
// for offline inspection.  The call is best-effort: errors are logged, never
// returned to the caller, because DLT publish must not block the consumer from
// advancing its offset.
func PublishDeadLetter(ctx context.Context, brokers string, env domain.EventEnvelope, dispatchErr error, reason string, retries int) {
	originalTopic, _ := config.EventTopic(env.EventType)
	if originalTopic == "" {
		originalTopic = strings.ToLower(strings.ReplaceAll(env.EventType, "_", "-"))
	}
	dltTopic := config.DeadLetterTopic(originalTopic)

	msg := buildDLTMessage(ctx, env, dispatchErr, reason, retries)

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[dlt] marshal error for %s (topic=%s): %v", env.EventType, dltTopic, err)
		dltFailed.Add(1)
		return
	}

	brk := strings.TrimSpace(brokers)
	if brk == "" {
		brk = "localhost:9093"
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brk),
		Topic:        dltTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer w.Close()

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	kmsg := kafka.Message{Value: payload}
	if env.Key != "" {
		kmsg.Key = []byte(env.Key)
	}
	if err := w.WriteMessages(writeCtx, kmsg); err != nil {
		log.Printf("[dlt] publish error for %s → topic=%s: %v", env.EventType, dltTopic, err)
		dltFailed.Add(1)
		return
	}
	dltPublished.Add(1)
	log.Printf("[dlt] dead-lettered %s → %s reason=%s retries=%d key=%s",
		env.EventType, dltTopic, reason, retries, string(env.Key))
	// Ops email is best-effort and must not delay Ack / offset advance.
	alertDLTAsync(msg)
}

// PublishDeadLetterFromBrokerMessage is a convenience wrapper that extracts
// the envelope from a domain.BrokerMessage.
func PublishDeadLetterFromBrokerMessage(ctx context.Context, brokers string, msg domain.BrokerMessage, dispatchErr error, reason string, retries int) {
	PublishDeadLetter(ctx, brokers, msg.Envelope, dispatchErr, reason, retries)
}

// buildDLTMessage constructs the DLT envelope, preferring ctx trace_id then payload.
func buildDLTMessage(ctx context.Context, env domain.EventEnvelope, dispatchErr error, reason string, retries int) DLTMessage {
	originalTopic, _ := config.EventTopic(env.EventType)
	if originalTopic == "" {
		originalTopic = strings.ToLower(strings.ReplaceAll(env.EventType, "_", "-"))
	}

	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		data = map[string]interface{}{
			"_raw": string(env.Data),
		}
	}

	errStr := ""
	if dispatchErr != nil {
		errStr = dispatchErr.Error()
	}

	traceID := strings.TrimSpace(tracelog.TraceIDFromContext(ctx))
	if traceID == "" {
		traceID = tracelog.TraceIDFromEnvelopeData(env.Data)
	}
	otelID := ""
	if traceID != "" {
		otelID = tracelog.OtelTraceIDHex(traceID)
	}

	return DLTMessage{
		OriginalEventType: env.EventType,
		OriginalTopic:     originalTopic,
		OriginalData:      data,
		OriginalKey:       string(env.Key),
		Error:             errStr,
		DeadLetteredAt:    time.Now().UTC().Format(time.RFC3339),
		FailureReason:     reason,
		RetryCount:        retries,
		TraceID:           traceID,
		OtelTraceID:       otelID,
	}
}

// IncDLTRetryRepubFailed increments the retry-republish failure counter.
// Called by the consumer runner when re-publishing a retryable event fails.
func IncDLTRetryRepubFailed() {
	dltRetryRepub.Add(1)
}
