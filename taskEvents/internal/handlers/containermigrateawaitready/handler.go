package containermigrateawaitready

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/cloudconfig"
	"tracelog"
)

const (
	EventType = "CONTAINER_MIGRATE_AWAIT_READY"
	// MaxAttempts bounds compensation polls (~36 * RetryDelay ≈ 3min when republishing).
	MaxAttempts = 36
	RetryDelay  = 5 * time.Second
)

// ConfigLoader loads CSC for the migrate target task.
type ConfigLoader interface {
	LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error)
}

// ContainerStarter re-triggers gateway start when machine is Running but heartbeat/reachability is missing.
type ContainerStarter interface {
	Start(ctx context.Context, tenantID, workspaceID, taskID, imageID, imageURL string) error
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

// Decision is the pure await-ready evaluation result (testable without I/O).
type Decision string

const (
	DecisionReady             Decision = "ready"
	DecisionWait              Decision = "wait"
	DecisionTriggerStartWait  Decision = "trigger_start_wait"
	DecisionExhausted         Decision = "exhausted"
)

// DecideAwaitReady evaluates CSC readiness for migrate compensation.
// Ready = last_runtime_status Running AND server_url non-empty (heartbeat/reachability confirmed).
func DecideAwaitReady(cfg *cloudconfig.ConfigRow, attempt, maxAttempts int) Decision {
	if maxAttempts <= 0 {
		maxAttempts = MaxAttempts
	}
	if attempt < 1 {
		attempt = 1
	}
	if attempt > maxAttempts {
		return DecisionExhausted
	}
	if cfg == nil {
		if attempt >= maxAttempts {
			return DecisionExhausted
		}
		return DecisionWait
	}
	running := isRuntimeRunning(cfg.LastRuntimeStatus)
	hasHeartbeat := strings.TrimSpace(cfg.ServerURL) != ""
	pending := strings.HasPrefix(strings.TrimSpace(cfg.InstanceID), "pending-start-")
	if running && hasHeartbeat && !pending {
		return DecisionReady
	}
	if attempt >= maxAttempts {
		return DecisionExhausted
	}
	if running && !hasHeartbeat {
		return DecisionTriggerStartWait
	}
	return DecisionWait
}

func isRuntimeRunning(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "Running")
}

// Handler processes CONTAINER_MIGRATE_AWAIT_READY compensation polls.
type Handler struct {
	Loader    ConfigLoader
	Publisher publish.EventPublisher
	Starter   ContainerStarter
	// Sleep is injectable; production uses time.Sleep before scheduling the next attempt.
	Sleep func(time.Duration)
	// MaxAttempts overrides default/env when > 0 (tests / explicit wiring).
	MaxAttempts int
	// RetryDelay overrides default/env when >= 0 and MaxAttempts was set via struct,
	// or when RetryDelay > 0. Zero means use DefaultPollConfig / env via retryDelay().
	RetryDelay time.Duration
	// pollFromEnv caches LoadPollConfigFromEnv once per handler (lazy).
	pollFromEnv *PollConfig
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

func (h *Handler) envPoll() PollConfig {
	if h.pollFromEnv != nil {
		return *h.pollFromEnv
	}
	cfg := LoadPollConfigFromEnv()
	h.pollFromEnv = &cfg
	return cfg
}

func (h *Handler) maxAttempts() int {
	if h.MaxAttempts > 0 {
		return h.MaxAttempts
	}
	return h.envPoll().MaxAttempts
}

func (h *Handler) retryDelay() time.Duration {
	if h.RetryDelay > 0 {
		return h.RetryDelay
	}
	return h.envPoll().RetryDelay
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
	imageID := payload.StrField(data, "container_image_id")
	imageURL := payload.StrField(data, "container_image_url")

	cfg, loadErr := h.loader().LoadForTask(companyID, workspaceID, taskID)
	if loadErr != nil {
		tracelog.LogEventConsume(ctx, "migrate_await_load_failed", EventType, map[string]any{
			"task_id": taskID, "attempt": attempt, "error": loadErr.Error(),
		})
		return domain.DispatchRetryable, loadErr
	}

	decision := DecideAwaitReady(cfg, attempt, max)
	tracelog.LogEventConsume(ctx, "migrate_await_decide", EventType, map[string]any{
		"task_id":  taskID,
		"attempt":  attempt,
		"decision": string(decision),
		"runtime":  cfgRuntime(cfg),
		"has_url":  cfg != nil && strings.TrimSpace(cfg.ServerURL) != "",
	})

	switch decision {
	case DecisionReady:
		_ = publishSSE(ctx, h.Publisher, taskID, map[string]interface{}{
			"status": "success", "message": "迁移容器已就绪（Running + 心跳可达）",
			"progress": 100, "event_name": "server_status_update",
		})
		return domain.DispatchSuccess, nil

	case DecisionExhausted:
		msg := fmt.Sprintf("迁移容器就绪超时：attempt=%d/%d runtime=%s has_server_url=%v",
			attempt, max, cfgRuntime(cfg), cfg != nil && strings.TrimSpace(cfg.ServerURL) != "")
		_ = publishSSE(ctx, h.Publisher, taskID, map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		})
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)

	case DecisionTriggerStartWait:
		if h.Starter != nil {
			if serr := h.Starter.Start(ctx, tenantID, workspaceID, taskID, imageID, imageURL); serr != nil {
				log.Printf("[container_migrate_await_ready] re-trigger start failed task_id=%s: %v", taskID, serr)
			} else {
				log.Printf("[container_migrate_await_ready] re-triggered container start task_id=%s", taskID)
			}
		}
		return h.scheduleOrRetry(ctx, data, attempt, max, taskID)

	default: // DecisionWait
		return h.scheduleOrRetry(ctx, data, attempt, max, taskID)
	}
}

func (h *Handler) scheduleOrRetry(ctx context.Context, data map[string]interface{}, attempt, max int, taskID string) (domain.DispatchOutcome, error) {
	next := attempt + 1
	if next > max {
		msg := fmt.Sprintf("迁移容器就绪超时：attempt=%d/%d", attempt, max)
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}
	// Prefer controlled republish (increments attempt + unique event_id for idempotency).
	if h.Publisher != nil {
		h.sleep(h.retryDelay())
		nextData := copyMap(data)
		nextData["attempt"] = next
		nextData["event_id"] = fmt.Sprintf("migrate-await-%s-%d", taskID, next)
		if err := h.Publisher.PublishEvent(ctx, EventType, nextData, taskID); err != nil {
			return domain.DispatchRetryable, fmt.Errorf("schedule next await attempt: %w", err)
		}
		tracelog.LogEventConsume(ctx, "migrate_await_scheduled", EventType, map[string]any{
			"task_id": taskID, "next_attempt": next,
		})
		return domain.DispatchSuccess, nil
	}
	// No publisher: kafka nack/redelivery path (tests / degraded).
	return domain.DispatchRetryable, fmt.Errorf("await running/heartbeat attempt=%d/%d", attempt, max)
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

func cfgRuntime(cfg *cloudconfig.ConfigRow) string {
	if cfg == nil {
		return ""
	}
	return cfg.LastRuntimeStatus
}

func copyMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in)+2)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func publishSSE(ctx context.Context, pub publish.EventPublisher, taskID string, statusData map[string]interface{}) error {
	if pub == nil || taskID == "" {
		return nil
	}
	return pub.PublishEvent(ctx, "SSE_MESSAGE", map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	}, taskID)
}
