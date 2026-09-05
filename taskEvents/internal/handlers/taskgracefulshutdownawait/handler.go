package taskgracefulshutdownawait

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"snowflake"
	"taskEvents/domain"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/handlers/taskstatuschanged"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/cloudconfig"
	"tracelog"
)

const (
	EventType   = "TASK_GRACEFUL_SHUTDOWN_AWAIT"
	MaxAttempts = 18
	RetryDelay  = 5 * time.Second
)

// ConfigLoader loads CSC for the terminal task.
type ConfigLoader interface {
	LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error)
}

type defaultConfigLoader struct{}

func (defaultConfigLoader) LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error) {
	row, err := cloudconfig.LoadForTask(companyID, workspaceID, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// Decision for await evaluation.
type Decision string

const (
	DecisionReleased  Decision = "released"
	DecisionWait      Decision = "wait"
	DecisionExhausted Decision = "exhausted"
)

// DecideAwait evaluates whether graceful release completed.
func DecideAwait(cfg *cloudconfig.ConfigRow, attempt, maxAttempts int) Decision {
	if maxAttempts <= 0 {
		maxAttempts = MaxAttempts
	}
	if attempt < 1 {
		attempt = 1
	}
	if isReleased(cfg) {
		return DecisionReleased
	}
	if attempt >= maxAttempts {
		return DecisionExhausted
	}
	return DecisionWait
}

func isReleased(cfg *cloudconfig.ConfigRow) bool {
	if cfg == nil {
		return true
	}
	return strings.TrimSpace(cfg.InstanceID) == "" && strings.TrimSpace(cfg.ServerURL) == ""
}

// Handler polls until container released the node, then hard-releases on timeout.
type Handler struct {
	Loader      ConfigLoader
	Publisher   publish.EventPublisher
	Stopper     taskstatuschanged.LocalStopper
	Migrator    taskstatuschanged.InstanceMigrator
	Sleep       func(time.Duration)
	MaxAttempts int
	RetryDelay  time.Duration
}

func (h *Handler) loader() ConfigLoader {
	if h.Loader != nil {
		return h.Loader
	}
	return defaultConfigLoader{}
}

func (h *Handler) sleep(d time.Duration) {
	if h.Sleep != nil {
		h.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (h *Handler) maxAttempts() int {
	if h.MaxAttempts > 0 {
		return h.MaxAttempts
	}
	return MaxAttempts
}

func (h *Handler) retryDelay() time.Duration {
	if h.RetryDelay > 0 {
		return h.RetryDelay
	}
	return RetryDelay
}

func (h *Handler) stopper() taskstatuschanged.LocalStopper {
	if h.Stopper != nil {
		return h.Stopper
	}
	return nil
}

func (h *Handler) migrator() taskstatuschanged.InstanceMigrator {
	if h.Migrator != nil {
		return h.Migrator
	}
	return nil
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != EventType {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}

	taskID := payload.StrField(data, "task_id")
	workspaceID := payload.StrField(data, "workspace_id")
	tenantID := payload.StrField(data, "tenant_id")
	if tenantID == "" {
		tenantID = payload.StrField(data, "company_id")
	}
	if taskID == "" || tenantID == "" || workspaceID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id, tenant_id/company_id, or workspace_id")
	}
	companyID, err := payload.Int64Field(data, "company_id")
	if err != nil {
		companyID, err = payload.Int64Field(data, "tenant_id")
	}
	if err != nil {
		return domain.DispatchPermanent, fmt.Errorf("invalid company_id/tenant_id: %w", err)
	}

	attempt := attemptFromData(data)
	max := h.maxAttempts()
	stopReason := payload.StrField(data, "stop_reason")
	if stopReason == "" {
		stopReason = "task_status_" + payload.StrField(data, "terminal_kind")
	}
	if stopReason == "task_status_" {
		stopReason = "task_status_cancelled"
	}

	cfg, loadErr := h.loader().LoadForTask(companyID, workspaceID, taskID)
	if loadErr != nil {
		tracelog.LogEventConsume(ctx, "graceful_await_load_failed", EventType, map[string]any{
			"task_id": taskID, "attempt": attempt, "error": loadErr.Error(),
		})
		return domain.DispatchRetryable, loadErr
	}

	decision := DecideAwait(cfg, attempt, max)
	tracelog.LogEventConsume(ctx, "graceful_await_decide", EventType, map[string]any{
		"task_id":  taskID,
		"attempt":  attempt,
		"decision": string(decision),
	})

	switch decision {
	case DecisionReleased:
		if mig := h.migrator(); mig != nil {
			_ = mig.MarkTerminalReleased(ctx, tenantID, workspaceID, taskID)
		}
		return domain.DispatchSuccess, nil

	case DecisionExhausted:
		return h.hardRelease(ctx, data, cfg, tenantID, workspaceID, taskID, companyID, stopReason)

	default:
		return h.scheduleNext(ctx, data, attempt, max, taskID)
	}
}

func (h *Handler) scheduleNext(ctx context.Context, data map[string]interface{}, attempt, max int, taskID string) (domain.DispatchOutcome, error) {
	next := attempt + 1
	if next > max {
		return domain.DispatchPermanent, fmt.Errorf("graceful await exhausted attempt=%d/%d", attempt, max)
	}
	if h.Publisher == nil {
		return domain.DispatchRetryable, fmt.Errorf("graceful await pending attempt=%d/%d", attempt, max)
	}
	h.sleep(h.retryDelay())
	nextData := copyMap(data)
	nextData["attempt"] = next
	nextData["event_id"] = fmt.Sprintf("graceful-await-%s-%d", taskID, next)
	if err := h.Publisher.PublishEvent(ctx, EventType, nextData, taskID); err != nil {
		return domain.DispatchRetryable, fmt.Errorf("schedule next graceful await: %w", err)
	}
	tracelog.LogEventConsume(ctx, "graceful_await_scheduled", EventType, map[string]any{
		"task_id": taskID, "next_attempt": next,
	})
	return domain.DispatchSuccess, nil
}

func (h *Handler) hardRelease(ctx context.Context, data map[string]interface{}, cfg *cloudconfig.ConfigRow, tenantID, workspaceID, taskID string, companyID int64, stopReason string) (domain.DispatchOutcome, error) {
	serverURL := payload.StrField(data, "server_url")
	if cfg != nil && strings.TrimSpace(cfg.ServerURL) != "" {
		serverURL = strings.TrimSpace(cfg.ServerURL)
	}
	instanceID := payload.StrField(data, "instance_id")
	if cfg != nil && strings.TrimSpace(cfg.InstanceID) != "" {
		instanceID = strings.TrimSpace(cfg.InstanceID)
	}

	log.Printf("[task_graceful_shutdown_await] timeout hard-release task_id=%s instance=%s server_url=%s", taskID, instanceID, serverURL)

	if serverURL != "" && h.stopper() != nil {
		if err := h.stopper().StopLocal(ctx, tenantID, workspaceID, taskID, serverURL, stopReason); err != nil {
			return domain.DispatchRetryable, err
		}
	}
	if instanceID != "" {
		if h.Publisher == nil {
			return domain.DispatchPermanent, fmt.Errorf("event publisher not configured")
		}
		eventData := map[string]interface{}{
			"task_id":             taskID,
			"tenant_id":           tenantID,
			"company_id":          tenantID,
			"workspace_id":        workspaceID,
			"instance_id":         instanceID,
			"stop_reason":         stopReason,
			"cloud_platform_type": "aliyun",
			"stop_request_id":     fmt.Sprintf("%d", snowflake.GenerateID()),
		}
		if cfg != nil {
			if r := strings.TrimSpace(cfg.Region); r != "" {
				eventData["region_id"] = r
			}
			if a := strings.TrimSpace(cfg.AuthorizationID); a != "" {
				eventData["authorization_id"] = a
			}
			if p := strings.TrimSpace(cfg.Platform); p != "" {
				eventData["cloud_platform_type"] = cloudcommon.NormalizeStopPlatform(p, instanceID)
			}
		}
		_ = companyID
		if err := h.Publisher.PublishEvent(ctx, "CLOUD_SERVER_STOPPED", eventData, taskID); err != nil {
			return domain.DispatchRetryable, err
		}
	}
	if mig := h.migrator(); mig != nil {
		if err := mig.MarkTerminalReleased(ctx, tenantID, workspaceID, taskID); err != nil {
			return domain.DispatchRetryable, err
		}
	}
	tracelog.LogEventConsume(ctx, "graceful_await_timeout_hard_release", EventType, map[string]any{
		"task_id": taskID, "instance_id": instanceID,
	})
	return domain.DispatchSuccess, nil
}

func attemptFromData(data map[string]interface{}) int {
	v, ok := data["attempt"]
	if !ok || v == nil {
		return 1
	}
	switch n := v.(type) {
	case float64:
		if int(n) < 1 {
			return 1
		}
		return int(n)
	case int:
		if n < 1 {
			return 1
		}
		return n
	case int64:
		if n < 1 {
			return 1
		}
		return int(n)
	default:
		parsed, err := payload.Int64Field(data, "attempt")
		if err != nil || parsed < 1 {
			return 1
		}
		return int(parsed)
	}
}

func copyMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}
