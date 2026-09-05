package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"taskGitOauth/infrastructure"
)

var tenantEventTopicMap = map[string]string{
	"TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED": "tenant-gitlab-oauth-connection-upserted",
	"TENANT_GITLAB_OAUTH_CONNECTION_DELETED":  "tenant-gitlab-oauth-connection-deleted",
	"GIT_MERGE_REQUEST_MERGED":                "git-merge-request-merged",
	"PROJECT_GIT_OAUTH_GRANTED":               "project-git-oauth-granted",
	"COMMENT_GIT_OAUTH_GRANTED":               "comment-git-oauth-granted",
}

func publishDomainEvent(cfg *infrastructure.Config, eventType string, data map[string]any, key string) {
	bootstrap := ""
	if cfg != nil {
		bootstrap = strings.TrimSpace(cfg.KafkaBootstrapServers)
	}
	if bootstrap == "" {
		log.Printf("[taskGitOauth] kafka skip (no KAFKA_BOOTSTRAP_SERVERS) event=%s", eventType)
		return
	}
	topic, ok := tenantEventTopicMap[eventType]
	if !ok {
		topic = strings.ToLower(strings.ReplaceAll(eventType, "_", "-"))
	}
	payload, err := json.Marshal(map[string]any{
		"event_type": eventType,
		"data":       data,
	})
	if err != nil {
		log.Printf("[taskGitOauth] kafka marshal %s: %v", eventType, err)
		return
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(bootstrap),
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
	if err := w.WriteMessages(context.Background(), msg); err != nil {
		log.Printf("[taskGitOauth] kafka publish %s failed: %v", eventType, err)
	}
}

func publishTenantGitLabOAuthConnectionUpserted(cfg *infrastructure.Config, data map[string]any, key string) {
	publishDomainEvent(cfg, "TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED", data, key)
}

func publishTenantGitLabOAuthConnectionDeleted(cfg *infrastructure.Config, data map[string]any, key string) {
	publishDomainEvent(cfg, "TENANT_GITLAB_OAUTH_CONNECTION_DELETED", data, key)
}
