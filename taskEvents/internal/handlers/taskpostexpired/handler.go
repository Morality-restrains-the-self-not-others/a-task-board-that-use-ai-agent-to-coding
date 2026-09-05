package taskpostexpired

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"tracelog"
)

// Handler consumes TASK_POST_EXPIRED（taskTaskService 到期扫描发布）。
// 帖子 12 个月有效期到期下架；此处记录到期并供下游通知消费者
// （站内通知提醒续存、Webhook）订阅。
type Handler struct{}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "TASK_POST_EXPIRED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		tracelog.LogEventConsume(ctx, "task_post_expired_bad_payload", cmd.EventType, map[string]any{
			"error": err.Error(),
		})
		return domain.DispatchPermanent, err
	}
	log.Printf("[task_post_expired] task post expired: task_id=%v tenant=%v workspace=%v",
		data["task_id"], data["tenant_id"], data["workspace_id"])
	tracelog.LogEventConsume(ctx, "task_post_expired", cmd.EventType, map[string]any{
		"task_id":      data["task_id"],
		"tenant_id":    data["tenant_id"],
		"workspace_id": data["workspace_id"],
	})
	return domain.DispatchSuccess, nil
}
