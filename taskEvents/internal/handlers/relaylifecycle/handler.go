package relaylifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"taskEvents/domain"
)

// Handler appends relay lifecycle bus events to taskCredentialService audit SSOT.
type Handler struct {
	CredentialBaseURL string
	HTTPClient        *http.Client
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	eventType := strings.TrimSpace(cmd.EventType)
	if !strings.HasPrefix(eventType, "RELAY_") {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", eventType)
	}
	auditType, ok := mapRelayBusToAuditEventType(eventType)
	if !ok {
		log.Printf("[relay_lifecycle] skip unmapped event %s", eventType)
		return domain.DispatchSuccess, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	taskID := str(data, "task_id")
	if taskID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id")
	}
	payload := map[string]interface{}{
		"tenant_id":        str(data, "tenant_id"),
		"workspace_id":     str(data, "workspace_id"),
		"task_id":          taskID,
		"event_type":       auditType,
		"source_component": "task-container-gateway",
		"trace_id":         firstNonEmpty(str(data, "trace_id"), str(data, "request_id")),
		"error_code":       str(data, "error_code"),
		"error_detail":     str(data, "error_detail"),
	}
	if seq, ok := data["seq"].(float64); ok {
		payload["seq"] = int(seq)
	}
	if tok := str(data, "access_token"); tok != "" {
		payload["access_token"] = tok
	}
	if err := h.postAudit(ctx, payload); err != nil {
		log.Printf("[relay_lifecycle] audit append failed event=%s task=%s: %v", eventType, taskID, err)
		return domain.DispatchRetryable, err
	}
	log.Printf("[relay_lifecycle] audit appended event=%s audit_type=%s task=%s", eventType, auditType, taskID)
	return domain.DispatchSuccess, nil
}

func mapRelayBusToAuditEventType(busType string) (string, bool) {
	switch busType {
	case "RELAY_REGISTER_ATTEMPTED":
		return "relay_register_attempted", true
	case "RELAY_REGISTER_SUCCEEDED":
		return "relay_register_succeeded", true
	case "RELAY_REGISTER_FAILED":
		return "relay_register_failed", true
	case "RELAY_START_ACCEPTED":
		return "relay_start_accepted", true
	case "RELAY_START_ATTEMPTED":
		return "relay_start_attempted", true
	case "RELAY_START_DISPATCH_SUCCEEDED":
		return "relay_start_succeeded", true
	case "RELAY_START_DISPATCH_FAILED":
		return "relay_start_failed", true
	case "RELAY_STOP_REQUESTED":
		return "relay_stop_requested", true
	case "RELAY_STOP_SUCCEEDED":
		return "relay_stop_succeeded", true
	case "RELAY_STOP_FAILED":
		return "relay_stop_failed", true
	default:
		return "", false
	}
}

func (h *Handler) postAudit(ctx context.Context, payload map[string]interface{}) error {
	base := strings.TrimRight(strings.TrimSpace(h.CredentialBaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8015"
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/audit/append", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := h.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{Proxy: nil}}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("audit append HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func str(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
