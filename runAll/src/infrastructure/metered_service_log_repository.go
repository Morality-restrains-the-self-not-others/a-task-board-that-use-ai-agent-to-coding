package infrastructure

import (
	"sync"
	"time"

	"runAll/src/domain"
)

// MeteredServiceLogRepository wraps a ServiceLogRepository and tracks per-service
// log collection statistics (line count, byte count, first/last timestamps).
// It implements both domain.ServiceLogRepository and domain.ServiceLogStatsProvider.
type MeteredServiceLogRepository struct {
	inner domain.ServiceLogRepository
	mu    sync.RWMutex
	stats map[string]*meteredStats
}

type meteredStats struct {
	totalLines int64
	totalBytes int64
	firstLogAt time.Time
	lastLogAt  time.Time
}

// NewMeteredServiceLogRepository creates a stats-tracking wrapper.
// Pass nil to create a standalone stats tracker backed by an in-memory repository.
func NewMeteredServiceLogRepository(inner domain.ServiceLogRepository) *MeteredServiceLogRepository {
	if inner == nil {
		inner = NewInMemoryServiceLogRepository(DefaultServiceLogCapacity)
	}
	return &MeteredServiceLogRepository{
		inner: inner,
		stats: make(map[string]*meteredStats),
	}
}

func (r *MeteredServiceLogRepository) getOrCreateStatsLocked(service string) *meteredStats {
	s, ok := r.stats[service]
	if !ok {
		s = &meteredStats{}
		r.stats[service] = s
	}
	return s
}

// Append implements domain.ServiceLogRepository.
func (r *MeteredServiceLogRepository) Append(service string, entry domain.LogEntry) {
	r.inner.Append(service, entry)

	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.getOrCreateStatsLocked(service)
	s.totalLines++
	s.totalBytes += int64(len(entry.Message))
	now := entry.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	if s.firstLogAt.IsZero() {
		s.firstLogAt = now
	}
	s.lastLogAt = now
}

// Tail implements domain.ServiceLogRepository.
func (r *MeteredServiceLogRepository) Tail(service string, lines int) []domain.LogEntry {
	return r.inner.Tail(service, lines)
}

// Clear implements domain.ServiceLogRepository.
func (r *MeteredServiceLogRepository) Clear(service string) {
	r.inner.Clear(service)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.stats, service)
}

// Stats returns log collection statistics for a single service.
func (r *MeteredServiceLogRepository) Stats(service string) domain.LogCollectionStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s := r.stats[service]
	if s == nil {
		return domain.LogCollectionStats{ServiceName: service}
	}
	return domain.LogCollectionStats{
		ServiceName: service,
		TotalLines:  s.totalLines,
		TotalBytes:  s.totalBytes,
		FirstLogAt:  s.firstLogAt,
		LastLogAt:   s.lastLogAt,
	}
}

// AllStats returns log collection statistics for all services.
func (r *MeteredServiceLogRepository) AllStats() map[string]domain.LogCollectionStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]domain.LogCollectionStats, len(r.stats))
	for name, s := range r.stats {
		result[name] = domain.LogCollectionStats{
			ServiceName: name,
			TotalLines:  s.totalLines,
			TotalBytes:  s.totalBytes,
			FirstLogAt:  s.firstLogAt,
			LastLogAt:   s.lastLogAt,
		}
	}
	return result
}

// Ensure it implements both interfaces.
var (
	_ domain.ServiceLogRepository    = (*MeteredServiceLogRepository)(nil)
	_ domain.ServiceLogStatsProvider = (*MeteredServiceLogRepository)(nil)
)
