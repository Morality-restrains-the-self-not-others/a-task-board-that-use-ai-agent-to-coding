package relaylifecycle

import (
	"context"
	"encoding/json"
	"log"

	"taskEvents/domain"
)

// OpenRuntimeSessionHandler opens runtime session on RELAY_START_ACCEPTED via taskCloudService.
type OpenRuntimeSessionHandler struct {
	TaskCloud *TaskCloudInternalClient
}

func (h *OpenRuntimeSessionHandler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "RELAY_START_ACCEPTED" {
		return domain.DispatchPermanent, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	if str(data, "task_id") == "" {
		return domain.DispatchPermanent, domainPermanentError("missing task_id")
	}
	client := h.TaskCloud
	if client == nil {
		client = NewTaskCloudInternalClient()
	}
	payload := eventScope(data)
	payload["runtime_source"] = "relay_local"
	payload["ensure_cloud_server_config"] = true
	if err := client.post(eventCorrelationCtx(ctx, data), "/api/internal/runtime-session/open/", payload); err != nil {
		log.Printf("[relay_lifecycle] open_runtime_session failed: %v", err)
		return domain.DispatchRetryable, err
	}
	return domain.DispatchSuccess, nil
}

// ClearReachabilityHandler clears reachability on RELAY_STOP_SUCCEEDED via taskCloudService.
type ClearReachabilityHandler struct {
	TaskCloud *TaskCloudInternalClient
}

func (h *ClearReachabilityHandler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "RELAY_STOP_SUCCEEDED" {
		return domain.DispatchPermanent, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	if str(data, "task_id") == "" {
		return domain.DispatchPermanent, domainPermanentError("missing task_id")
	}
	client := h.TaskCloud
	if client == nil {
		client = NewTaskCloudInternalClient()
	}
	payload := eventScope(data)
	payload["stop_reason"] = firstNonEmpty(str(data, "reason"), "relay_stop")
	if err := client.post(eventCorrelationCtx(ctx, data), "/api/internal/cloud-server-config/clear-after-stop/", payload); err != nil {
		log.Printf("[relay_lifecycle] clear_reachability failed: %v", err)
		return domain.DispatchRetryable, err
	}
	return domain.DispatchSuccess, nil
}

// WorkflowUpdateHandler acknowledges relay startup workflow events.
// Hot path already writes Redis via taskCloudService upsert; Django transition was default no-op.
type WorkflowUpdateHandler struct{}

func (h *WorkflowUpdateHandler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if _, ok := mapWorkflowTransition(cmd.EventType); !ok {
		return domain.DispatchPermanent, nil
	}
	// Hot path already writes Redis via taskCloudService upsert; Django transition was default no-op.
	return domain.DispatchSuccess, nil
}

func mapWorkflowTransition(eventType string) (string, bool) {
	switch eventType {
	case "RELAY_START_ACCEPTED":
		return "accept", true
	case "RELAY_START_ATTEMPTED":
		return "dispatch_attempted", true
	case "RELAY_START_DISPATCH_FAILED":
		return "dispatch_failed", true
	default:
		return "", false
	}
}

func domainPermanentError(msg string) error {
	return &permanentError{msg: msg}
}

type permanentError struct{ msg string }

func (e *permanentError) Error() string { return e.msg }
