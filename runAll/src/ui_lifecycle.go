package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

func runLifecycleActionAsync(action func(context.Context) error, fieldName, body string) {
	go func() {
		if err := action(context.Background()); err != nil {
			log.Printf("[ui] async %s %q failed: %v", fieldName, body, err)
		}
	}()
}

var lifecycleCancelRegistry = struct {
	sync.Mutex
	startAll       context.CancelFunc
	stopAll        context.CancelFunc
	buildAll       context.CancelFunc
	restartAll     context.CancelFunc
	preciseRestart context.CancelFunc
}{}

func runCancellableLifecycleActionAsync(op string, action func(context.Context) error, fieldName, body string) {
	ctx, cancel := context.WithCancel(context.Background())
	setLifecycleCancel(op, cancel)
	go func() {
		defer clearLifecycleCancel(op)
		if err := action(ctx); err != nil {
			log.Printf("[ui] async %s %q failed: %v", fieldName, body, err)
		}
	}()
}

func setLifecycleCancel(op string, cancel context.CancelFunc) {
	lifecycleCancelRegistry.Lock()
	defer lifecycleCancelRegistry.Unlock()
	switch op {
	case "start-all":
		lifecycleCancelRegistry.startAll = cancel
	case "stop-all":
		lifecycleCancelRegistry.stopAll = cancel
	case "build-all":
		lifecycleCancelRegistry.buildAll = cancel
	case "restart-all":
		lifecycleCancelRegistry.restartAll = cancel
	case "precise-restart":
		lifecycleCancelRegistry.preciseRestart = cancel
	}
}

func clearLifecycleCancel(op string) {
	lifecycleCancelRegistry.Lock()
	defer lifecycleCancelRegistry.Unlock()
	switch op {
	case "start-all":
		lifecycleCancelRegistry.startAll = nil
	case "stop-all":
		lifecycleCancelRegistry.stopAll = nil
	case "build-all":
		lifecycleCancelRegistry.buildAll = nil
	case "restart-all":
		lifecycleCancelRegistry.restartAll = nil
	case "precise-restart":
		lifecycleCancelRegistry.preciseRestart = nil
	}
}

func cancelLifecycleAction(op string) bool {
	lifecycleCancelRegistry.Lock()
	defer lifecycleCancelRegistry.Unlock()
	var cancel context.CancelFunc
	switch op {
	case "start-all":
		cancel = lifecycleCancelRegistry.startAll
	case "stop-all":
		cancel = lifecycleCancelRegistry.stopAll
	case "build-all":
		cancel = lifecycleCancelRegistry.buildAll
	case "restart-all":
		cancel = lifecycleCancelRegistry.restartAll
	case "precise-restart":
		cancel = lifecycleCancelRegistry.preciseRestart
	default:
		return false
	}
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func writeLifecycleAccepted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted"})
}

func readCascadeFlag(payload map[string]any) bool {
	raw, ok := payload["cascade"]
	if !ok {
		return true
	}
	value, ok := raw.(bool)
	if !ok {
		return true
	}
	return value
}

// readPreviewFlag reports whether the payload asks for a dry-run plan preview
// instead of executing the lifecycle action (OPT-20260811-033).
func readPreviewFlag(payload map[string]any) bool {
	raw, ok := payload["preview"]
	if !ok {
		return false
	}
	value, ok := raw.(bool)
	if !ok {
		return false
	}
	return value
}

// buildLifecyclePreview returns the cascade plan for a lifecycle action without
// executing it. The UI uses this to confirm wide-impact stops before they run:
// stopping a core dependency (e.g. task-auth) would otherwise cascade-stop all
// downstream services (task-gateway → taskFE) and take the public site down.
func buildLifecyclePreview(runner *Runner, op, name string) (map[string]any, error) {
	services := []string{name}
	switch op {
	case "stop":
		plan, err := runner.cascadeOrchestration().PlanStopCascade(name)
		if err != nil {
			return nil, err
		}
		if len(plan.OrderedNames) > 0 {
			services = plan.OrderedNames
		}
	case "start":
		plan, err := runner.cascadeOrchestration().PlanStartCascade(name)
		if err != nil {
			return nil, err
		}
		if len(plan.OrderedNames) > 0 {
			services = plan.OrderedNames
		}
	}
	return map[string]any{
		"status":   "preview",
		"op":       op,
		"target":   name,
		"services": services,
	}, nil
}

func writeCascadeError(w http.ResponseWriter, failure *CascadeFailure) {
	message := "cascade failed"
	if failure != nil && failure.Err != nil {
		message = failure.Err.Error()
	}
	payload := map[string]any{"error": message}
	if failure != nil {
		payload["cascade"] = map[string]any{
			"completed": failure.Report.Completed,
			"failed_at": failure.Report.FailedAt,
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ui] json encode error: %v", err)
	}
}

func readStringFieldFromJSON(r *http.Request, fieldName string) (string, error) {
	payload, err := readJSONPayload(r)
	if err != nil {
		return "", err
	}
	return readRequiredStringField(payload, fieldName)
}

func readJSONPayload(r *http.Request) (map[string]any, error) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid json")
	}
	return payload, nil
}

func readRequiredStringField(payload map[string]any, fieldName string) (string, error) {
	raw, ok := payload[fieldName]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	return value, nil
}
