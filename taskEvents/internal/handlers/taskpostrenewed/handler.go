package taskpostrenewed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"tracelog"
)

// Handler consumes TASK_POST_RENEWED (taskTaskService 续存成功后发布)。
// 帖子续存 = 消耗 1 创建帖次数，存续期延长 12 个月；此处记录续存并供
// 下游通知消费者（站内通知、Webhook）订阅。
type Handler struct{}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "TASK_POST_RENEWED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		tracelog.LogEventConsume(ctx, "task_post_renewed_bad_payload", cmd.EventType, map[string]any{
			"error": err.Error(),
		})
		return domain.DispatchPermanent, err
	}
	log.Printf("[task_post_renewed] task post renewed: task_id=%v tenant=%v workspace=%v user=%v expires_at=%v",
		data["task_id"], data["tenant_id"], data["workspace_id"], data["user_id"], data["post_expires_at"])
	tracelog.LogEventConsume(ctx, "task_post_renewed", cmd.EventType, map[string]any{
		"task_id":        data["task_id"],
		"tenant_id":      data["tenant_id"],
		"workspace_id":   data["workspace_id"],
		"user_id":        data["user_id"],
		"post_expires_at": data["post_expires_at"],
	})
	return domain.DispatchSuccess, nil
}
