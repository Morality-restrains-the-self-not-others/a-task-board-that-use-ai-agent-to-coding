package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// runAllExitReason records the trigger behind a graceful runAll exit so that a
// silent shutdown (e.g. a periodic SIGTERM that is not shutdown-self) can be
// attributed from the logs alone (OPT-20260813-007).
type runAllExitReason struct {
	mu     sync.Mutex
	source string // "signal" | "shutdown-self" | "" (unknown)
	detail string // signal name or request path
}

func newRunAllExitReason() *runAllExitReason { return &runAllExitReason{} }

// record stores the first trigger only; later triggers are ignored so the
// initiating cause is never overwritten by the shutdown cascade.
func (r *runAllExitReason) record(source, detail string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.source == "" {
		r.source = source
		r.detail = detail
	}
}

func (r *runAllExitReason) get() (source, detail string) {
	if r == nil {
		return "", ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.source, r.detail
}

// summarizeServiceLifecycle renders a compact per-status snapshot of managed
// services at exit time, so a graceful shutdown can be correlated with which
// services were healthy/stopped/failed when the run loop ended.
func summarizeServiceLifecycle(store *StatusStore) string {
	if store == nil {
		return "store=nil"
	}
	counts := make(map[Status]int)
	var nonHealthy []string
	for _, s := range store.All() {
		counts[s.Status]++
		if s.Status != StatusHealthy && s.Status != StatusStopped {
			nonHealthy = append(nonHealthy, fmt.Sprintf("%s:%s", s.Name, s.Status))
		}
	}
	sort.Strings(nonHealthy)
	var parts []string
	for _, st := range []Status{StatusHealthy, StatusStarting, StatusRetrying, StatusBuilding, StatusRestarting, StatusFailed, StatusSkipped, StatusStopped, StatusPending} {
		if n := counts[st]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", st, n))
		}
	}
	if len(nonHealthy) > 0 {
		parts = append(parts, "non_healthy="+strings.Join(nonHealthy, ","))
	}
	if len(parts) == 0 {
		return "empty"
	}
	return strings.Join(parts, " ")
}
