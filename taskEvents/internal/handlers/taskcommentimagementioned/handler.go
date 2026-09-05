package taskcommentimagementioned

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"autorunstartvm"
	"taskEvents/domain"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"tracelog"
)

// Handler processes TASK_COMMENT_IMAGE_MENTIONED → start-vm(-auto) with idle reuse.
type Handler struct {
	Publisher  publish.EventPublisher
	Tasks      TaskClient
	Projects   ProjectClient
	Cloud      CloudClient
	AIComments AICommentClient
}

func (h *Handler) tasks() TaskClient {
	if h.Tasks != nil {
		return h.Tasks
	}
	return NewDefaultTaskClient()
}

func (h *Handler) projects() ProjectClient {
	if h.Projects != nil {
		return h.Projects
	}
	return NewDefaultProjectClient()
}

func (h *Handler) cloud() CloudClient {
	if h.Cloud != nil {
		return h.Cloud
	}
	return NewDefaultCloudClient()
}

func (h *Handler) aiComments() AICommentClient {
	if h.AIComments != nil {
		return h.AIComments
	}
	return NewDefaultAICommentClient()
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "TASK_COMMENT_IMAGE_MENTIONED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}

	taskID := payload.StrField(data, "task_id")
	tenantID := payload.StrField(data, "tenant_id")
	if tenantID == "" {
		tenantID = payload.StrField(data, "company_id")
	}
	workspaceID := payload.StrField(data, "workspace_id")
	imageID := payload.StrField(data, "installed_image_id")
	createdBy := payload.StrField(data, "created_by_id")
	parentCommentID := payload.StrField(data, "parent_comment_id")
	if parentCommentID == "" {
		parentCommentID = payload.StrField(data, "comment_id")
	}

	if taskID == "" || tenantID == "" || workspaceID == "" || imageID == "" || createdBy == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id, tenant_id, workspace_id, installed_image_id, or created_by_id")
	}

	if !shouldColdStartOnImageMention(data) {
		containerName := ""
		if parentCommentID != "" {
			containerName = cloudcommon.DeriveContainerName("", taskID, parentCommentID)
		}
		scope := cloudcommon.StartupLogScope{
			TaskID: taskID, CommentID: parentCommentID, ContainerName: containerName,
		}
		log.Printf("[task_comment_image_mentioned] event=at_mention_start_vm_deferred comment_id=%s task_id=%s reason=wait_previous",
			parentCommentID, taskID)
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status":     "processing",
			"message":    "串行评论等待前序完成后再启动运行环境",
			"progress":   10,
			"event_name": "server_status_update",
		}, scope))
		tracelog.LogEventConsume(ctx, "at_mention_start_vm_deferred", cmd.EventType, map[string]any{
			"task_id":           taskID,
			"parent_comment_id": parentCommentID,
			"execution_mode":    payload.StrField(data, "execution_mode"),
		})
		return domain.DispatchSuccess, nil
	}

	tracelog.LogEventConsume(ctx, "at_mention_start_vm_begin", cmd.EventType, map[string]any{
		"task_id":            taskID,
		"tenant_id":          tenantID,
		"workspace_id":       workspaceID,
		"installed_image_id": imageID,
		"parent_comment_id":  parentCommentID,
	})
	containerName := ""
	if parentCommentID != "" {
		containerName = cloudcommon.DeriveContainerName("", taskID, parentCommentID)
	}
	scope := cloudcommon.StartupLogScope{
		TaskID:        taskID,
		CommentID:     parentCommentID,
		ContainerName: containerName,
	}
	_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
		"status":     "processing",
		"message":    "正在根据 @镜像 启动运行环境...",
		"progress":   20,
		"event_name": "server_status_update",
	}, scope))

	tpl, outcome, err := h.resolveMentionStartTemplate(ctx, data, taskID, tenantID, createdBy, scope)
	if err != nil {
		return outcome, err
	}

	req, err := autorunstartvm.BuildAutoRunStartVmRequest(taskID, imageID, tpl)
	if err != nil {
		msg := fmt.Sprintf("无法构建 start-vm 请求: %v", err)
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}, scope))
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}
	autorunstartvm.ApplyRuntimeSource(req.Body, autorunstartvm.RuntimeSourceCloudVMCommentMention)
	// Propagate comment/image identity into cloud start events → SSE log_label.
	req.Body["installed_image_id"] = imageID
	req.Body["container_image_id"] = imageID
	if parentCommentID != "" {
		req.Body["comment_id"] = parentCommentID
		req.Body["parent_comment_id"] = parentCommentID
		req.Body["container_name"] = cloudcommon.DeriveContainerName("", taskID, parentCommentID)
		req.Body["mock_container_name"] = req.Body["container_name"]
	}
	if rawIdents, ok := data["repo_identities"]; ok && rawIdents != nil {
		req.Body["repo_identities"] = rawIdents
	}

	tracelog.LogEventConsume(ctx, "at_mention_start_vm_post", cmd.EventType, map[string]any{
		"task_id":  taskID,
		"api_path": req.APIPath,
	})
	if err := h.cloud().StartVM(ctx, tenantID, workspaceID, req.APIPath, createdBy, req.Body); err != nil {
		msg := fmt.Sprintf("启动云主机失败: %v", err)
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}, scope))
		return domain.DispatchRetryable, fmt.Errorf("%s", msg)
	}

	if parentCommentID != "" {
		if err := h.aiComments().SetRunStatusByParent(ctx, parentCommentID, "starting"); err != nil {
			if errors.Is(err, ErrAICommentNotFound) {
				// start-vm 已成功；AI 评论不存在/已清理 → 状态回写跳过，属预期降级而非启服失败。
				log.Printf("[task_comment_image_mentioned] warn: 按 parent 回写 AI 评论运行状态失败（start-vm 已成功，忽略该回写）: %v", err)
			} else {
				log.Printf("[task_comment_image_mentioned] set agent status starting failed parent=%s: %v", parentCommentID, err)
			}
		}
	}

	_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
		"status":     "processing",
		"message":    "已提交启动请求",
		"progress":   40,
		"event_name": "server_status_update",
	}, scope))
	tracelog.LogEventConsume(ctx, "at_mention_start_vm_ok", cmd.EventType, map[string]any{
		"task_id":  taskID,
		"api_path": req.APIPath,
	})
	return domain.DispatchSuccess, nil
}

func completeEventServerRunTemplate(m map[string]interface{}) map[string]interface{} {
	if m == nil || len(m) == 0 {
		return nil
	}
	region := strings.TrimSpace(payload.StrField(m, "region"))
	platform := strings.TrimSpace(payload.StrField(m, "cloud_platform_id"))
	if region == "" || platform == "" {
		return nil
	}
	return m
}

func (h *Handler) resolveMentionStartTemplate(
	ctx context.Context,
	data map[string]interface{},
	taskID, tenantID, createdBy string,
	scope cloudcommon.StartupLogScope,
) (map[string]interface{}, domain.DispatchOutcome, error) {
	if override := completeEventServerRunTemplate(payload.MapField(data, "server_run_template")); override != nil {
		return override, domain.DispatchSuccess, nil
	}

	linked, err := h.tasks().FetchTaskLinkedProjects(ctx, taskID, tenantID, createdBy)
	if err != nil {
		msg := fmt.Sprintf("加载任务关联项目失败: %v", err)
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}, scope))
		return nil, domain.DispatchRetryable, fmt.Errorf("%s", msg)
	}

	projects := make([]map[string]interface{}, 0, len(linked))
	var fetchFailN int
	for _, row := range linked {
		pid := strings.TrimSpace(payload.StrField(row, "project_id"))
		if pid == "" {
			pid = strings.TrimSpace(payload.StrField(row, "id"))
		}
		if pid == "" {
			continue
		}
		proj, err := h.projects().FetchProject(ctx, tenantID, pid, createdBy)
		if err != nil {
			fetchFailN++
			log.Printf("[task_comment_image_mentioned] project fetch failed project_id=%s: %v", pid, err)
			continue
		}
		projects = append(projects, proj)
	}

	tpl := autorunstartvm.ResolveProjectServerRunTemplateFromProjects(projects, linked)
	if tpl == nil {
		msg := "任务关联项目未配置 server_run_template"
		if len(linked) > 0 && len(projects) == 0 && fetchFailN > 0 {
			msg = fmt.Sprintf("加载任务关联项目详情失败（%d 次），无法读取 server_run_template", fetchFailN)
		}
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}, scope))
		return nil, domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}
	return tpl, domain.DispatchSuccess, nil
}
