package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

var eventTopicMap = map[string]string{
	"CLOUD_PLATFORM_AUTHORIZATION_CREATED": "cloud-platform-authorization-created",
	"CommentContainerBindingAdvanced":      "comment-container-binding-advanced",
	"LayerGraphSnapshotPersisted":          "layer-graph-snapshot-persisted",
	"JobStepFullArchived":                  "job-step-full-archived",
	"CommentStartupLogArchived":            "comment-startup-log-archived",
	"StepFullCOSConfigUpdated":             "step-full-cos-config-updated",
	"CONTAINER_INSTRUCTION_IDLE_MARKED":    "container-instruction-idle-marked",
	"CONTAINER_INSTRUCTION_IDLE_CLEARED":   "container-instruction-idle-cleared",
}

func publishDomainEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		return nil
	}
	topic, ok := eventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	data = tracelog.EnsureTraceInData(ctx, data)
	payload, err := json.Marshal(map[string]interface{}{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		return err
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBootstrapServers),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer w.Close()
	msg := kafka.Message{Value: payload}
	if key != "" {
		msg.Key = []byte(key)
	}
	if err := w.WriteMessages(ctx, msg); err != nil {
		log.Printf("[taskCloudService] kafka publish %s failed: %v", eventType, err)
		return err
	}
	tracelog.LogEventPublish(ctx, eventType, topic, map[string]any{
		"task_id":  strField(data, "task_id"),
		"event_id": strField(data, "event_id"),
	})
	return nil
}

func publishSSEMessage(ctx context.Context, taskID string, statusData map[string]interface{}) error {
	if strings.TrimSpace(taskID) == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		// 心跳等入站在 TAS/客户端超时断开后 r.Context 会取消；SSE 面向浏览器订阅者，
		// 必须仍能写入 Kafka，否则任务详情「容器连接」会永久停在 idle。
		ctx = context.WithoutCancel(ctx)
	}
	if statusData == nil {
		statusData = map[string]interface{}{}
	}
	if tid := tracelog.TraceIDFromContext(ctx); tid != "" {
		if _, ok := statusData["trace_id"]; !ok {
			statusData["trace_id"] = tid
		}
		tracelog.EnsureTraceCorrelationInMap(statusData, tid)
	}
	return publishDomainEvent(ctx, "SSE_MESSAGE", map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	}, taskID)
}

// publishTaskSSE is like publishSSEMessage but injects comment_id into the SSE
// payload when non-empty, so the frontend can route events to per-comment panels.
// 同时 best-effort 将服务器调度进度写入 comment binding 启动日志（冷打开可还原）。
func publishTaskSSE(ctx context.Context, taskID, commentID string, statusData map[string]interface{}) error {
	taskID = strings.TrimSpace(taskID)
	commentID = strings.TrimSpace(commentID)
	if statusData == nil {
		statusData = map[string]interface{}{}
	}
	if commentID != "" {
		statusData["comment_id"] = commentID
		tid := strField(statusData, "trace_id")
		if tid == "" {
			tid = tracelog.TraceIDFromContext(ctx)
		}
		persistCommentBindingStartTraceID(taskID, commentID, tid)
		logServerSchedulingToBindingBestEffort(taskID, commentID, statusData)
	} else {
		// auto_run / 任务级 start-vm 无 comment_id：扇出调度文案，但不复制同一 trace_id
		logServerSchedulingToActiveBindingsBestEffort(taskID, statusData)
	}
	return publishSSEMessage(ctx, taskID, statusData)
}

func publishCloudPlatformAuthorizationCreated(authID, platformType, authType, secretID, secretKey, companyID string) {
	if authType != "access_key" {
		return
	}
	_ = publishDomainEvent(context.Background(), "CLOUD_PLATFORM_AUTHORIZATION_CREATED", map[string]interface{}{
		"id":                 authID,
		"platform_type":      platformType,
		"authorization_type": authType,
		"secret_id":          secretID,
		"secret_key":         secretKey,
		"company_id":         companyID,
	}, authID)
}
