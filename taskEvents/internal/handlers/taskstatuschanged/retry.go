package taskstatuschanged

import (
	"context"
	"fmt"
	"log"

	"taskEvents/domain"
	"tracelog"
)

// RetryDecider encapsulates the decision to retry when a required entity
// (CSC row, server_url, instance_id) is not yet available.
//
// Since event data is immutable between Kafka/Redis retries, the decider
// does NOT track retry count within a single message lifecycle. Instead it
// relies on the broker's natural backoff (Redis XAutoClaim 60s idle reclaim)
// and provides clear logging for observability.
//
// For retry-count-aware patterns, see taskgracefulshutdownawait.Handler
// which re-publishes events with an incremented attempt counter.
type RetryDecider struct {
	Stage  string // tracelog consume_stage value (e.g. "no_config_retry")
	Entity string // human-readable entity being waited for (e.g. "csc row")
}

// ShouldRetry logs the decision and returns a DispatchRetryable outcome+error.
func (d RetryDecider) ShouldRetry(ctx context.Context, taskID, terminalKind string) (domain.DispatchOutcome, error) {
	log.Printf("[task_status_changed] %s not found task_id=%s terminal_kind=%s — will retry",
		d.Entity, taskID, terminalKind)
	tracelog.LogEventConsume(ctx, d.Stage, "TASK_STATUS_CHANGED", map[string]any{
		"task_id":       taskID,
		"terminal_kind": terminalKind,
	})
	return domain.DispatchRetryable, fmt.Errorf("%s not yet available for task %s", d.Entity, taskID)
}

// cscNotFoundDecider is a package-level instance for the "CSC row not found" case.
var cscNotFoundDecider = RetryDecider{Stage: "no_config_retry", Entity: "csc row"}

// noResourceDecider is a package-level instance for the "no running resource" case.
var noResourceDecider = RetryDecider{Stage: "no_running_resource_retry", Entity: "running resource"}
