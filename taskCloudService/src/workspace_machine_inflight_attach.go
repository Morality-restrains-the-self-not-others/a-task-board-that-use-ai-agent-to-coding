package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"tracelog"
)

type idleReuseResult struct {
	Reused     bool
	InstanceID string
	PublicIP   string
}

// applyStartVmPolicyGate 校验工作空间机器策略，并尝试同评论 inflight 附着。
// 跨任务闲置复用已移除（ADR-0013）：无候选机时返回 handled=false，由调用方冷启动。
func applyStartVmPolicyGate(w http.ResponseWriter, ctx context.Context, tenantID, workspaceID, taskID string, cloudAuth *cloudAuthRecord, commentID, containerImageID string) (handled bool, err error) {
	if cloudAuth == nil {
		return false, nil
	}
	ok, msg := enforceStartVmMachinePolicy(tenantID, workspaceID, cloudAuth.ID)
	if !ok {
		_ = publishTaskSSE(ctx, taskID, commentID, startVmErrorStatusData(tenantID, workspaceID, msg))
		writeErrorMapJSON(w, nil, http.StatusForbidden, map[string]interface{}{"status": "error", "message": msg})
		return true, nil
	}
	if inflight, inflightErr := tryAttachSameTaskInFlightMachine(tenantID, workspaceID, taskID, commentID, containerImageID); inflightErr != nil {
		return false, inflightErr
	} else if inflight.Reused {
		msg := "已附着本任务启动中/运行中的机器节点（跳过二次冷启动）"
		_ = publishTaskSSE(ctx, taskID, commentID, withStartupImageLogScope(map[string]interface{}{
			"instance_id": inflight.InstanceID,
		}, map[string]interface{}{
			"status": "success", "message": msg, "progress": 100, "event_name": "server_status_update",
			"reuse": true, "inflight_attach": true, "instance_id": inflight.InstanceID,
		}))
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":          "success",
			"reuse":           true,
			"inflight_attach": true,
			"instance_id":     inflight.InstanceID,
			"public_ip":       inflight.PublicIP,
			"message":         msg,
			"trace_id":        tracelog.TraceIDFromContext(ctx),
		})
		return true, nil
	}
	return false, nil
}

func enforceStartVmMachinePolicy(tenantID, workspaceID, authorizationID string) (bool, string) {
	policy, err := loadWorkspaceMachinePolicy(tenantID, workspaceID)
	if err != nil {
		return true, "读取工作空间机器策略失败: " + err.Error()
	}
	allowed, msg := authorizationAllowedByPolicy(policy, authorizationID)
	if !allowed {
		return false, msg
	}
	return true, ""
}

// tryAttachSameTaskInFlightMachine 仅附着「同一评论」自己的 Starting/Running 实例，
// 供同评论重试 start-vm 幂等。任务级模板行不再持有运行实例，禁止从任务级领养。
func tryAttachSameTaskInFlightMachine(tenantID, workspaceID, taskID, commentID, containerImageID string) (idleReuseResult, error) {
	_ = containerImageID
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return idleReuseResult{}, nil
	}
	cfg, err := loadCloudServerConfigForComment(tenantID, workspaceID, taskID, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return idleReuseResult{}, nil
		}
		return idleReuseResult{}, err
	}
	if cfg == nil {
		return idleReuseResult{}, nil
	}
	healCommentCSCMockMetaFromTaskBase(cfg)
	inst := strings.TrimSpace(cfg.InstanceID)
	if inst == "" || isMockMachineInstanceID(inst) {
		return idleReuseResult{}, nil
	}
	status := cfg.LastRuntimeStatus
	if machineRuntimeCountsAsStarting(inst, status) || machineRuntimeCountsAsStarted(inst, status) {
		logInfo("event=inflight_machine_attach task_id="+taskID+
			" instance_id="+inst+" runtime_status="+strings.TrimSpace(status)+
			" comment_id="+commentID, taskID)
		return idleReuseResult{
			Reused:     true,
			InstanceID: inst,
			PublicIP:   strings.TrimSpace(cfg.PublicIP),
		}, nil
	}
	return idleReuseResult{}, nil
}
