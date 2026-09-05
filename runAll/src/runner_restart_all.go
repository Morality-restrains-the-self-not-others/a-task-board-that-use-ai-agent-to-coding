package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// RestartAll rolling-canary restarts every start-all service (ADR-0058).
func (r *Runner) RestartAll(ctx context.Context) error {
	return r.RestartAllWithActor(ctx, defaultOwnershipSessionID)
}

// RestartAllWithActor canary-restarts services in start-all topology order.
func (r *Runner) RestartAllWithActor(ctx context.Context, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	if r == nil {
		return fmt.Errorf("runner is required")
	}
	if msg := r.bulkStartMigrationBlock(ctx); msg != "" {
		return fmt.Errorf("restart-all blocked: %s", msg)
	}

	startedHere := false
	if !r.IsRestartAllActive() {
		if !r.TryBeginRestartAll() {
			return fmt.Errorf("restart-all already in progress")
		}
		startedHere = true
	}
	if startedHere {
		defer r.endRestartAll()
	}

	r.setRestartAllPhase("canary")
	if r.GetActiveRestartAllRunID() == "" {
		_ = r.SetActiveRestartAllRunID()
	}
	plan, err := r.cascadeOrchestration().PlanCanaryRestartAll()
	if err != nil {
		return err
	}
	runID := r.GetActiveRestartAllRunID()
	publish := func(ev StartAllProgressEvent) {
		if runID != "" && r.progressBroadcaster != nil {
			ev.Operation = "restart"
			r.progressBroadcaster.Publish(runID, ev)
		}
	}
	total := len(plan.OrderedNames)
	if total == 0 {
		publish(StartAllProgressEvent{Total: 0, Done: true, Phase: "done", Operation: "restart"})
		return nil
	}
	var allErrors []string
	for i, svcName := range plan.OrderedNames {
		if err := ctx.Err(); err != nil {
			publish(StartAllProgressEvent{
				Total: total, Started: i, Failed: len(allErrors), Remaining: total - i,
				Done: true, Phase: progressPhaseForContextErr(ctx), Operation: "restart", Error: err.Error(),
			})
			return err
		}
		publish(StartAllProgressEvent{
			Total: total, Started: i, Failed: len(allErrors), Remaining: total - i,
			Current: svcName, Phase: "progress", Operation: "restart",
		})
		if rerr := r.RestartServiceWithActor(ctx, svcName, actor); rerr != nil {
			log.Printf("[cascade] restart-all canary %s: %v", svcName, rerr)
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", svcName, rerr))
		}
	}
	errMsg := ""
	if len(allErrors) > 0 {
		errMsg = strings.Join(allErrors, "; ")
	}
	publish(StartAllProgressEvent{
		Total: total, Started: total, Failed: len(allErrors), Remaining: 0,
		Done: true, Phase: "done", Operation: "restart", Error: errMsg, Errors: allErrors,
	})
	if len(allErrors) > 0 {
		return fmt.Errorf("restart-all: %d/%d services failed: %s", len(allErrors), total, errMsg)
	}
	return nil
}

func (r *Runner) TryBeginRestartAll() bool {
	if r == nil {
		return false
	}
	if !r.TryBeginBulk("restart-all") {
		return false
	}
	r.activeRestartAllMu.Lock()
	defer r.activeRestartAllMu.Unlock()
	if r.activeRestartAll {
		r.EndBulk("restart-all")
		return false
	}
	r.activeRestartAll = true
	r.restartAllPhase = "canary"
	return true
}

func (r *Runner) endRestartAll() {
	r.activeRestartAllMu.Lock()
	r.activeRestartAll = false
	r.restartAllPhase = ""
	r.activeRestartAllRunID = ""
	r.activeRestartAllMu.Unlock()
	r.EndBulk("restart-all")
}

func (r *Runner) setRestartAllPhase(phase string) {
	r.activeRestartAllMu.Lock()
	r.restartAllPhase = phase
	r.activeRestartAllMu.Unlock()
}

func (r *Runner) IsRestartAllActive() bool {
	if r == nil {
		return false
	}
	r.activeRestartAllMu.RLock()
	defer r.activeRestartAllMu.RUnlock()
	return r.activeRestartAll
}

func (r *Runner) RestartAllPhase() string {
	if r == nil {
		return ""
	}
	r.activeRestartAllMu.RLock()
	defer r.activeRestartAllMu.RUnlock()
	return r.restartAllPhase
}

func (r *Runner) GetActiveRestartAllRunID() string {
	if r == nil {
		return ""
	}
	r.activeRestartAllMu.RLock()
	defer r.activeRestartAllMu.RUnlock()
	return r.activeRestartAllRunID
}

func (r *Runner) SetActiveRestartAllRunID() string {
	runID := fmt.Sprintf("restart-all-%d", time.Now().UnixNano())
	r.activeRestartAllMu.Lock()
	r.activeRestartAllRunID = runID
	r.activeRestartAllMu.Unlock()
	return runID
}

func (r *Runner) SubscribeRestartAllProgress() (chan StartAllProgressEvent, string) {
	runID := r.GetActiveRestartAllRunID()
	if runID == "" {
		return nil, ""
	}
	ch := r.progressBroadcaster.Subscribe(runID)
	return ch, runID
}
