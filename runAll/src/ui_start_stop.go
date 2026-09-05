package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func registerStartStopHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/start-all", func(w http.ResponseWriter, r *http.Request) {
		handleStartAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/start-all/cancel", func(w http.ResponseWriter, r *http.Request) {
		handleCancelStartAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/start-all/progress", func(w http.ResponseWriter, r *http.Request) {
		handleStartAllProgressSSE(w, r, runner)
	})
	mux.HandleFunc("/api/stop-all", func(w http.ResponseWriter, r *http.Request) {
		handleStopAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/stop-all/cancel", func(w http.ResponseWriter, r *http.Request) {
		handleCancelStopAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/stop-all/progress", func(w http.ResponseWriter, r *http.Request) {
		handleStopAllProgressSSE(w, r, runner)
	})
	mux.HandleFunc("/api/progress", func(w http.ResponseWriter, r *http.Request) {
		handleGenericProgressSSE(w, r, runner)
	})
}

func handleStartAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
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
		writeJSONErrorWithStatus(w, http.StatusBadRequest, "start-all blocked: "+msg)
		return
	}
	if !runner.TryBeginBulk("start-all") {
		writeJSONConflictWithBulkProgress(w, runner, "another bulk operation already in progress")
		return
	}
	// Pre-generate runID so SSE clients can subscribe before the goroutine starts.
	runID := runner.SetActiveStartAllRunID()
	runCancellableLifecycleActionAsync("start-all", func(ctx context.Context) error {
		defer runner.EndBulk("start-all")
		return runner.StartAllWithActor(ctx, sessionID)
	}, "start-all", "all")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handleStartAllProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
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

	// Try to subscribe to the active run. Retry briefly if not yet available.
	var ch chan StartAllProgressEvent
	var runID string
	for i := 0; i < 20; i++ {
		ch, runID = runner.SubscribeStartAllProgress()
		if ch != nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if ch == nil {
		// No active start-all; send a no-op event so client knows to stop.
		fmt.Fprintf(w, "data: {\"phase\":\"idle\",\"done\":true}\n\n")
		flusher.Flush()
		return
	}

	defer runner.GetProgressBroadcaster().Unsubscribe(runID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, ev.ToSSE())
			flusher.Flush()
			if ev.Done {
				return
			}
		}
	}
}

func handleStopAllProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
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

	// Try to subscribe to the active stop-all run. Retry briefly if not yet available.
	var ch chan StartAllProgressEvent
	var runID string
	for i := 0; i < 20; i++ {
		ch, runID = runner.SubscribeStopAllProgress()
		if ch != nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if ch == nil {
		fmt.Fprintf(w, "data: {\"phase\":\"idle\",\"done\":true}\n\n")
		flusher.Flush()
		return
	}

	defer runner.GetProgressBroadcaster().Unsubscribe(runID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, ev.ToSSE())
			flusher.Flush()
			if ev.Done {
				return
			}
		}
	}
}

func handleGenericProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if runner == nil {
		writeJSONErrorWithStatus(w, http.StatusServiceUnavailable, "runner not available")
		return
	}

	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeJSONErrorWithStatus(w, http.StatusBadRequest, "missing run_id query parameter")
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

	ch := runner.SubscribeProgress(runID)
	defer runner.GetProgressBroadcaster().Unsubscribe(runID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, ev.ToSSE())
			flusher.Flush()
			if ev.Done {
				return
			}
		}
	}
}

func handleStopAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
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
	if !runner.TryBeginBulk("stop-all") {
		writeJSONConflictWithBulkProgress(w, runner, "another bulk operation already in progress")
		return
	}
	// Pre-generate runID so SSE clients can subscribe before the goroutine starts.
	runID := runner.SetActiveStopAllRunID()
	runCancellableLifecycleActionAsync("stop-all", func(ctx context.Context) error {
		defer runner.EndBulk("stop-all")
		return runner.StopAllWithActor(ctx, sessionID)
	}, "stop-all", "all")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handleCancelStartAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cancelLifecycleAction("start-all") {
		writeJSON(w, map[string]string{"status": "cancelled"})
		return
	}
	if runner != nil && runner.ClearOrphanedBulkProgress("start-all") {
		writeJSON(w, map[string]string{"status": "cleared"})
		return
	}
	writeJSONErrorWithStatus(w, http.StatusNotFound, "no active start-all operation")
}

func handleCancelStopAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cancelLifecycleAction("stop-all") {
		writeJSON(w, map[string]string{"status": "cancelled"})
		return
	}
	if runner != nil && runner.ClearOrphanedBulkProgress("stop-all") {
		writeJSON(w, map[string]string{"status": "cleared"})
		return
	}
	writeJSONErrorWithStatus(w, http.StatusNotFound, "no active stop-all operation")
}
