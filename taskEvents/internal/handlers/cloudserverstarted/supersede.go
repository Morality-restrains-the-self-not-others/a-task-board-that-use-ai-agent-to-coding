package cloudserverstarted

import (
	"context"
	"fmt"
	"log"
	"strings"

	"autorunstartvm"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/repository/cloudconfig"
	"tracelog"
)

type supersedeResult struct {
	// ReusedExisting means the prior binding must be kept; caller short-circuits without RunInstances.
	ReusedExisting bool
	InstanceID     string
}

// supersedeExistingInstance may release a previously bound ECS before RunInstances so a
// deliberate restart cannot orphan the first instance under the same InstanceName.
//
// It MUST NOT release an instance that is still useful to the task:
//   - @镜像评论启动（runtime_source=cloud_vm_comment_mention）：永远附着，不删旧机
//   - last_runtime_status 为 Running / Starting / Pending / Initializing：附着，不删
//   - DeleteInstance 若返回 IncorrectInstanceStatus.* 过渡态：附着，不删
func (h *Handler) supersedeExistingInstance(ctx context.Context, companyID int64, workspaceID, taskID, regionID, accessKey, secretKey, runtimeSource string) (supersedeResult, error) {
	row, err := cloudconfig.LoadForTask(companyID, workspaceID, taskID)
	if err != nil || row == nil {
		return supersedeResult{}, nil
	}
	oldID := strings.TrimSpace(row.InstanceID)
	if oldID == "" || strings.HasPrefix(oldID, "mock-") {
		return supersedeResult{}, nil
	}
	region := strings.TrimSpace(regionID)
	if region == "" {
		region = strings.TrimSpace(row.Region)
	}
	if region == "" {
		region = "cn-hongkong"
	}

	if shouldPreserveBoundInstance(runtimeSource, row.LastRuntimeStatus) {
		log.Printf("[cloud_server_started] supersede skipped (preserve bound instance) task=%s instance=%s runtime_source=%s last_runtime=%s",
			taskID, oldID, strings.TrimSpace(runtimeSource), strings.TrimSpace(row.LastRuntimeStatus))
		tracelog.LogEventConsume(ctx, "supersede_old_instance_reuse_inflight", "CLOUD_SERVER_STARTED", map[string]any{
			"task_id":             taskID,
			"instance_id":         oldID,
			"runtime_source":      strings.TrimSpace(runtimeSource),
			"last_runtime_status": strings.TrimSpace(row.LastRuntimeStatus),
			"reason":              "preserve_bound_instance",
		})
		return supersedeResult{ReusedExisting: true, InstanceID: oldID}, nil
	}

	tracelog.LogEventConsume(ctx, "supersede_old_instance_begin", "CLOUD_SERVER_STARTED", map[string]any{
		"task_id":     taskID,
		"instance_id": oldID,
		"region_id":   region,
	})
	if _, stopErr := h.stopper().StopVM(accessKey, secretKey, aliyun.StopVMInput{
		RegionID:   region,
		InstanceID: oldID,
	}); stopErr != nil {
		raw := stopErr.Error()
		if isNonStoppableInstanceStatusError(raw) {
			log.Printf("[cloud_server_started] supersede skipped (instance transitional) task=%s instance=%s err=%v",
				taskID, oldID, stopErr)
			tracelog.LogEventConsume(ctx, "supersede_old_instance_reuse_inflight", "CLOUD_SERVER_STARTED", map[string]any{
				"task_id":     taskID,
				"instance_id": oldID,
				"error":       raw,
			})
			return supersedeResult{ReusedExisting: true, InstanceID: oldID}, nil
		}
		if !strings.Contains(raw, "InvalidInstanceId.NotFound") && !strings.Contains(raw, "InvalidInstanceId") {
			return supersedeResult{}, stopErr
		}
		log.Printf("[cloud_server_started] supersede old instance already gone task=%s instance=%s", taskID, oldID)
	}
	tenantID := fmt.Sprint(companyID)
	if clearErr := cloudconfig.ClearAfterStop(tenantID, workspaceID, taskID, "superseded_by_new_start", oldID, ""); clearErr != nil {
		log.Printf("[cloud_server_started] supersede clear-after-stop soft-fail task=%s: %v", taskID, clearErr)
	}
	tracelog.LogEventConsume(ctx, "supersede_old_instance_ok", "CLOUD_SERVER_STARTED", map[string]any{
		"task_id":     taskID,
		"instance_id": oldID,
	})
	return supersedeResult{}, nil
}

// shouldPreserveBoundInstance reports whether an existing ECS binding must not be deleted.
// Alive / mid-boot instances are never released by a later @镜像评论 (or any start that would
// clobber a still-useful node). Explicitly down instances (Stopped/…) may still be superseded.
func shouldPreserveBoundInstance(runtimeSource, lastRuntimeStatus string) bool {
	status := strings.ToLower(strings.TrimSpace(lastRuntimeStatus))
	switch status {
	case "running", "starting", "pending", "initializing":
		return true
	case "stopped", "stopping", "released", "terminated":
		return false
	default:
		// Empty/unknown: treat as mid-boot for @镜像 so the second comment cannot DeleteInstance
		// the first comment's node before last_runtime_status is reconciled.
		return strings.TrimSpace(runtimeSource) == autorunstartvm.RuntimeSourceCloudVMCommentMention
	}
}

func isNonStoppableInstanceStatusError(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.Contains(raw, "IncorrectInstanceStatus.Initializing") ||
		strings.Contains(raw, "IncorrectInstanceStatus.Starting") ||
		strings.Contains(raw, "IncorrectInstanceStatus.Pending") {
		return true
	}
	if strings.Contains(raw, "IncorrectInstanceStatus") {
		lower := strings.ToLower(raw)
		return strings.Contains(lower, "initializing") ||
			strings.Contains(lower, "starting") ||
			strings.Contains(lower, "pending") ||
			strings.Contains(lower, "running")
	}
	return false
}
