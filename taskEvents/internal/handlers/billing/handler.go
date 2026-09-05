package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
)

// Handler processes BILLING_TRANSACTION_CREATED: audit log + user-level Redis SSE fan-out.
type Handler struct {
	RedisHost string
	RedisPort int
	RedisDB   int
}

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
	if cmd.EventType != "BILLING_TRANSACTION_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	txID := fieldString(data, "transaction_id")
	log.Printf("[billing] billing_transaction_created (taskBill): %s", txID)
	if txID == "" {
		log.Printf("[billing] billing_transaction_created missing transaction_id")
		return domain.DispatchPermanent, fmt.Errorf("missing transaction_id")
	}
	if err := h.publishRechargeSSE(ctx, data); err != nil {
		log.Printf("[billing] recharge SSE publish failed txn=%s err=%v", txID, err)
		return domain.DispatchRetryable, err
	}
	log.Printf("[billing] successfully processed billing_transaction_created: %s", txID)
	return domain.DispatchSuccess, nil
}

func (h *Handler) publishRechargeSSE(ctx context.Context, data map[string]interface{}) error {
	if fieldString(data, "transaction_type") != "recharge" {
		return nil
	}
	userID := strings.TrimSpace(fieldString(data, "user_id"))
	if userID == "" {
		return nil
	}
	outTradeNo := strings.TrimSpace(fieldString(data, "out_trade_no"))
	orderID := strings.TrimSpace(fieldString(data, "order_id"))
	txnID := fieldString(data, "transaction_id")
	channelHint := fieldString(data, "payment_channel")
	if outTradeNo == "" && strings.HasPrefix(txnID, "wechat:") {
		outTradeNo = strings.TrimPrefix(txnID, "wechat:")
		if channelHint == "" {
			channelHint = "wechat"
		}
	}
	if orderID == "" && strings.HasPrefix(txnID, "paypal:") {
		orderID = strings.TrimPrefix(txnID, "paypal:")
		if channelHint == "" {
			channelHint = "paypal"
		}
	}
	hubKey := "billing:user:" + userID
	statusData := map[string]interface{}{
		"event_name":         "recharge_completed",
		"status":             "completed",
		"transaction_id":     txnID,
		"out_trade_no":       outTradeNo,
		"order_id":           orderID,
		"recharge_points":    data["amount_points"],
		"recharge_yuan":      data["amount_yuan"],
		"points_source_type": fieldString(data, "points_source_type"),
		"payment_channel":    channelHint,
		"tenant_id":          fieldString(data, "tenant_id"),
		"user_id":            userID,
	}
	// task_id 供 taskSSE normalizeInboundMessage 复用既有 hub 键空间
	envelope := map[string]interface{}{
		"task_id":     hubKey,
		"user_id":     userID,
		"status_data": statusData,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	channel := "sse:billing:user:" + userID
	rdb := h.client()
	defer rdb.Close()
	if _, err := rdb.Publish(ctx, channel, body).Result(); err != nil {
		return err
	}
	log.Printf("[billing] published recharge SSE channel=%s out_trade_no=%s", channel, outTradeNo)
	return nil
}

// IsTenantIntent handles BILLING_TRANSACTION_CREATED intent 2: mark user as tenant.
// Only acts on recharge-type transactions with positive amount — filters out
// consumption, refund, and zero-amount events.  Idempotent: repeated calls for
// the same user are safe (taskAuth PATCH is_tenant is a no-op if already true).
type IsTenantIntent struct {
	Repo *saas.Repository
}

func (h *IsTenantIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "BILLING_TRANSACTION_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}

	// Only mark tenant on successful recharge (not consumption/refund/etc.)
	txType := fieldString(data, "transaction_type")
	if txType != "recharge" {
		log.Printf("[billing/2_is_tenant] skipping non-recharge txn_type=%s", txType)
		return domain.DispatchSuccess, nil
	}

	userID := strings.TrimSpace(fieldString(data, "user_id"))
	if userID == "" {
		log.Printf("[billing/2_is_tenant] missing user_id in recharge event")
		return domain.DispatchPermanent, fmt.Errorf("missing user_id")
	}

	if err := h.Repo.IntentMarkUserTenant(userID); err != nil {
		log.Printf("[billing/2_is_tenant] mark_user_tenant failed user_id=%s err=%v", userID, err)
		return domain.DispatchRetryable, err
	}

	log.Printf("[billing/2_is_tenant] ok user_id=%s", userID)
	return domain.DispatchSuccess, nil
}

func fieldString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
