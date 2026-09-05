package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func registerServiceHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/restart", func(w http.ResponseWriter, r *http.Request) {
		handleSingleServiceProgress(w, r, runner, "restart", func(ctx context.Context, name string, sessionID string) error {
			log.Printf("[api] restart request for %s", name)
			return runner.RestartServiceWithActor(ctx, name, sessionID)
		})
	})
	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		handleSingleServiceProgressCascade(w, r, runner, "stop",
			runner.StopServiceCascadeWithActor,
			runner.StopServiceWithActor,
		)
	})
	mux.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		handleSingleServiceProgressCascade(w, r, runner, "start",
			runner.StartServiceCascadeWithActor,
			runner.StartServiceWithActor,
		)
	})
	mux.HandleFunc("/api/stop-group", func(w http.ResponseWriter, r *http.Request) {
		handleServiceActionWithSession(w, r, runner, "group", runner.StopGroupWithActor)
	})
	mux.HandleFunc("/api/start-group", func(w http.ResponseWriter, r *http.Request) {
		handleServiceActionWithSession(w, r, runner, "group", runner.StartGroupWithActor)
	})
	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		handleServiceAction(w, r, runner, "name", func(ctx context.Context, name string) error {
			log.Printf("[api] build request for %s", name)
			return runner.BuildService(ctx, name)
		})
	})
	mux.HandleFunc("/api/build-group", func(w http.ResponseWriter, r *http.Request) {
		handleBuildGroupAction(w, r, runner)
	})
}

func handleServiceAction(w http.ResponseWriter, r *http.Request, runner *Runner, fieldName string, action func(context.Context, string) error) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readStringFieldFromJSON(r, fieldName)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	if err := action(r.Context(), body); err != nil {
		writeJSONError(w, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// handleSingleServiceProgress handles a single-service lifecycle action with SSE progress.
// Used for restart (no cascade).
func handleSingleServiceProgress(
	w http.ResponseWriter,
	r *http.Request,
	runner *Runner,
	op string,
	action func(context.Context, string, string) error,
) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	name, err := readRequiredStringField(payload, "name")
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
	if err := validateLifecycleTarget(runner, "name", name); err != nil {
		writeJSONError(w, err.Error())
		return
	}

	runID := runner.GenerateSingleRunID(op, name)
	runLifecycleActionAsync(func(ctx context.Context) error {
		runner.PublishProgress(runID, StartAllProgressEvent{
			Total: 1, Current: name, Remaining: 1,
			Phase: "starting", Operation: op,
		})
		err := action(ctx, name, sessionID)
		if err != nil {
			runner.PublishProgress(runID, StartAllProgressEvent{
				Total: 1, Failed: 1, Remaining: 0, Done: true,
				Error:  fmt.Sprintf("%s: %v", name, err),
				Errors: []string{fmt.Sprintf("%s: %v", name, err)},
				Phase:  "done", Operation: op,
			})
		} else {
			runner.PublishProgress(runID, StartAllProgressEvent{
				Total: 1, Started: 1, Remaining: 0, Done: true,
				Phase: "done", Operation: op,
			})
		}
		runner.CloseProgressRun(runID, 30*time.Second)
		return err
	}, name, name)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

// handleSingleServiceProgressCascade handles a single-service action with cascade flag and SSE progress.
// Used for start/stop.
func handleSingleServiceProgressCascade(
	w http.ResponseWriter,
	r *http.Request,
	runner *Runner,
	op string,
	cascadeAction func(context.Context, string, string) error,
	singleAction func(context.Context, string, string) error,
) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	name, err := readRequiredStringField(payload, "name")
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
	if err := validateLifecycleTarget(runner, "name", name); err != nil {
		writeJSONError(w, err.Error())
		return
	}
	action := cascadeAction
	if !readCascadeFlag(payload) {
		action = singleAction
	}

	// Dry-run preview: return the cascade plan without executing, so the UI can
	// confirm wide-impact stops (e.g. stopping task-auth would cascade to
	// task-gateway → taskFE). See OPT-20260811-033.
	if readPreviewFlag(payload) {
		preview, err := buildLifecyclePreview(runner, op, name)
		if err != nil {
			writeJSONError(w, err.Error())
			return
		}
		writeJSON(w, preview)
		return
	}

	runID := runner.GenerateSingleRunID(op, name)
	runLifecycleActionAsync(func(ctx context.Context) error {
		runner.PublishProgress(runID, StartAllProgressEvent{
			Total: 1, Current: name, Remaining: 1,
			Phase: "starting", Operation: op,
		})
		err := action(ctx, name, sessionID)
		if err != nil {
			runner.PublishProgress(runID, StartAllProgressEvent{
				Total: 1, Failed: 1, Remaining: 0, Done: true,
				Error:  fmt.Sprintf("%s: %v", name, err),
				Errors: []string{fmt.Sprintf("%s: %v", name, err)},
				Phase:  "done", Operation: op,
			})
		} else {
			runner.PublishProgress(runID, StartAllProgressEvent{
				Total: 1, Started: 1, Remaining: 0, Done: true,
				Phase: "done", Operation: op,
			})
		}
		runner.CloseProgressRun(runID, 30*time.Second)
		return err
	}, name, name)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handleServiceActionWithSession(
	w http.ResponseWriter,
	r *http.Request,
	runner *Runner,
	fieldName string,
	action func(context.Context, string, string) error,
) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	body, err := readRequiredStringField(payload, fieldName)
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
	if err := validateLifecycleTarget(runner, fieldName, body); err != nil {
		writeJSONError(w, err.Error())
		return
	}
	runLifecycleActionAsync(func(ctx context.Context) error {
		return action(ctx, body, sessionID)
	}, fieldName, body)
	writeLifecycleAccepted(w)
}

func handleCascadeServiceActionWithSession(
	w http.ResponseWriter,
	r *http.Request,
	runner *Runner,
	fieldName string,
	cascadeAction func(context.Context, string, string) error,
	singleAction func(context.Context, string, string) error,
) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	body, err := readRequiredStringField(payload, fieldName)
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
	if err := validateLifecycleTarget(runner, fieldName, body); err != nil {
		writeJSONError(w, err.Error())
		return
	}
	action := cascadeAction
	if !readCascadeFlag(payload) {
		action = singleAction
	}
	runLifecycleActionAsync(func(ctx context.Context) error {
		return action(ctx, body, sessionID)
	}, fieldName, body)
	writeLifecycleAccepted(w)
}

func validateLifecycleTarget(runner *Runner, fieldName, body string) error {
	switch fieldName {
	case "name":
		if runner.findService(body) == nil {
			return fmt.Errorf("service %q not found", body)
		}
	case "group":
		for _, group := range runner.cfg.Groups {
			if group.Name == body {
				return nil
			}
		}
		return fmt.Errorf("group %q not found", body)
	default:
		return fmt.Errorf("unsupported lifecycle target %q", fieldName)
	}
	return nil
}
