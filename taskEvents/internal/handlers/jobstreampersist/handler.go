package jobstreampersist

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/repository/saas"
	"tracelog"
)

// JobStreamPersistRequest is the Cloud internal persist body.
type JobStreamPersistRequest struct {
	CompanyID       string         `json:"company_id"`
	WorkspaceID     string         `json:"workspace_id"`
	TaskID          string         `json:"task_id"`
	CommentID       string         `json:"comment_id"`
	JobID           string         `json:"job_id"`
	Seq             int            `json:"seq"`
	Phase           string         `json:"phase"`
	Message         string         `json:"message"`
	StepNumber      int            `json:"step_number"`
	DeliverySummary string         `json:"delivery_summary"`
	StepState       string         `json:"step_state"`
	JobStatus       string         `json:"job_status"`
	LayerID         string         `json:"layer_id"`
	Event           map[string]any `json:"event,omitempty"`
}

func ShouldPersistPhase(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "step", "start", "running", "completed", "failed", "interrupted":
		return true
	default:
		return false
	}
}

type Handler struct {
	Persist func(ctx context.Context, rec JobStreamPersistRequest) error
	Repo    *saas.Repository
}

func (h *Handler) persistFn() func(ctx context.Context, rec JobStreamPersistRequest) error {
	if h.Persist != nil {
		return h.Persist
	}
	return func(ctx context.Context, rec JobStreamPersistRequest) error {
		if h.Repo == nil {
			return fmt.Errorf("missing cloud repository")
		}
		return h.Repo.PersistJobExecutionEvent(ctx, rec)
	}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "SSE_MESSAGE" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	statusData := payload.MapField(data, "status_data")
	if statusData == nil {
		return domain.DispatchSuccess, nil
	}
	if payload.StrField(statusData, "status") != "container_job_stream" {
		return domain.DispatchSuccess, nil
	}
	phase := payload.StrField(statusData, "phase")
	if !ShouldPersistPhase(phase) {
		tracelog.LogEventConsume(ctx, "job_stream_persist_skip_phase", cmd.EventType, map[string]any{
			"phase":  phase,
			"job_id": payload.StrField(statusData, "job_id"),
		})
		return domain.DispatchSuccess, nil
	}
	taskID := payload.StrField(data, "task_id")
	jobID := payload.StrField(statusData, "job_id")
	if taskID == "" || jobID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id or job_id")
	}
	rec := JobStreamPersistRequest{
		CompanyID:       firstNonEmpty(payload.StrField(statusData, "company_id"), payload.StrField(data, "company_id")),
		WorkspaceID:     firstNonEmpty(payload.StrField(statusData, "workspace_id"), payload.StrField(data, "workspace_id")),
		TaskID:          taskID,
		CommentID:       firstNonEmpty(payload.StrField(statusData, "comment_id"), payload.StrField(data, "comment_id")),
		JobID:           jobID,
		Seq:             intField(statusData, "seq"),
		Phase:           phase,
		Message:         payload.StrField(statusData, "message"),
		StepNumber:      intField(statusData, "step_number"),
		DeliverySummary: payload.StrField(statusData, "delivery_summary"),
		StepState:       firstNonEmpty(payload.StrField(statusData, "step_state"), payload.StrField(statusData, "state")),
		JobStatus:       payload.StrField(statusData, "job_status"),
		LayerID:         payload.StrField(statusData, "layer_id"),
		Event:           payload.MapField(statusData, "event"),
	}
	if err := h.persistFn()(ctx, rec); err != nil {
		tracelog.LogEventConsume(ctx, "job_stream_persist_error", cmd.EventType, map[string]any{
			"task_id": taskID,
			"job_id":  jobID,
			"error":   err.Error(),
		})
		return domain.DispatchRetryable, err
	}
	tracelog.LogEventConsume(ctx, "job_stream_persist_ok", cmd.EventType, map[string]any{
		"task_id": taskID,
		"job_id":  jobID,
		"phase":   phase,
		"seq":     rec.Seq,
	})
	return domain.DispatchSuccess, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" && v != "<nil>" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func intField(data map[string]interface{}, key string) int {
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(n))
		return i
	default:
		i, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(n)))
		return i
	}
}
