package domain

import "time"

// LogCollectionStats captures per-service log production metrics.
type LogCollectionStats struct {
	ServiceName string    `json:"service_name"`
	TotalLines  int64     `json:"total_lines"`
	TotalBytes  int64     `json:"total_bytes"`
	FirstLogAt  time.Time `json:"first_log_at,omitempty"`
	LastLogAt   time.Time `json:"last_log_at,omitempty"`
}

// ServiceLogStatsProvider exposes log collection metrics for services.
type ServiceLogStatsProvider interface {
	Stats(service string) LogCollectionStats
	AllStats() map[string]LogCollectionStats
}
