package main

import (
	"context"
	"testing"
)

func TestPublishDomainEventKafkaNotConfigured(t *testing.T) {
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })
	err := publishDomainEventKafka(context.Background(), "EMAIL_SENT", map[string]interface{}{"x": 1}, "k")
	if err != errKafkaNotConfigured {
		t.Fatalf("got %v", err)
	}
}

func TestEmailSentTopicMapping(t *testing.T) {
	if eventTopicMap["EMAIL_SENT"] != "email-sent" {
		t.Fatalf("topic=%q", eventTopicMap["EMAIL_SENT"])
	}
	if eventTopicMap["EMAIL_UNSUBSCRIBED"] != "email-unsubscribed" {
		t.Fatalf("unsub topic=%q", eventTopicMap["EMAIL_UNSUBSCRIBED"])
	}
	if eventTopicMap["EMAIL_RESUBSCRIBED"] != "email-resubscribed" {
		t.Fatalf("resub topic=%q", eventTopicMap["EMAIL_RESUBSCRIBED"])
	}
}

func TestPublishEmailSentSMTPFallback(t *testing.T) {
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })
	t.Setenv("KAFKA_BOOTSTRAP_SERVERS", "")
	t.Setenv("TASKAUTH_KAFKA_BOOTSTRAP_SERVERS", "")
	// Kafka 未配置时，应回退到 SMTP（SMTP 在测试环境不可用，返回错误）
	_, err := publishEmailSent(context.Background(), "a@b.c", "subj", "verification_code", map[string]interface{}{"code": "123456"})
	if err == nil {
		t.Fatal("expected error when both Kafka and SMTP are unavailable")
	}
	// 不应返回 errKafkaNotConfigured（因为有 SMTP 回退）
	if err == errKafkaNotConfigured {
		t.Fatal("expected SMTP fallback error, not errKafkaNotConfigured")
	}
}

func TestEventTopicMapRBACAuthz(t *testing.T) {
	if eventTopicMap["RoleChanged"] != "role-changed" {
		t.Fatalf("RoleChanged topic=%q", eventTopicMap["RoleChanged"])
	}
	if eventTopicMap["RoleResourceGroupsChanged"] != "role-resource-groups-changed" {
		t.Fatalf("RoleResourceGroupsChanged topic=%q", eventTopicMap["RoleResourceGroupsChanged"])
	}
	if eventTopicMap["PlatformRoleChanged"] != "platform-role-changed" {
		t.Fatalf("PlatformRoleChanged topic=%q", eventTopicMap["PlatformRoleChanged"])
	}
}
