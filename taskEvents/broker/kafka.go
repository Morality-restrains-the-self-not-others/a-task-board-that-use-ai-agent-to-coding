package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"taskEvents/domain"
)

// KafkaBroker implements domain.EventBrokerPort using kafka-go.
type KafkaBroker struct {
	brokers   []string
	groupID   string
	readers   []*kafka.Reader
	mu        sync.Mutex
	connState *ConnectionState
	retryLog  *RateLimitedLogger
}

// NewKafkaBroker creates a consumer group reader per topic.
func NewKafkaBroker(bootstrapServers, groupID string, topics []string, connState *ConnectionState) *KafkaBroker {
	brokers := []string{bootstrapServers}
	if bootstrapServers == "" {
		brokers = []string{"localhost:9093"}
	}
	kb := &KafkaBroker{
		brokers:   brokers,
		groupID:   groupID,
		connState: connState,
		retryLog:  NewRateLimitedLogger(30 * time.Second),
	}
	for _, topic := range topics {
		kb.readers = append(kb.readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			Topic:       topic,
			MinBytes:    1,
			MaxBytes:    10e6,
			StartOffset: kafka.FirstOffset,
		}))
	}
	return kb
}

// Subscribe merges messages from all topic readers into one channel.
func (k *KafkaBroker) Subscribe(ctx context.Context, topics []string) (<-chan domain.BrokerMessage, error) {
	_ = topics
	out := make(chan domain.BrokerMessage, 32)
	var wg sync.WaitGroup
	for _, r := range k.readers {
		reader := r
		wg.Add(1)
		// All readers share connState; with one topic per intent consumer this is sufficient.
		go func() {
			defer wg.Done()
			backoff := DefaultRuntimeBackoff()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				msg, err := reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					k.connState.SetConnected(false)
					k.retryLog.Warnf("[kafka-broker] fetch failed: %v", err)
					if waitErr := backoff.Wait(ctx); waitErr != nil {
						return
					}
					continue
				}
				k.connState.SetConnected(true)
				backoff.Reset()
				env, err := ParseEnvelope(msg.Value, msg.Key)
				if err != nil {
					_ = reader.CommitMessages(ctx, msg)
					continue
				}
				kmsg := msg
				kreader := reader
				cursor := domain.BrokerCursor{
					Raw: fmt.Sprintf("%s:%d:%d", kmsg.Topic, kmsg.Partition, kmsg.Offset),
				}
				bm := domain.BrokerMessage{
					Envelope: env,
					Cursor:   cursor,
					AckFunc: func(ctx context.Context) error {
						return kreader.CommitMessages(ctx, kmsg)
					},
				}
				select {
				case out <- bm:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out, nil
}

// Ack commits the kafka message when attached.
func (k *KafkaBroker) Ack(ctx context.Context, msg domain.BrokerMessage) error {
	return domain.AckMessage(ctx, msg)
}

// Close closes all readers.
func (k *KafkaBroker) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	var first error
	for _, r := range k.readers {
		if err := r.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
