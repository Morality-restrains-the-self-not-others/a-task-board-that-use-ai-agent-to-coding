package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func registerBuildHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/build-all", func(w http.ResponseWriter, r *http.Request) {
		handleBuildAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/build-all/cancel", func(w http.ResponseWriter, r *http.Request) {
		handleCancelBuildAllAction(w, r, runner)
	})
	mux.HandleFunc("/api/build-all/progress", func(w http.ResponseWriter, r *http.Request) {
		handleBuildAllProgressSSE(w, r, runner)
	})
	mux.HandleFunc("/api/conf/sync", func(w http.ResponseWriter, r *http.Request) {
		handleConfSyncAction(w, r, runner)
	})
}

func handleConfSyncAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	runLifecycleActionAsync(func(ctx context.Context) error {
		return runner.SyncMonorepoConf(ctx)
	}, confSyncLifecycleLabel, "all")
	writeLifecycleAccepted(w)
}

func handleBuildGroupAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Group string `json:"group"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid json")
		return
	}
	body.Group = strings.TrimSpace(body.Group)
	if body.Group == "" {
		writeJSONError(w, "group is required")
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	if runner.findServicesByGroup(body.Group) == nil {
		writeJSONError(w, fmt.Sprintf("group %q not found", body.Group))
		return
	}
	log.Printf("[api] build-group request for %s", body.Group)
	runID, ok := runner.TryBeginBuildAllRun()
	if !ok {
		log.Printf("[api] build-group conflict: bulk rebuild already in progress group=%s", body.Group)
		writeJSONConflictWithBulkProgress(w, runner, "bulk rebuild already in progress")
		return
	}
	groupName := body.Group
	runCancellableLifecycleActionAsync("build-all", func(ctx context.Context) error {
		defer runner.EndBulk("build-all")
		_, err := runner.BuildGroup(ctx, groupName)
		return err
	}, "build-group", groupName)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID, "group": groupName})
}

func handleBuildAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	log.Printf("[api] build-all request")
	runID, ok := runner.TryBeginBuildAllRun()
	if !ok {
		log.Printf("[api] build-all conflict: bulk rebuild already in progress")
		writeJSONConflictWithBulkProgress(w, runner, "bulk rebuild already in progress")
		return
	}
	if runner.ConsumePendingExecQueueType("build-all") {
		log.Printf("[api] build-all consumed matching pending queue slot run_id=%s", runID)
	}
	runCancellableLifecycleActionAsync("build-all", func(ctx context.Context) error {
		defer runner.EndBulk("build-all")
		_, err := runner.BuildAll(ctx)
		return err
	}, "build-all", "all")
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handleCancelBuildAllAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cancelLifecycleAction("build-all") {
		writeJSON(w, map[string]string{"status": "cancelled"})
		return
	}
	if runner != nil && runner.ClearOrphanedBulkProgress("build-all") {
		writeJSON(w, map[string]string{"status": "cleared"})
		return
	}
	writeJSONErrorWithStatus(w, http.StatusNotFound, "no active build-all operation")
}

func handleBuildAllProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
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

	var ch chan StartAllProgressEvent
	var runID string
	for i := 0; i < 20; i++ {
		ch, runID = runner.SubscribeBuildAllProgress()
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
