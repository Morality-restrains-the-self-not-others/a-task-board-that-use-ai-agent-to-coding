package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

func publishProjectGitOAuthGranted(projectID, userID, gitsite, remoteUserID string) {
	log.Printf("[taskProjectService] PROJECT_GIT_OAUTH_GRANTED project_id=%s user_id=%s gitsite=%s remote_user_id=%s",
		strings.TrimSpace(projectID), strings.TrimSpace(userID), strings.TrimSpace(gitsite), strings.TrimSpace(remoteUserID))
}

var eventTopicMap = map[string]string{
	"PROJECT_REVISION_RECORDED": "project-revision-recorded",
	"PROJECT_DELETED":           "project-deleted",
	"PROJECT_GIT_OAUTH_GRANTED": "project-git-oauth-granted",
}

var publishProjectRevisionRecordedFn = publishProjectRevisionRecorded
var publishProjectDeletedFn = publishProjectDeleted

func publishDomainEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		return nil
	}
	topic, ok := eventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	if ctx == nil {
		ctx = context.Background()
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
		log.Printf("[taskProjectService] kafka publish %s failed: %v", eventType, err)
		return err
	}
	tracelog.LogEventPublish(ctx, eventType, topic, map[string]any{
		"project_id": strField(data, "project_id"),
	})
	return nil
}

func publishProjectRevisionRecorded(
	ctx context.Context,
	tenantID, projectID, revisionID string,
	versionNum int,
	actorUserID, changedFields string,
) error {
	data := map[string]interface{}{
		"tenant_id":      tenantID,
		"project_id":     projectID,
		"revision_id":    revisionID,
		"version_num":    versionNum,
		"actor_user_id":  actorUserID,
		"changed_fields": changedFields,
	}
	return publishDomainEvent(ctx, "PROJECT_REVISION_RECORDED", data, projectID)
}

func publishProjectDeleted(ctx context.Context, tenantID, projectID string) error {
	data := map[string]interface{}{
		"tenant_id":  tenantID,
		"project_id": projectID,
	}
	return publishDomainEvent(ctx, "PROJECT_DELETED", data, projectID)
}
