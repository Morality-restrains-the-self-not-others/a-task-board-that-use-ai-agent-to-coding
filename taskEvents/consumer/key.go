package consumer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"taskEvents/domain"
)

// IdempotencyKeyFromEnvelope derives a business idempotency key from event data.
func IdempotencyKeyFromEnvelope(env domain.EventEnvelope) domain.IdempotencyKey {
	if env.EventType == "SSE_MESSAGE" {
		return idempotencyKeyForSSEMessage(env)
	}
	if env.EventType == "CLOUD_SERVER_STOPPED" {
		return idempotencyKeyForCloudServerStopped(env)
	}
	if env.EventType == "CLOUD_SERVER_STARTED" || env.EventType == "CLOUD_SERVER_START_AUTO" {
		return idempotencyKeyForCloudServerStart(env)
	}
	if env.EventType == "TASK_STATUS_CHANGED" {
		return idempotencyKeyForTaskStatusChanged(env)
	}
	// TASK_CREATED / TASK_DELETED always carry user_id; keying by user_id would collapse
	// every create/delete from the same actor into one delivery (work-panel SSE silent).
	if env.EventType == "TASK_CREATED" || env.EventType == "TASK_DELETED" {
		return idempotencyKeyForTaskLifecycle(env)
	}
	if env.EventType == "TASK_REVISION_RECORDED" || env.EventType == "PROJECT_REVISION_RECORDED" {
		return idempotencyKeyForRevisionRecorded(env)
	}
	if env.EventType == "PROJECT_DELETED" {
		return idempotencyKeyForProjectDeleted(env)
	}
	// CLOUD_PLATFORM_AUTHORIZATION_CREATED carries company_id but payload id is the
	// auth record grain; company_id would collapse all authorizations in a tenant.
	if env.EventType == "CLOUD_PLATFORM_AUTHORIZATION_CREATED" {
		return idempotencyKeyForCloudPlatformAuthorization(env)
	}
	// BILLING_ORDER_COMMENT_CREATED is published with an empty Kafka key and payload
	// comment_id; without a dedicated key all order-comment events collapse to one.
	if env.EventType == "BILLING_ORDER_COMMENT_CREATED" {
		return idempotencyKeyForBillingOrderCommentCreated(env)
	}
	if env.EventType == "TASK_COMMENT_IMAGE_MENTIONED" {
		return idempotencyKeyForTaskCommentImageMentioned(env)
	}
	// GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED (taskBill) 只读 ops 告警：必须以 order_id
	// 为幂等粒（重放同 order_id 只告警一次）；禁止用 tenant_id —— 同租户多订单会互相吞掉。
	if env.EventType == "GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED" {
		return idempotencyKeyForGitlabManualNodeFulfillmentQueued(env)
	}
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	// Prefer task_id over user_id/company_id: the latter collapse per-actor / per-tenant deliveries.
	for _, field := range []string{"event_id", "transaction_id", "task_id", "user_id", "company_id"} {
		if v, ok := data[field]; ok && v != nil && formatID(v) != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, formatID(v))
		}
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// TASK_CREATED / TASK_DELETED are one-shot per task_id (not per creating user).
func idempotencyKeyForTaskLifecycle(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
	}
	if v, ok := data["event_id"]; ok && v != nil {
		if id := formatID(v); id != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, id)
		}
	}
	if taskID := formatID(data["task_id"]); taskID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// TASK_STATUS_CHANGED repeats for the same task across column/completed transitions.
// Key by task_id + status fingerprint so later terminals are not swallowed after the first.
func idempotencyKeyForTaskStatusChanged(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
	}
	if v, ok := data["event_id"]; ok && v != nil {
		if id := formatID(v); id != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, id)
		}
	}
	taskID := formatID(data["task_id"])
	fingerprint := map[string]interface{}{
		"progress_column_id":          formatID(data["progress_column_id"]),
		"previous_progress_column_id": formatID(data["previous_progress_column_id"]),
		"completed":                   data["completed"],
		"previous_completed":          data["previous_completed"],
		"progress_column_name":        formatID(data["progress_column_name"]),
		"terminal_kind":               formatID(data["terminal_kind"]),
	}
	statusBytes, err := json.Marshal(fingerprint)
	if err != nil {
		if taskID != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, taskID)
		}
		return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
	}
	sum := sha256.Sum256(statusBytes)
	digest := hex.EncodeToString(sum[:8])
	if taskID == "" {
		return domain.IdempotencyKeyForEvent(env.EventType, digest)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, taskID+":"+digest)
}

// Start flows persist a cloud_server_events row per request; event_id is the correct grain.
// Never key by company_id (tenant-wide collapse). Missing event_id falls back to task_id:trace_id.
func idempotencyKeyForCloudServerStart(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	if v, ok := data["event_id"]; ok && v != nil {
		if id := formatID(v); id != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, id)
		}
	}
	taskID := formatID(data["task_id"])
	traceID := formatID(data["trace_id"])
	if taskID != "" && traceID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID+":"+traceID)
	}
	if taskID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// CLOUD_PLATFORM_AUTHORIZATION_CREATED payload carries company_id plus auth record id;
// key by the auth record id (never company_id — tenant-wide collapse).
func idempotencyKeyForCloudPlatformAuthorization(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	for _, field := range []string{"id", "auth_id", "authorization_id"} {
		if v, ok := data[field]; ok && v != nil {
			if id := formatID(v); id != "" {
				return domain.IdempotencyKeyForEvent(env.EventType, id)
			}
		}
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// BILLING_ORDER_COMMENT_CREATED is published with an empty Kafka key; key by comment_id
// so each order comment is an independent delivery (fallback order_id, then envelope key).
func idempotencyKeyForBillingOrderCommentCreated(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	for _, field := range []string{"comment_id", "order_id"} {
		if v, ok := data[field]; ok && v != nil {
			if id := formatID(v); id != "" {
				return domain.IdempotencyKeyForEvent(env.EventType, id)
			}
		}
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// CLOUD_SERVER_STOPPED carries company_id on every publish; using it as the key would
// silently skip all subsequent stops in the same tenant after the first success.
func idempotencyKeyForCloudServerStopped(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	if v, ok := data["stop_request_id"]; ok && v != nil {
		if id := formatID(v); id != "" {
			return domain.IdempotencyKeyForEvent(env.EventType, id)
		}
	}
	taskID := formatID(data["task_id"])
	instanceID := formatID(data["instance_id"])
	if taskID != "" && instanceID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID+":"+instanceID)
	}
	if taskID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// SSE progress updates share task_id but are distinct deliveries; key by status payload hash.
func idempotencyKeyForSSEMessage(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
	}
	taskID := formatID(data["task_id"])
	statusRaw, ok := data["status_data"]
	if !ok || statusRaw == nil {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	statusBytes, err := json.Marshal(statusRaw)
	if err != nil {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	sum := sha256.Sum256(statusBytes)
	suffix := taskID + ":" + hex.EncodeToString(sum[:8])
	return domain.IdempotencyKeyForEvent(env.EventType, suffix)
}

// TASK_COMMENT_IMAGE_MENTIONED is one start-vm intent per @mention comment.
// Default keying by task_id collapses later comments on the same task (idempotency skip).
func idempotencyKeyForTaskCommentImageMentioned(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	for _, field := range []string{"comment_id", "parent_comment_id", "event_id"} {
		if v, ok := data[field]; ok && v != nil {
			if id := formatID(v); id != "" {
				return domain.IdempotencyKeyForEvent(env.EventType, id)
			}
		}
	}
	if taskID := formatID(data["task_id"]); taskID != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, taskID)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// TASK_REVISION_RECORDED / PROJECT_REVISION_RECORDED must key by revision_id.
// Payload also carries task_id/project_id; using those would collapse later versions.
func idempotencyKeyForRevisionRecorded(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	if id := formatID(data["revision_id"]); id != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, id)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

func idempotencyKeyForProjectDeleted(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	if id := formatID(data["project_id"]); id != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, id)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

// GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED payload 携带 tenant_id + order_id；
// key 必须以 order_id（taskBill 亦以 order_id 作为 Kafka key），否则同租户后续订单
// 的告警会在首个成功后全部被幂等跳过。
func idempotencyKeyForGitlabManualNodeFulfillmentQueued(env domain.EventEnvelope) domain.IdempotencyKey {
	var data map[string]interface{}
	_ = json.Unmarshal(env.Data, &data)
	if id := formatID(data["order_id"]); id != "" {
		return domain.IdempotencyKeyForEvent(env.EventType, id)
	}
	return domain.IdempotencyKeyForEvent(env.EventType, env.Key)
}

func formatID(v interface{}) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}
