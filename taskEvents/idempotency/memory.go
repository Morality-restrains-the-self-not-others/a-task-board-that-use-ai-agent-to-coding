package idempotency

import (
	"sync"

	"taskEvents/domain"
)

// MemoryStore is an in-process idempotency cache.
type MemoryStore struct {
	mu   sync.Mutex
	seen map[domain.IdempotencyKey]struct{}
}

// NewMemoryStore creates an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{seen: make(map[domain.IdempotencyKey]struct{})}
}

// Seen reports whether the key was processed.
func (m *MemoryStore) Seen(key domain.IdempotencyKey) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.seen[key]
	return ok
}

// Mark records a processed key.
func (m *MemoryStore) Mark(key domain.IdempotencyKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seen[key] = struct{}{}
}
