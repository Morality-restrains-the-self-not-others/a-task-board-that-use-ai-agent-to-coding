package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// RBAC 事件发布 + 通用工具 (v63 design §7)
// 角色变更 → Kafka RoleChanged → taskEvents 缓存失效
// 发送失败不阻塞主流程（缓存 TTL 兜底收敛）
// ═══════════════════════════════════════════════════════════════

func randSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "r" + hex.EncodeToString([]byte(time.Now().String()))[:8]
	}
	return hex.EncodeToString(b)
}

func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}

// Authz Kafka event types (v63/v72). Topic fallback via publishDomainEventKafka.
const (
	eventTypeRoleChanged               = "RoleChanged"
	eventTypePlatformRoleChanged       = "PlatformRoleChanged"
	eventTypeRoleResourceGroupsChanged = "RoleResourceGroupsChanged"
)

// publishRoleChanged 发布 RoleChanged 事件（自定义角色增改删 → 受影响租户缓存失效）
func publishRoleChanged(companyID string) {
	publishAuthzEvent(eventTypeRoleChanged, map[string]any{
		"company_id": companyID,
		"at":         time.Now().Format(time.RFC3339),
	}, companyID)
}

// publishRoleResourceGroupsChanged 角色↔资源组绑定变更（与 RoleChanged 语义区分，便于审计）
func publishRoleResourceGroupsChanged(companyID, roleID string) {
	publishAuthzEvent(eventTypeRoleResourceGroupsChanged, map[string]any{
		"company_id": companyID,
		"role_id":    roleID,
		"at":         time.Now().Format(time.RFC3339),
	}, companyID)
}

// publishPlatformRoleChanged 平台角色分配/撤销 → 用户级缓存失效
func publishPlatformRoleChanged(userID string) {
	publishAuthzEvent(eventTypePlatformRoleChanged, map[string]any{
		"user_id": userID,
		"at":      time.Now().Format(time.RFC3339),
	}, userID)
}

// publishAuthzEvent 向 Kafka 发布授权相关事件（topic fallback: role-changed）
func publishAuthzEvent(eventType string, payload map[string]any, key string) {
	if err := publishDomainEventKafka(context.Background(), eventType, payload, key); err != nil {
		if err == errKafkaNotConfigured {
			log.Printf("[taskAuth] event=kafka_skip type=%s reason=not-configured", eventType)
			return
		}
		log.Printf("[taskAuth] event=kafka_publish status=error type=%s err=%v", eventType, err)
		return
	}
	log.Printf("[taskAuth] event=kafka_publish status=ok type=%s", eventType)
}
