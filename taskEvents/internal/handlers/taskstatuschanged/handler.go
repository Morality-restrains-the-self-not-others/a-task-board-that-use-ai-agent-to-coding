package taskstatuschanged

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"snowflake"
	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/cloudconfig"
	"tracelog"
)

// ConfigLoader loads CloudServerConfig for a task (injectable for tests).
type ConfigLoader interface {
	LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error)
}

// LocalStopper stops a local/relay/mock runtime when server_url is set.
type LocalStopper interface {
	StopLocal(ctx context.Context, tenantID, workspaceID, taskID, serverURL, stopReason string) error
	// StopContainerOnly stops sidecars for taskID without clear-after-stop (used when CSC was already rebound).
	StopContainerOnly(ctx context.Context, taskID, stopReason string) error
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

type defaultLocalStopper struct{}

// NewDefaultLocalStopper stops relay/mock sidecars and clears CSC after stop.
func NewDefaultLocalStopper() LocalStopper {
	return defaultLocalStopper{}
}

func (defaultLocalStopper) StopLocal(ctx context.Context, tenantID, workspaceID, taskID, serverURL, stopReason string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := (defaultLocalStopper{}).StopContainerOnly(ctx, taskID, stopReason); err != nil {
		log.Printf("[task_status_changed] container-only stop soft-fail task_id=%s: %v", taskID, err)
	}
	log.Printf("[task_status_changed] local stop clear-after-stop task_id=%s server_url=%s reason=%s",
		taskID, serverURL, stopReason)
	return cloudconfig.ClearAfterStop(tenantID, workspaceID, taskID, stopReason, "", "")
}

func (defaultLocalStopper) StopContainerOnly(ctx context.Context, taskID, stopReason string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := stopRelaySidecar(ctx, taskID); err != nil {
		log.Printf("[task_status_changed] relay sidecar stop failed task_id=%s: %v (continue)", taskID, err)
	}
	if err := stopMockRunSidecar(ctx, taskID); err != nil {
		log.Printf("[task_status_changed] mock-run stop failed task_id=%s: %v (continue)", taskID, err)
	}
	log.Printf("[task_status_changed] container-only stop done task_id=%s reason=%s", taskID, stopReason)
	return nil
}

// Handler releases running servers when a task reaches a terminal status.
type Handler struct {
	Loader    ConfigLoader
	Lister    RuntimeLister
	Publisher publish.EventPublisher
	Stopper   LocalStopper
	Migrator  InstanceMigrator
	Notifier  ContainerNotifier
}

func (h *Handler) loader() ConfigLoader {
	if h.Loader != nil {
		return h.Loader
	}
	return defaultConfigLoader{}
}

func (h *Handler) stopper() LocalStopper {
	if h.Stopper != nil {
		return h.Stopper
	}
	return defaultLocalStopper{}
}

func (h *Handler) migrator() InstanceMigrator {
	if h.Migrator != nil {
		return h.Migrator
	}
	return noopMigrator{}
}

func (h *Handler) notifier() ContainerNotifier {
	if h.Notifier != nil {
		return h.Notifier
	}
	return defaultNotifier()
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "TASK_STATUS_CHANGED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}

	taskID := payload.StrField(data, "task_id")
	if taskID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id")
	}
	tenantID := payload.StrField(data, "tenant_id")
	workspaceID := payload.StrField(data, "workspace_id")
	companyID, companyErr := payload.Int64Field(data, "company_id")
	if companyErr != nil || companyID == 0 {
		if tenantID != "" {
			if n, err := payload.Int64Field(map[string]interface{}{"tenant_id": tenantID}, "tenant_id"); err == nil {
				companyID = n
			}
		}
	}
	if tenantID == "" && companyID != 0 {
		tenantID = fmt.Sprint(companyID)
	}
	if companyID == 0 {
		return domain.DispatchPermanent, fmt.Errorf("missing company_id/tenant_id")
	}

	columnName := payload.StrField(data, "progress_column_name")
	prevCompleted := boolField(data, "previous_completed")
	completed := boolField(data, "completed")
	completedBecameTrue := !prevCompleted && completed

	terminalKind := strings.TrimSpace(payload.StrField(data, "terminal_kind"))
	if terminalKind == "" {
		terminalKind = ResolveTerminalKind(columnName, completedBecameTrue)
	}
	if terminalKind != "completed" && terminalKind != "cancelled" {
		tracelog.LogEventConsume(ctx, "non_terminal_skip", cmd.EventType, map[string]any{
			"task_id":     taskID,
			"column_name": columnName,
			"completed":   completed,
		})
		return domain.DispatchSuccess, nil
	}

	stopReason := "task_status_completed"
	if terminalKind == "cancelled" {
		stopReason = "task_status_cancelled"
	}

	list, loadErr := h.resolveRuntimes(companyID, workspaceID, taskID)
	if loadErr != nil {
		log.Printf("[task_status_changed] load config failed task_id=%s: %v", taskID, loadErr)
		tracelog.LogEventConsume(ctx, "config_load_failed", cmd.EventType, map[string]any{
			"task_id": taskID,
			"error":   loadErr.Error(),
		})
		return domain.DispatchRetryable, loadErr
	}
	if list == nil {
		list = &cloudconfig.TaskRuntimeList{}
	}
	releaseable := filterReleaseable(list.Comments)
	tracelog.LogEventConsume(ctx, "terminal_release_comment_csc", cmd.EventType, map[string]any{
		"task_id":     taskID,
		"machines":    list.RunningMachineCount,
		"containers":  list.RunningContainerCount,
		"candidates":  len(list.Comments),
		"releaseable": len(releaseable),
	})
	if len(releaseable) == 0 {
		tracelog.LogEventConsume(ctx, "no_running_resource_skip", cmd.EventType, map[string]any{
			"task_id":       taskID,
			"terminal_kind": terminalKind,
		})
		return domain.DispatchSuccess, nil
	}

	wsID := workspaceID
	if wsID == "" {
		wsID = strings.TrimSpace(releaseable[0].WorkspaceID)
	}

	mig := h.migrator()
	if err := mig.SetTerminalReleasedFlag(ctx, tenantID, wsID, taskID); err != nil {
		log.Printf("[task_status_changed] set terminal_released flag failed task_id=%s: %v", taskID, err)
		tracelog.LogEventConsume(ctx, "set_terminal_released_flag_failed", cmd.EventType, map[string]any{
			"task_id": taskID, "error": err.Error(),
		})
		return domain.DispatchRetryable, err
	}

	anyHard := false
	for i := range releaseable {
		row := &releaseable[i]
		kind, relErr := h.releaseOne(ctx, cmd.EventType, tenantID, wsID, taskID, companyID, terminalKind, stopReason, row)
		if relErr != nil {
			if strings.Contains(relErr.Error(), "publisher not configured") {
				return domain.DispatchPermanent, relErr
			}
			return domain.DispatchRetryable, relErr
		}
		if kind == releaseHard {
			anyHard = true
		}
	}
	if !anyHard {
		return domain.DispatchSuccess, nil
	}

	if err := mig.MarkTerminalReleased(ctx, tenantID, wsID, taskID); err != nil {
		log.Printf("[task_status_changed] mark terminal_released failed task_id=%s: %v", taskID, err)
		tracelog.LogEventConsume(ctx, "mark_terminal_released_failed", cmd.EventType, map[string]any{
			"task_id": taskID, "error": err.Error(),
		})
		return domain.DispatchRetryable, err
	}
	tracelog.LogEventConsume(ctx, "terminal_hard_release_done", cmd.EventType, map[string]any{
		"task_id": taskID, "terminal_kind": terminalKind, "released": len(releaseable),
	})
	return domain.DispatchSuccess, nil
}

func buildStopVmEventData(tenantID, workspaceID, taskID string, companyID int64, row *cloudconfig.ConfigRow, stopReason string) map[string]interface{} {
	wsID := strings.TrimSpace(workspaceID)
	region := ""
	authID := ""
	platform := "aliyun"
	instanceID := ""
	commentID := ""
	if row != nil {
		region = strings.TrimSpace(row.Region)
		authID = strings.TrimSpace(row.AuthorizationID)
		platform = strings.TrimSpace(row.Platform)
		instanceID = strings.TrimSpace(row.InstanceID)
		commentID = strings.TrimSpace(row.CommentID)
		if platform == "" {
			platform = "aliyun"
		}
		if wsID == "" {
			wsID = strings.TrimSpace(row.WorkspaceID)
		}
	}
	company := tenantID
	if company == "" {
		company = fmt.Sprint(companyID)
	}
	return map[string]interface{}{
		"task_id":             taskID,
		"comment_id":          commentID,
		"tenant_id":           tenantID,
		"company_id":          company,
		"workspace_id":        wsID,
		"instance_id":         instanceID,
		"region_id":           region,
		"authorization_id":    authID,
		"cloud_platform_type": platform,
		"stop_reason":         stopReason,
		"stop_request_id":     fmt.Sprintf("%d", snowflake.GenerateID()),
	}
}

func boolField(data map[string]interface{}, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch n := v.(type) {
	case bool:
		return n
	case string:
		return strings.EqualFold(strings.TrimSpace(n), "true") || n == "1"
	case float64:
		return n != 0
	case int:
		return n != 0
	case int64:
		return n != 0
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		return strings.EqualFold(s, "true") || s == "1"
	}
}
