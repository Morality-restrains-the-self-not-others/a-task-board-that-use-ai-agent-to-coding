package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
)

// RedisStreamBroker consumes domain-events from a single Redis stream.
type RedisStreamBroker struct {
	client    *redis.Client
	stream    string
	group     string
	consumer  string
	connState *ConnectionState
	retryLog  *RateLimitedLogger
}

// NewRedisStreamBroker creates a Redis stream consumer group reader (group starts at "0").
func NewRedisStreamBroker(host string, port, db int, stream, group string, connState *ConnectionState) *RedisStreamBroker {
	return NewRedisStreamBrokerAt(host, port, db, stream, group, "0", connState)
}

// NewRedisStreamBrokerAt creates a group starting at startID ("0" = backlog, "$" = tail only).
func NewRedisStreamBrokerAt(host string, port, db int, stream, group, startID string, connState *ConnectionState) *RedisStreamBroker {
	if port == 0 {
		port = 6379
	}
	if stream == "" {
		stream = "domain-events:all"
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: db})
	consumer := group + "-instance"
	if err := rdb.XGroupCreateMkStream(context.Background(), stream, group, startID).Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			fmt.Printf("[redis-broker] XGroupCreateMkStream %s/%s: %v\n", stream, group, err)
		}
	}
	return &RedisStreamBroker{
		client:    rdb,
		stream:    stream,
		group:     group,
		consumer:  consumer,
		connState: connState,
		retryLog:  NewRateLimitedLogger(30 * time.Second),
	}
}

// Subscribe reads messages for allowed event types.
// Also runs a periodic pending-message recovery goroutine so that retryable
// dispatch failures (which leave messages in the PEL) are eventually retried.
func (r *RedisStreamBroker) Subscribe(ctx context.Context, events []string) (<-chan domain.BrokerMessage, error) {
	allowed := make(map[string]struct{}, len(events))
	for _, e := range events {
		allowed[e] = struct{}{}
	}
	out := make(chan domain.BrokerMessage, 32)

	// Periodically claim pending messages older than 60s and redeliver them.
	go r.recoverPending(ctx, allowed, out)

	go func() {
		defer close(out)
		backoff := DefaultRuntimeBackoff()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    r.group,
				Consumer: r.consumer,
				Streams:  []string{r.stream, ">"},
				Count:    10,
				Block:    2 * time.Second,
			}).Result()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if err == redis.Nil {
					r.connState.SetConnected(true)
					backoff.Reset()
					continue
				}
				r.connState.SetConnected(false)
				r.retryLog.Warnf("[redis-broker] xreadgroup failed: %v", err)
				// NOGROUP 表示 stream 或消费者组丢失（如 Redis 重启）；尝试重建。
				if strings.Contains(err.Error(), "NOGROUP") {
					_ = r.client.XGroupCreateMkStream(ctx, r.stream, r.group, "0").Err()
				}
				if waitErr := backoff.Wait(ctx); waitErr != nil {
					return
				}
				continue
			}
			r.connState.SetConnected(true)
			backoff.Reset()
			for _, stream := range streams {
				for _, msg := range stream.Messages {
					raw, ok := msg.Values["payload"].(string)
					if !ok {
						b, ok := msg.Values["payload"].([]byte)
						if ok {
							raw = string(b)
						}
					}
					if raw == "" {
						_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
						continue
					}
					value := []byte(raw)
					key := ""
					if k, ok := msg.Values["key"].(string); ok {
						key = k
					}
					env, err := ParseEnvelope(value, []byte(key))
					if err != nil {
						_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
						continue
					}
					if _, ok := allowed[env.EventType]; len(allowed) > 0 && !ok {
						_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
						continue
					}
					id := msg.ID
					bm := domain.BrokerMessage{
						Envelope: env,
						Cursor:   domain.BrokerCursor{Raw: id},
						AckFunc: func(ctx context.Context) error {
							return r.client.XAck(ctx, r.stream, r.group, id).Err()
						},
					}
					select {
					case out <- bm:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, nil
}

// Ack commits via AckFunc.
func (r *RedisStreamBroker) Ack(ctx context.Context, msg domain.BrokerMessage) error {
	return domain.AckMessage(ctx, msg)
}

// Close closes the redis client.
func (r *RedisStreamBroker) Close() error {
	return r.client.Close()
}

// recoverPending periodically claims pending messages and redelivers them so
// that retryable dispatch failures are retried rather than stranded in the PEL.
func (r *RedisStreamBroker) recoverPending(ctx context.Context, allowed map[string]struct{}, out chan<- domain.BrokerMessage) {
	const (
		recoverInterval = 30 * time.Second
		minIdleTime     = 60 * time.Second
	)
	ticker := time.NewTicker(recoverInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		startID := "0-0"
		for {
			claimed, nextStart, err := r.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
				Stream:   r.stream,
				Group:    r.group,
				Consumer: r.consumer,
				MinIdle:  minIdleTime,
				Start:    startID,
				Count:    10,
			}).Result()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				break
			}
			if len(claimed) == 0 {
				break
			}
			for _, msg := range claimed {
				raw, ok := msg.Values["payload"].(string)
				if !ok {
					if b, ok := msg.Values["payload"].([]byte); ok {
						raw = string(b)
					}
				}
				if raw == "" {
					_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
					continue
				}
				env, err := ParseEnvelope([]byte(raw), nil)
				if err != nil {
					_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
					continue
				}
				if _, ok := allowed[env.EventType]; len(allowed) > 0 && !ok {
					_ = r.client.XAck(ctx, r.stream, r.group, msg.ID)
					continue
				}
				id := msg.ID
				bm := domain.BrokerMessage{
					Envelope: env,
					Cursor:   domain.BrokerCursor{Raw: id},
					AckFunc: func(ctx context.Context) error {
						return r.client.XAck(ctx, r.stream, r.group, id).Err()
					},
				}
				select {
				case out <- bm:
				case <-ctx.Done():
					return
				}
			}
			startID = nextStart
		}
	}
}

// PublishEnvelope is used by tests to XADD an event.
func PublishEnvelope(ctx context.Context, rdb *redis.Client, stream string, eventType string, data interface{}, key string) error {
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	fields := map[string]interface{}{"payload": string(payload)}
	if key != "" {
		fields["key"] = key
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: fields}).Err()
}

// NormalizeStreamKey ensures stream ends with :all for single-stream mode.
func NormalizeStreamKey(prefix string) string {
	prefix = strings.TrimRight(prefix, ":")
	if prefix == "" {
		return "domain-events:all"
	}
	return prefix + ":all"
}
