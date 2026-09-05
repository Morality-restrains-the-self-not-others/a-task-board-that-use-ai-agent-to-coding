package cloudserverstopped

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/cloudconfig"
	"taskEvents/internal/repository/saas"
	"tracelog"
)

// VMStopper abstracts ECS DeleteInstance (mockable in tests).
type VMStopper interface {
	StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error)
}

type aliyunStopper struct{}

func (aliyunStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	return aliyun.StopVM(accessKey, secretKey, in)
}

// Handler processes CLOUD_SERVER_STOPPED (release ECS + clear reachability).
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
	Stopper   VMStopper
}

func (h *Handler) stopper() VMStopper {
	if h.Stopper != nil {
		return h.Stopper
	}
	return aliyunStopper{}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "CLOUD_SERVER_STOPPED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	if h.Publisher == nil {
		return domain.DispatchPermanent, fmt.Errorf("SSE publisher not configured")
	}
	if h.Repo == nil {
		return domain.DispatchPermanent, fmt.Errorf("saas repository not configured")
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}

	taskID := payload.StrField(data, "task_id")
	companyID, err := payload.Int64Field(data, "company_id")
	if err != nil || taskID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id or company_id")
	}
	commentID := payload.StrField(data, "comment_id")
	workspaceID := payload.StrField(data, "workspace_id")
	tenantID := payload.StrField(data, "tenant_id")
	if tenantID == "" {
		tenantID = fmt.Sprint(companyID)
	}
	stopRequestID := payload.StrField(data, "stop_request_id")

	// OPT-20260818-015：owner DB 已按 stop_request_id 落库（cloud_stop_request 唯一键）。
	// 重放同一 stop_request_id 时直接跳过，避免重复 DeleteInstance / ClearAfterStop。
	// 检查失败 fail-open（当作未处理），真实停止绝不能被幂等误跳。
	if stopRequestID != "" {
		processed, checkErr := cloudconfig.StopRequestProcessed(stopRequestID)
		if checkErr != nil {
			tracelog.LogEventConsume(ctx, "stop_idempotency_check_failed", cmd.EventType, map[string]any{
				"task_id":         taskID,
				"stop_request_id": stopRequestID,
				"error":           checkErr.Error(),
			})
		} else if processed {
			tracelog.LogEventConsume(ctx, "idempotency skip", cmd.EventType, map[string]any{
				"task_id":         taskID,
				"stop_request_id": stopRequestID,
			})
			return domain.DispatchSuccess, nil
		}
	}

	platformType := payload.StrField(data, "cloud_platform_type")
	if platformType == "" {
		platformType = "aliyun"
	}

	var sseEvents []map[string]interface{}
	defer func() {
		if pubErr := cloudcommon.PublishSSEBatch(ctx, h.Publisher, sseEvents); pubErr != nil {
			tracelog.LogEventConsume(ctx, "sse_publish_failed", cmd.EventType, map[string]any{
				"task_id": taskID,
				"error":   pubErr.Error(),
			})
		}
	}()

	platformLabel := platformType
	if platformLabel == "" {
		platformLabel = "云"
	}

	instanceID := payload.StrField(data, "instance_id")
	regionID := payload.StrField(data, "region_id")
	if instanceID == "" || regionID == "" {
		if row, loadErr := cloudconfig.LoadForTask(companyID, workspaceID, taskID); loadErr == nil && row != nil {
			if instanceID == "" {
				instanceID = strings.TrimSpace(row.InstanceID)
			}
			if regionID == "" {
				regionID = strings.TrimSpace(row.Region)
			}
		}
	}
	if instanceID == "" {
		msg := "未提供实例ID"
		h.failStop(taskID, commentID, &sseEvents, msg)
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}
	if regionID == "" {
		regionID = "cn-hongkong"
	}

	if remapped := cloudcommon.NormalizeStopPlatform(platformType, instanceID); remapped != platformType {
		tracelog.LogEventConsume(ctx, "stop_vm_platform_remapped", cmd.EventType, map[string]any{
			"task_id":     taskID,
			"from":        platformType,
			"to":          remapped,
			"instance_id": instanceID,
		})
		platformType = remapped
	}
	if platformType != "aliyun" && !cloudcommon.IsLocalSkipCloudPlatform(platformType) {
		return domain.DispatchPermanent, fmt.Errorf("unsupported platform %s", platformType)
	}

	stopReason := payload.StrField(data, "stop_reason")
	if stopReason == "" {
		stopReason = "stop_vm"
	}
	stopReasonLabel := payload.StrField(data, "stop_reason_label")

	isMockInstance := cloudcommon.ShouldSkipCloudAPI(platformType, instanceID)
	processingMsg := fmt.Sprintf("正在调用%sAPI停止服务器...", platformLabel)
	successMsg := "停止虚拟机成功"
	if isMockInstance {
		processingMsg = "正在停止 Mock 实例..."
		successMsg = "Mock 实例已停止"
	}
	processingMsg = cloudcommon.AnnotateStopServerMessageWithLabel(processingMsg, stopReason, stopReasonLabel)
	successMsg = cloudcommon.AnnotateStopServerMessageWithLabel(successMsg, stopReason, stopReasonLabel)
	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id": taskID,
		"status_data": attachStopLiveFields(commentID, "Stopping", map[string]interface{}{
			"status": "processing", "message": processingMsg,
			"progress": 50, "event_name": "server_status_update",
		}),
	})

	tracelog.LogEventConsume(ctx, "stop_vm_begin", cmd.EventType, map[string]any{
		"task_id":     taskID,
		"instance_id": instanceID,
		"region_id":   regionID,
	})

	resultInstanceID := instanceID
	resultRegionID := regionID

	// Mock / local Docker instances are not real ECS; skip Aliyun DeleteInstance.
	if isMockInstance {
		tracelog.LogEventConsume(ctx, "stop_vm_mock_skip_cloud", cmd.EventType, map[string]any{
			"task_id":     taskID,
			"instance_id": instanceID,
		})
	} else {
		// authorization_id 为平台 SSOT 字符串 ID（cpa_<snowflake>），不可用 Int64Field
		// 解析（strconv.ParseInt 对 "cpa_..." 失败 → 误报缺少 authorization_id，OPT-20260809-026）。
		authID := payload.StrField(data, "authorization_id")
		if authID == "" {
			h.failStop(taskID, commentID, &sseEvents, "缺少 authorization_id")
			return domain.DispatchPermanent, fmt.Errorf("missing authorization_id")
		}
		auth, err := h.Repo.CloudAuthorizationByID(authID)
		if err != nil {
			h.failStop(taskID, commentID, &sseEvents, err.Error())
			return domain.DispatchPermanent, err
		}

		result, stopErr := h.stopper().StopVM(auth.SecretID, auth.SecretKey, aliyun.StopVMInput{
			RegionID:   regionID,
			InstanceID: instanceID,
		})
		if stopErr != nil {
			tracelog.LogEventConsume(ctx, "stop_vm_failed", cmd.EventType, map[string]any{
				"task_id":     taskID,
				"instance_id": instanceID,
				"error":       stopErr.Error(),
			})
			h.failStop(taskID, commentID, &sseEvents, stopErr.Error())
			return domain.DispatchRetryable, stopErr
		}
		if strings.TrimSpace(result.InstanceID) != "" {
			resultInstanceID = result.InstanceID
		}
		if strings.TrimSpace(result.RegionID) != "" {
			resultRegionID = result.RegionID
		}
	}

	_ = cloudconfig.ClearAfterStop(tenantID, workspaceID, taskID, stopReason, resultInstanceID, stopRequestID)

	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id": taskID,
		"status_data": attachStopLiveFields(commentID, "Stopped", map[string]interface{}{
			"status": "success", "message": successMsg, "progress": 100, "event_name": "server_status_update",
			"vm_info": map[string]interface{}{"instance_id": resultInstanceID, "region": resultRegionID},
		}),
	})
	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id":     taskID,
		"status_data": map[string]interface{}{"type": "close", "event_name": "server_status_update"},
	})

	tracelog.LogEventConsume(ctx, "stop_vm_success", cmd.EventType, map[string]any{
		"task_id":     taskID,
		"instance_id": resultInstanceID,
		"region_id":   resultRegionID,
	})
	return domain.DispatchSuccess, nil
}

func attachStopLiveFields(commentID, runtimeStatus string, status map[string]interface{}) map[string]interface{} {
	if status == nil {
		status = map[string]interface{}{}
	}
	if strings.TrimSpace(commentID) != "" {
		status["comment_id"] = strings.TrimSpace(commentID)
	}
	if strings.TrimSpace(runtimeStatus) != "" {
		status["runtime_status"] = strings.TrimSpace(runtimeStatus)
	}
	return status
}

func (h *Handler) failStop(taskID, commentID string, sse *[]map[string]interface{}, msg string) {
	*sse = append(*sse, map[string]interface{}{
		"task_id": taskID,
		"status_data": attachStopLiveFields(commentID, "", map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}),
	})
	*sse = append(*sse, map[string]interface{}{
		"task_id":     taskID,
		"status_data": map[string]interface{}{"type": "close", "event_name": "server_status_update"},
	})
}
