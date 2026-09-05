package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// claimRestartFrom is the operator-restartable set. It must stay aligned with
// domain.IsCanaryRestartableStatus so bulk/precise restart cannot plan a
// service that restartService then rejects (runAll Status #prog-error).
var claimRestartFrom = []Status{
	StatusHealthy,
	StatusFailed,
	StatusStopped,
	StatusPending,
	StatusRetrying,
	StatusSkipped,
}

func claimServiceRestart(store *StatusStore, name string) (Status, error) {
	if store == nil {
		return "", fmt.Errorf("status store is required")
	}
	for _, from := range claimRestartFrom {
		if store.CompareAndSwapStatus(name, from, StatusRestarting) {
			return from, nil
		}
	}
	current := store.Get(name)
	if current == nil {
		return "", fmt.Errorf("service %q not found", name)
	}
	return "", fmt.Errorf("service %q is %s, can only restart %s services",
		name, current.Status, formatClaimRestartFrom())
}

func formatClaimRestartFrom() string {
	parts := make([]string, len(claimRestartFrom))
	for i, s := range claimRestartFrom {
		parts[i] = string(s)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", or " + parts[len(parts)-1]
}

// shouldCanaryOverlap starts a peer beside a live process. Healthy is the
// intended canary path; Failed+live covers alive-but-unhealthy (health URL
// still answers from a sidecar/test listener). Retrying is mid-health-check
// and must not double-bind the port.
func shouldCanaryOverlap(previous Status, hasLiveProcess bool) bool {
	if !hasLiveProcess {
		return false
	}
	switch previous {
	case StatusHealthy, StatusFailed:
		return true
	default:
		return false
	}
}

func (r *Runner) startupStillOwnsProcess(name string, cmd *exec.Cmd) bool {
	if r == nil || cmd == nil || cmd.Process == nil {
		return false
	}
	return r.trackedCmd(name) == cmd
}

// startupAlreadyClaimed is true when startAndCheck must not CAS idle→Starting.
// Retrying is included only for drain-then-start (forceFreshStart) after a
// canary overlap that left the store on Retrying (false-healthy old listener).
func startupAlreadyClaimed(st *ServiceStatus, forceFreshStart bool) bool {
	if st == nil {
		return false
	}
	switch st.Status {
	case StatusStarting, StatusRestarting, StatusBuilding:
		return true
	case StatusRetrying:
		return forceFreshStart
	default:
		return false
	}
}

func reclaimRestartAfterCanaryFallback(store *StatusStore, name string) {
	if store == nil {
		return
	}
	for _, from := range []Status{StatusRetrying, StatusBuilding, StatusFailed} {
		if store.CompareAndSwapStatus(name, from, StatusRestarting) {
			return
		}
	}
}
