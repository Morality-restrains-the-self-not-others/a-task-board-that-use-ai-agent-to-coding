package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func registerRestartAllHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/restart-all", func(w http.ResponseWriter, r *http.Request) {
		handleRestartAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/restart-all/cancel", func(w http.ResponseWriter, r *http.Request) {
		handleCancelRestartAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/restart-all/progress", func(w http.ResponseWriter, r *http.Request) {
		handleRestartAllProgressSSE(w, r, runner)
	})
}

func handleRestartAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	sessionID, err := readRequiredStringField(payload, "session_id")
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	if msg := runner.bulkStartMigrationBlock(r.Context()); msg != "" {
		writeJSONErrorWithStatus(w, http.StatusBadRequest, "restart-all blocked: "+msg)
		return
	}
	if !runner.TryBeginRestartAll() {
		writeJSONConflictWithBulkProgress(w, runner, "restart-all already in progress")
		return
	}
	// Pre-generate canary runID so SSE clients can subscribe before the goroutine starts.
	runID := runner.SetActiveRestartAllRunID()
	runCancellableLifecycleActionAsync("restart-all", func(ctx context.Context) error {
		defer runner.endRestartAll()
		return runner.RestartAllWithActor(ctx, sessionID)
	}, "restart-all", "all")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handleCancelRestartAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cancelLifecycleAction("restart-all") {
		writeJSON(w, map[string]string{"status": "cancelled"})
		return
	}
	if runner != nil && runner.ClearOrphanedBulkProgress("restart-all") {
		writeJSON(w, map[string]string{"status": "cleared"})
		return
	}
	writeJSONErrorWithStatus(w, http.StatusNotFound, "no active restart-all operation")
}

func handleRestartAllProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if runner == nil {
		writeJSONErrorWithStatus(w, http.StatusServiceUnavailable, "runner not available")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()

	_ = streamBulkProgressPhase(ctx, w, flusher, runner, "canary", runner.SubscribeRestartAllProgress)
}

// streamBulkProgressPhase subscribes to a bulk progress channel and writes SSE
// until a done event. Returns false if the client disconnected.
func streamBulkProgressPhase(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	runner *Runner,
	phase string,
	subscribe func() (chan StartAllProgressEvent, string),
) bool {
	var ch chan StartAllProgressEvent
	var runID string
	for i := 0; i < 20; i++ {
		ch, runID = subscribe()
		if ch != nil {
			break
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(250 * time.Millisecond):
		}
	}
	if ch == nil {
		log.Printf("[ui] restart-all progress: no active %s run", phase)
		return true
	}
	defer runner.GetProgressBroadcaster().Unsubscribe(runID, ch)

	for {
		select {
		case <-ctx.Done():
			return false
		case ev, ok := <-ch:
			if !ok {
				return true
			}
			fmt.Fprint(w, ev.ToSSE())
			flusher.Flush()
			if ev.Done {
				return true
			}
		}
	}
}
