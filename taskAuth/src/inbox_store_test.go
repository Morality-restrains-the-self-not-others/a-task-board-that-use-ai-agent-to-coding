package main

import (
	"strings"
	"testing"
	"time"

	"taskAuth/domain"
)

func TestInboxMessageJSONStripsEmbeddedReason(t *testing.T) {
	reason := "排查线上工单问题"
	row := inboxMessageRow{
		ID:        1,
		Kind:      "impersonation_notice",
		Title:     "系统管理员以你的身份登录",
		Body:      domain.ImpersonationNoticeBody("admin") + "\n理由：" + reason,
		Reason:    reason,
		CreatedAt: time.Date(2026, 8, 24, 15, 22, 13, 0, time.UTC),
	}
	item := inboxMessageJSON(row)
	body, _ := item["body"].(string)
	if strings.Contains(body, "理由") {
		t.Fatalf("API body still embeds reason: %q", body)
	}
	if item["reason"] != reason {
		t.Fatalf("reason=%v", item["reason"])
	}
}
