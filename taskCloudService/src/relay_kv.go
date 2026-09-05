package main

import (
	"context"
	"sync"
	"time"
)

// relayStringKV is the minimal Redis string API used by relay startup sessions.
type relayStringKV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type memoryRelayKV struct {
	mu   sync.Mutex
	data map[string]string
}

func newMemoryRelayKV() *memoryRelayKV {
	return &memoryRelayKV{data: map[string]string{}}
}

func (m *memoryRelayKV) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", errRelayKVNil
	}
	return v, nil
}

func (m *memoryRelayKV) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *memoryRelayKV) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

// errRelayKVNil mirrors redis.Nil for missing keys.
var errRelayKVNil = errSentinel("relay kv: nil")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func isRelayKVNil(err error) bool {
	return err == errRelayKVNil
}
