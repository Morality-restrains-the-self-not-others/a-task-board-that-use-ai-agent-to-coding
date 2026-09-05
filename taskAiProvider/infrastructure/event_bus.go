package infrastructure

import (
	"context"
	"log"
	"sync"
)

// LogEventBus 将领域事件写入结构化日志（taskAiProvider 无 Kafka 配置时的投递实现）。
type LogEventBus struct{}

func (LogEventBus) Publish(_ context.Context, name string, payload map[string]any) error {
	log.Printf("[taskAiProvider] event=%s payload=%s", name, MustJSON(payload))
	return nil
}

// SpyEventBus 单测记录已发布事件名。
type SpyEventBus struct {
	mu    sync.Mutex
	Names []string
}

func (s *SpyEventBus) Publish(_ context.Context, name string, _ map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Names = append(s.Names, name)
	return nil
}
