package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	"taskEvents/broker"
	"taskEvents/commandhttp"
	"taskEvents/config"
	"taskEvents/domain"
	"taskEvents/idempotency"
	"taskEvents/routing"
	"taskEvents/server"
	"tracelog"
)

// Run starts health HTTP + broker consume loop using Django HTTP dispatch.
func Run(domainName string, cfg config.Config, consumerCfg config.ConsumerConfig) {
	RunWithDelivery(domainName, cfg, consumerCfg, commandhttp.NewClient("", cfg.InternalSecret), nil)
}

// RunWithDelivery starts the consumer with a custom DomainCommandPort (e.g. notifications local delivery).
// keyFor overrides idempotency key derivation when non-nil.
func RunWithDelivery(domainName string, cfg config.Config, consumerCfg config.ConsumerConfig, commands domain.DomainCommandPort, keyFor func(domain.EventEnvelope) domain.IdempotencyKey) {
	serviceName := resolveObservabilityServiceName(domainName)
	tracelog.InitConsumer(serviceName)
	shutdownOtel, err := tracelog.InitOtel(context.Background(), serviceName)
	if err != nil {
		log.Printf("[%s] otel disabled: %v", serviceName, err)
	} else {
		defer func() { _ = shutdownOtel(context.Background()) }()
	}

	if cfg.Transport == domain.TransportMemory {
		log.Printf("[%s] transport=memory, exiting", serviceName)
		return
	}

	session := domain.ConsumerSession{
		DomainName:       domainName,
		Transport:        cfg.Transport,
		GroupID:          consumerCfg.GroupID,
		SubscribedEvents: consumerCfg.Events,
	}
	router := routing.DispatchRouter(domainName, consumerCfg.Events)
	keyFn := IdempotencyKeyFromEnvelope
	if keyFor != nil {
		keyFn = keyFor
	}
	dispatch := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: idempotency.NewMemoryStore(),
		KeyFor:      keyFn,
	}

	handle, err := broker.NewHandle(cfg, consumerCfg.GroupID, consumerCfg.Events)
	if err != nil {
		tracelog.Fatal(serviceName, "broker", err)
	}
	defer handle.Port.Close()
	conn := handle.Conn

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ops DLT report email (5m cooldown, cross-process via Redis); never blocks consume loop.
	ensureDLTEmailAlert(cfg)

	if err := broker.PingWithRetry(ctx, cfg); err != nil {
		tracelog.Fatal(serviceName, "broker ping", err)
	}
	conn.SetConnected(true)

	ch, err := handle.Port.Subscribe(ctx, consumerCfg.Events)
	if err != nil {
		tracelog.Fatal(serviceName, "subscribe", err)
	}

	addr := consumerCfg.Host + ":" + strconv.Itoa(consumerCfg.Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health/", server.NewHealthHandler(session.DomainName, session.Transport, session.SubscribedEvents))
	mux.HandleFunc("/api/health/ready", server.NewReadinessHandler(session.DomainName, session.Transport, session.SubscribedEvents, conn))
	log.Printf("[%s] health http://%s/api/health/ transport=%s", serviceName, addr, cfg.Transport)
	httpDone := server.ListenHealthReusePort(ctx, serviceName, addr, tracelog.Middleware(mux))

	// DLT config
	dltMaxRetries := config.DLTMaxRetries()
	kafkaBrokers := cfg.BootstrapServers

	go func() {
		for msg := range ch {
			corr := tracelog.CorrelationFromEnvelopeData(msg.Envelope.Data)
			msgCtx := tracelog.ContextWithCorrelation(ctx, corr)
			traceID := corr.TraceID
			msgCtx, endSpan := tracelog.StartSpan(msgCtx, "consume "+msg.Envelope.EventType, traceID,
				tracelog.String("messaging.event_type", msg.Envelope.EventType),
				tracelog.String("task_events.domain", domainName),
			)
			tracelog.LogEventConsume(msgCtx, "received", msg.Envelope.EventType, map[string]any{
				"domain": domainName,
			})
			if !session.AllowsEvent(msg.Envelope.EventType) {
				_ = handle.Port.Ack(msgCtx, msg)
				endSpan()
				continue
			}
			cmd, ok := router.CommandFor(msg.Envelope)
			if !ok {
				tracelog.EmitComponent("warn", "no route for event", "consumer", traceID, map[string]string{
					"event_type": msg.Envelope.EventType,
				})
				tracelog.LogEventConsume(msgCtx, "no_route", msg.Envelope.EventType, nil)
				_ = handle.Port.Ack(msgCtx, msg)
				endSpan()
				continue
			}
			out, err := dispatch.Handle(msgCtx, cmd)
			// DispatchPermanent must be checked first — when a handler
			// returns both an error and DispatchPermanent, the handler
			// already determined this event cannot succeed.  Send to DLT
			// and Ack so the consumer group offset advances.
			if out == domain.DispatchPermanent {
				tracelog.EmitComponent("error", "dispatch permanent fail", "consumer", traceID, map[string]string{
					"event_type": msg.Envelope.EventType,
				})
				tracelog.LogEventConsume(msgCtx, "dispatch_permanent_fail", msg.Envelope.EventType, map[string]any{
					"error": errString(err),
				})
				broker.PublishDeadLetterFromBrokerMessage(msgCtx, kafkaBrokers, msg, err, broker.FailureReasonPermanent, 1)
				_ = handle.Port.Ack(msgCtx, msg)
				endSpan()
				continue
			}
			if err != nil || out == domain.DispatchRetryable {
				// Extract durable attempt counter from payload (survives restarts).
				attempt := attemptFromEnvelope(msg.Envelope) + 1

				// Check retry exhaustion (durable — persisted in re-published payload).
				if attempt > dltMaxRetries {
					tracelog.EmitComponent("error", "dispatch retry exhausted", "consumer", traceID, map[string]string{
						"event_type":  msg.Envelope.EventType,
						"retry_count": strconv.Itoa(attempt),
					})
					tracelog.LogEventConsume(msgCtx, "dispatch_retry_exhausted", msg.Envelope.EventType, map[string]any{
						"error":       errString(err),
						"retry_count": attempt,
					})
					broker.PublishDeadLetterFromBrokerMessage(msgCtx, kafkaBrokers, msg, err, broker.FailureReasonRetryExhausted, attempt)
					_ = handle.Port.Ack(msgCtx, msg)
					endSpan()
					continue
				}

				wait := dispatchRetryWait(attempt)
				tracelog.EmitComponent("warn", "dispatch retryable", "consumer", traceID, map[string]string{
					"event_type":    msg.Envelope.EventType,
					"retry_count":   strconv.Itoa(attempt),
					"retry_wait_ms": strconv.FormatInt(wait.Milliseconds(), 10),
				})
				tracelog.LogEventConsume(msgCtx, "dispatch_retryable", msg.Envelope.EventType, map[string]any{
					"error":         errString(err),
					"retry_count":   attempt,
					"retry_wait_ms": wait.Milliseconds(),
				})
				endSpan()
				// Exponential backoff from durable attempt: 1s → 2s → 4s → 8s → 16s → 30s (max).
				// Must not use a fresh in-memory Backoff after Ack+republish (that collapsed 11 retries into ~20s).
				bo := broker.Backoff{Initial: dispatchRetryInitial, Max: dispatchRetryMax}
				_ = bo.WaitAttempt(msgCtx, attempt)

				// Re-publish with incremented attempt counter so retry count
				// survives process restarts.  Then Ack the original message
				// so the consumer group offset advances regardless.
				if repubErr := republishWithAttempt(msgCtx, kafkaBrokers, msg, attempt); repubErr != nil {
					broker.IncDLTRetryRepubFailed()
					log.Printf("[consumer] re-publish for retry failed event=%s attempt=%d key=%s: %v",
						msg.Envelope.EventType, attempt, msg.Envelope.Key, repubErr)
					// Message is not Ack'd; Kafka will redeliver after the consumer restarts
					// or the reader commits the previous offset only.
				} else {
					_ = handle.Port.Ack(msgCtx, msg)
				}
				continue
			}
			tracelog.EmitComponent("info", "dispatch ok", "consumer", traceID, map[string]string{
				"event_type": msg.Envelope.EventType,
			})
			tracelog.LogEventConsume(msgCtx, "dispatch_ok", msg.Envelope.EventType, nil)
			if err := handle.Port.Ack(msgCtx, msg); err != nil {
				tracelog.EmitComponent("warn", "ack failed", "consumer", traceID, map[string]string{
					"error": err.Error(),
				})
			} else {
				tracelog.LogEventConsume(msgCtx, "acked", msg.Envelope.EventType, nil)
			}
			endSpan()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	server.WaitHealthShutdown(httpDone)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// attemptFromEnvelope reads the durable retry counter from the event payload.
// Returns 0 when no attempt field is present (first dispatch attempt).
// This survives process restarts because the counter is embedded in the
// Kafka message payload and re-published on each retry.
func attemptFromEnvelope(env domain.EventEnvelope) int {
	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return 0
	}
	v, ok := data["_retry_attempt"]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// republishWithAttempt re-publishes the event to its original topic with an
// incremented _retry_attempt counter. The original message must be Ack'd
// separately after this call succeeds.
func republishWithAttempt(ctx context.Context, brokers string, msg domain.BrokerMessage, attempt int) error {
	// Parse original payload
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Envelope.Data, &data); err != nil {
		return fmt.Errorf("republish: unmarshal data: %w", err)
	}
	data["_retry_attempt"] = attempt

	// Build the same envelope structure the publisher uses
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": msg.Envelope.EventType,
		"data":       data,
	})
	if err != nil {
		return fmt.Errorf("republish: marshal: %w", err)
	}

	topic, _ := config.EventTopic(msg.Envelope.EventType)
	if topic == "" {
		topic = strings.ToLower(strings.ReplaceAll(msg.Envelope.EventType, "_", "-"))
	}

	brk := strings.TrimSpace(brokers)
	if brk == "" {
		brk = "localhost:9093"
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brk),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer w.Close()

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	kmsg := kafka.Message{Value: payload}
	if msg.Envelope.Key != "" {
		kmsg.Key = []byte(msg.Envelope.Key)
	}
	return w.WriteMessages(writeCtx, kmsg)
}

// resolveObservabilityServiceName returns the slog/OTel service name for Grafana/Promtail.
// eventbin already passes BinaryName/GroupID (e.g. task-events-cloud-server-stopped-1-process-server-stop);
// do not prepend task-events- again. Prefer OTEL_SERVICE_NAME when set (runAll injects the canonical job name).
func resolveObservabilityServiceName(domainName string) string {
	if env := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); env != "" {
		return env
	}
	name := strings.TrimSpace(domainName)
	if name == "" {
		return "task-events"
	}
	if strings.HasPrefix(name, "task-events-") || name == "task-events" {
		return name
	}
	return "task-events-" + name
}
