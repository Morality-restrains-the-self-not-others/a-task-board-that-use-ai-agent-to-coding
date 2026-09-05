package main

import (
	"database/sql"
	"testing"
)

// OPT-20260819-031 回归：refundApplicationJSON 序列化必须输出 frozen_yuan
// 元字符串（与 frozen_points 分一致），前端无需再做分→元换算。
func TestRefundApplicationJSONIncludesFrozenYuan(t *testing.T) {
	app := &RefundApplication{
		ID:              123,
		TenantID:        456,
		AccountID:       789,
		ApplicantUserID: "user-1",
		FrozenPoints:    55,
		Status:          "pending",
		Reason:          "reasons",
		ReviewerUserID:  "",
		ReviewNote:      "",
		PaymentRefundRefs: "",
		OrderID:          sql.NullInt64{},
		ReviewedAt:       sql.NullString{},
		CreatedAt:        "2026-08-19 10:00:00",
		UpdatedAt:        "2026-08-19 10:00:00",
	}
	m := refundApplicationJSON(app)
	if m["frozen_points"] != int64(55) {
		t.Fatalf("frozen_points = %v, want 55", m["frozen_points"])
	}
	if m["frozen_yuan"] != "0.55" {
		t.Fatalf("frozen_yuan = %v, want 0.55", m["frozen_yuan"])
	}
}

// 整数元不丢精度：100 分 → 1.00。
func TestRefundApplicationJSONFrozenYuanInteger(t *testing.T) {
	app := &RefundApplication{ID: 1, TenantID: 2, AccountID: 3, ApplicantUserID: "u", FrozenPoints: 100, Status: "pending", CreatedAt: "2026-08-19 10:00:00", UpdatedAt: "2026-08-19 10:00:00"}
	m := refundApplicationJSON(app)
	if m["frozen_yuan"] != "1.00" {
		t.Fatalf("frozen_yuan = %v, want 1.00", m["frozen_yuan"])
	}
}
