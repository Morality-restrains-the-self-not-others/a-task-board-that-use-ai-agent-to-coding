package billingordercomment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
	"tracelog"
)

const (
	authorSideTenant      = "tenant"
	authorSideSystemAdmin = "system_admin"
)

// Handler consumes BILLING_ORDER_COMMENT_CREATED (taskBill 发布)：订单评论产生后
// 按 author_side 通知对侧（一期为 SSE——订单级 Redis 通道，租户/超管订单页均可订阅，
// 载荷含 author_side 供接收方过滤自己是否「对侧」）。
type Handler struct {
	RedisHost string
	RedisPort int
	RedisDB   int
}

// publishOrderCommentSSEFn 可注入（单测替换），默认真实 Redis Publish。
var publishOrderCommentSSEFn = (*Handler).publishOrderCommentSSE

func (h *Handler) client() *redis.Client {
	port := h.RedisPort
	if port == 0 {
		port = 6379
	}
	host := h.RedisHost
	if host == "" {
		host = "127.0.0.1"
	}
	return redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%d", host, port), DB: h.RedisDB})
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "BILLING_ORDER_COMMENT_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		tracelog.LogEventConsume(ctx, "billing_order_comment_bad_payload", cmd.EventType, map[string]any{
			"error": err.Error(),
		})
		return domain.DispatchPermanent, err
	}
	commentID := strings.TrimSpace(fieldString(data, "comment_id"))
	orderID := strings.TrimSpace(fieldString(data, "order_id"))
	tenantID := strings.TrimSpace(fieldString(data, "tenant_id"))
	authorSide := strings.TrimSpace(fieldString(data, "author_side"))
	authorUserID := strings.TrimSpace(fieldString(data, "author_user_id"))
	if commentID == "" || orderID == "" || tenantID == "" || authorSide == "" {
		tracelog.LogEventConsume(ctx, "billing_order_comment_missing_fields", cmd.EventType, map[string]any{
			"comment_id": commentID, "order_id": orderID, "tenant_id": tenantID, "author_side": authorSide,
		})
		return domain.DispatchPermanent, fmt.Errorf("missing comment_id/order_id/tenant_id/author_side")
	}
	if authorSide != authorSideTenant && authorSide != authorSideSystemAdmin {
		return domain.DispatchPermanent, fmt.Errorf("invalid author_side %q", authorSide)
	}

	tracelog.LogEventConsume(ctx, "billing_order_comment_created", cmd.EventType, map[string]any{
		"comment_id":     commentID,
		"order_id":       orderID,
		"tenant_id":      tenantID,
		"author_side":    authorSide,
		"author_user_id": authorUserID,
	})

	if err := publishOrderCommentSSEFn(h, ctx, data, commentID, orderID, tenantID, authorSide); err != nil {
		tracelog.LogEventConsume(ctx, "billing_order_comment_sse_error", cmd.EventType, map[string]any{
			"order_id": orderID, "comment_id": commentID, "error": err.Error(),
		})
		return domain.DispatchRetryable, err
	}
	return domain.DispatchSuccess, nil
}

// buildOrderCommentSSEPayload 构造订单级 SSE 通知载荷（纯函数，便于单测）。
func buildOrderCommentSSEPayload(data map[string]interface{}, commentID, orderID, tenantID, authorSide string) []byte {
	payload := map[string]interface{}{
		"task_id": "billing:order:" + orderID, // 供 taskSSE normalizeInboundMessage 复用既有 hub 键空间
		"status_data": map[string]interface{}{
			"event_name":     "billing_order_comment_created",
			"comment_id":     commentID,
			"order_id":       orderID,
			"tenant_id":      tenantID,
			"author_side":    authorSide,
			"author_user_id": fieldString(data, "author_user_id"),
			"created_at":     fieldString(data, "created_at"),
			"message":        "订单有新评论",
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

// publishOrderCommentSSE 向订单级 Redis 通道 sse:billing:order:<order_id> 发布通知，
// 供租户订单页 / 超管订单记录页的「对侧」订阅方感知新评论（一期不推邮件）。
func (h *Handler) publishOrderCommentSSE(ctx context.Context, data map[string]interface{}, commentID, orderID, tenantID, authorSide string) error {
	body := buildOrderCommentSSEPayload(data, commentID, orderID, tenantID, authorSide)
	channel := "sse:billing:order:" + orderID
	rdb := h.client()
	defer rdb.Close()
	if _, err := rdb.Publish(ctx, channel, body).Result(); err != nil {
		return err
	}
	return nil
}

func fieldString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
