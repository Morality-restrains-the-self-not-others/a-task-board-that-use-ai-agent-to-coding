package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	relayPhaseRunning     = "running"
	relayPhaseStartFailed = "start_failed"
)

// RelayStatusConvergeResult mirrors Django RelayStatusConvergeResult.
type RelayStatusConvergeResult struct {
	Converged   bool
	TraceID     string
	WorkflowID  string
	ErrorCode   string
	ErrorDetail string
}

func buildFallbackTraceID(scope RelayTaskScope, seq *int, workflowID string) string {
	seqPart := "na"
	if seq != nil {
		seqPart = fmt.Sprintf("%d", *seq)
	}
	if workflowID != "" {
		return fmt.Sprintf("relay-status:%s:%s", workflowID, seqPart)
	}
	return fmt.Sprintf("relay-status:%s:%s:%s:%s", scope.TenantID, scope.WorkspaceID, scope.TaskID, seqPart)
}

func snapshotFromRelayStatusMap(relayStatus map[string]any) RelayStatusSnapshot {
	errText := strings.TrimSpace(fmt.Sprintf("%v", relayStatus["error"]))
	if errText == "<nil>" {
		errText = ""
	}
	if len(errText) > 500 {
		errText = errText[:500]
	}
	return RelayStatusSnapshot{
		Running:         toBool(relayStatus["running"]),
		OnlineServiceUp: toBool(relayStatus["online_service_up"]),
		Error:           errText,
	}
}

func toBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case string:
		text := strings.TrimSpace(strings.ToLower(v))
		switch text {
		case "1", "true", "yes", "y", "on":
			return true
		default:
			return false
		}
	default:
		text := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", value)))
		switch text {
		case "1", "true", "yes", "y", "on":
			return true
		default:
			return false
		}
	}
}

// convergeStatusPushOnSession applies Django RelayStartupSession.converge_status_push rules.
func convergeStatusPushOnSession(session *RelayStartupSession, seq *int, status RelayStatusSnapshot, now time.Time) error {
	if seq == nil && session.LastStatusSeq != nil {
		return fmt.Errorf("status seq 已建立后不可缺失")
	}
	if seq != nil && *seq < 0 {
		return fmt.Errorf("status seq 不能为负数")
	}
	if seq != nil && session.LastStatusSeq != nil && *seq < *session.LastStatusSeq {
		return fmt.Errorf("status seq 不能回退")
	}
	cp := status
	session.LastStatus = &cp
	session.LastStatusSeq = seq
	if status.convergedRunning() {
		session.Phase = relayPhaseRunning
		session.FailureReason = ""
	} else if status.hasError() {
		session.Phase = relayPhaseStartFailed
		session.FailureReason = status.Error
	}
	session.UpdatedAt = now
	session.UpdatedAtRaw = now.UTC().Format(time.RFC3339Nano)
	return nil
}

// convergeRelayStatusPushByScope is the Go equivalent of Django converge_relay_status_push_by_scope.
func convergeRelayStatusPushByScope(
	ctx context.Context,
	store *relaySessionStore,
	scope RelayTaskScope,
	seq *int,
	status RelayStatusSnapshot,
	traceID string,
	now time.Time,
) RelayStatusConvergeResult {
	normalizedTrace := strings.TrimSpace(traceID)
	if store == nil || store.client == nil {
		return RelayStatusConvergeResult{
			Converged:   false,
			TraceID:     normalizedTrace,
			ErrorCode:   "status_converge_failed",
			ErrorDetail: "relay redis session store unavailable",
		}
	}
	session, err := store.findLatestByScope(ctx, scope)
	if err != nil {
		effective := normalizedTrace
		if effective == "" {
			effective = buildFallbackTraceID(scope, seq, "")
		}
		return RelayStatusConvergeResult{
			Converged:   false,
			TraceID:     effective,
			ErrorCode:   "status_converge_failed",
			ErrorDetail: err.Error(),
		}
	}
	if session == nil {
		effective := normalizedTrace
		if effective == "" {
			effective = buildFallbackTraceID(scope, seq, "")
		}
		return RelayStatusConvergeResult{
			Converged:   false,
			TraceID:     effective,
			WorkflowID:  "",
			ErrorCode:   "status_converge_session_missing",
			ErrorDetail: "status push 未找到可收敛的 relay startup session",
		}
	}

	workflowID := session.WorkflowID
	convergedTrace := normalizedTrace
	if convergedTrace == "" && session.RequestID != nil {
		convergedTrace = strings.TrimSpace(*session.RequestID)
	}
	if convergedTrace == "" {
		convergedTrace = buildFallbackTraceID(scope, seq, workflowID)
	}

	if err := convergeStatusPushOnSession(session, seq, status, now); err != nil {
		detail := strings.TrimSpace(err.Error())
		if detail == "" {
			detail = "status push convergence failed"
		}
		return RelayStatusConvergeResult{
			Converged:   false,
			TraceID:     convergedTrace,
			WorkflowID:  workflowID,
			ErrorCode:   "status_converge_failed",
			ErrorDetail: detail,
		}
	}
	if err := store.save(ctx, session); err != nil {
		return RelayStatusConvergeResult{
			Converged:   false,
			TraceID:     convergedTrace,
			WorkflowID:  workflowID,
			ErrorCode:   "status_converge_failed",
			ErrorDetail: err.Error(),
		}
	}
	return RelayStatusConvergeResult{
		Converged:  true,
		TraceID:    convergedTrace,
		WorkflowID: workflowID,
	}
}

// applyRelayStatusPushLocal: Redis converge (sync) + debounced Kafka SSE (async).
func applyRelayStatusPushLocal(
	ctx context.Context,
	tenantID, workspaceID, taskID string,
	seq *int,
	relayStatus map[string]any,
	traceID string,
) RelayStatusConvergeResult {
	if relayStatus == nil {
		relayStatus = map[string]any{}
	}
	scope := RelayTaskScope{
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
	}
	snapshot := snapshotFromRelayStatusMap(relayStatus)
	result := convergeRelayStatusPushByScope(ctx, relaySessions, scope, seq, snapshot, traceID, time.Now().UTC())
	if !result.Converged {
		log.Printf(
			"[taskCloudService] status push converge failed task_id=%s workflow_id=%s seq=%v code=%s detail=%s",
			taskID, result.WorkflowID, seq, result.ErrorCode, result.ErrorDetail,
		)
	}
	// SSE after converge so UI sees consistent phase; debounce absorbs status-push storms.
	scheduleRelayStatusSSEDebounced(taskID, relayStatus, traceID)
	return result
}
