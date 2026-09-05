package domain

import (
	"context"
	"time"
)

// LokiQueryClient queries Loki for trace-correlated log lines.
type LokiQueryClient interface {
	Ready(ctx context.Context) error
	CountLogsByJob(ctx context.Context, traceID string, since time.Duration) (map[string]int, error)
}
