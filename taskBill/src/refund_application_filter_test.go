package main

import (
	"context"
	"fmt"
	"testing"
)

// OPT-20260823-045 回归：管理端退款审批表头列过滤 — listRefundApplications 必须
// 支持 tenant_id / order_id / status 组合等值过滤，且不互相干扰。
func TestListRefundApplicationsFilterByTenantOrderStatus(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	insertRefundApp := func(t *testing.T, tenantID, orderID int64, status string) int64 {
		t.Helper()
		res, err := db.Exec(`
			INSERT INTO billing_refund_application (
				tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
				reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at
			) VALUES (?, ?, ?, 100, ?, 'test reason', '', '', '[]', ?, NULL, ?, ?)`,
			tenantID, tenantID, fmt.Sprintf("user-%d", tenantID), status, orderID, utcNow(), utcNow(),
		)
		if err != nil {
			t.Fatalf("insert billing_refund_application: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("last insert id: %v", err)
		}
		return id
	}

	// 三个申请：租户 11/22，订单 111/222/333，状态 pending/approved/rejected
	_ = insertRefundApp(t, 11, 111, "pending")
	_ = insertRefundApp(t, 22, 222, "approved")
	_ = insertRefundApp(t, 11, 333, "rejected")

	ctx := context.Background()

	// 仅按租户过滤
	rows, err := listRefundApplications(ctx, refundListFilter{TenantID: 11})
	if err != nil {
		t.Fatalf("list by tenant: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("tenant=11 rows = %d, want 2", len(rows))
	}

	// 仅按订单过滤
	rows, err = listRefundApplications(ctx, refundListFilter{OrderID: 222})
	if err != nil {
		t.Fatalf("list by order: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("order=222 rows = %d, want 1", len(rows))
	}
	if rows[0]["tenant_id"] != formatID(22) {
		t.Fatalf("order=222 tenant = %v, want %s", rows[0]["tenant_id"], formatID(22))
	}

	// 租户 + 订单 组合（交集）
	rows, err = listRefundApplications(ctx, refundListFilter{TenantID: 11, OrderID: 111})
	if err != nil {
		t.Fatalf("list by tenant+order: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("tenant=11+order=111 rows = %d, want 1", len(rows))
	}

	// 租户 + 订单 组合不命中（租户 11 无订单 222）
	rows, err = listRefundApplications(ctx, refundListFilter{TenantID: 11, OrderID: 222})
	if err != nil {
		t.Fatalf("list tenant+order mismatch: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("tenant=11+order=222 rows = %d, want 0", len(rows))
	}

	// 仅按状态过滤
	rows, err = listRefundApplications(ctx, refundListFilter{Status: "pending"})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("status=pending rows = %d, want 1", len(rows))
	}

	// 无过滤返回全部
	rows, err = listRefundApplications(ctx, refundListFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("all rows = %d, want 3", len(rows))
	}
}
