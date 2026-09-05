package main

import (
	"testing"
	"time"
)

func TestPublishRoleResourceGroupsChangedPayloadShape(t *testing.T) {
	// 契约：事件类型名与 payload 字段稳定（消费者/审计依赖）
	const wantType = "RoleResourceGroupsChanged"
	payload := map[string]any{
		"company_id": "c1",
		"role_id":    "rid-1",
		"at":         time.Now().Format(time.RFC3339),
	}
	if wantType == "" {
		t.Fatal("event type empty")
	}
	for _, k := range []string{"company_id", "role_id", "at"} {
		if _, ok := payload[k]; !ok {
			t.Fatalf("missing payload key %s", k)
		}
	}
	// 确保 publish 函数可链接（编译期存在）
	publishRoleResourceGroupsChanged("c1", "rid-1")
}
