package main

import (
	"context"
	"testing"
)

// OPT-20260823-045 回归：管理端开票申请表头列过滤 — listInvoiceApplications 必须
// 支持 id / tenant_id / order_id / buyer_name / status 组合过滤，且不互相干扰。
func TestListInvoiceApplicationsFilterByColumns(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	insertInvoiceApp := func(t *testing.T, id, tenantID, orderID int64, buyerName, status string) {
		t.Helper()
		_, err := db.Exec(`
			INSERT INTO billing_invoice_application (
				id, tenant_id, order_id, applicant_user_id, buyer_type, buyer_name, status,
				reviewer_user_id, review_note, reviewed_at, created_at, updated_at
			) VALUES (?, ?, ?, 'u1', 'ORGANIZATION', ?, ?, '', '', NULL, ?, ?)`,
			id, tenantID, orderID, buyerName, status, utcNow(), utcNow(),
		)
		if err != nil {
			t.Fatalf("insert billing_invoice_application: %v", err)
		}
	}

	// 三个申请：租户 11/22，订单 111/222/333，状态 pending/approved/rejected，抬头名不同。
	insertInvoiceApp(t, 1001, 11, 111, "张三科技有限公司", "pending")
	insertInvoiceApp(t, 1002, 22, 222, "李四网络", "approved")
	insertInvoiceApp(t, 1003, 11, 333, "张三个人", "rejected")

	ctx := context.Background()

	// 仅按租户过滤
	rows, err := listInvoiceApplications(ctx, invoiceListFilter{TenantID: 11})
	if err != nil {
		t.Fatalf("list by tenant: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("tenant=11 rows = %d, want 2", len(rows))
	}

	// 仅按订单过滤
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{OrderID: 222})
	if err != nil {
		t.Fatalf("list by order: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("order=222 rows = %d, want 1", len(rows))
	}
	if rows[0]["tenant_id"] != formatID(22) {
		t.Fatalf("order=222 tenant = %v, want %s", rows[0]["tenant_id"], formatID(22))
	}

	// 按申请 ID 精确过滤
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{ID: 1001})
	if err != nil {
		t.Fatalf("list by id: %v", err)
	}
	if len(rows) != 1 || rows[0]["tenant_id"] != formatID(11) {
		t.Fatalf("id filter rows = %v, want 1 tenant=%s", rows, formatID(11))
	}

	// 按抬头名称模糊过滤（张三个人 + 张三科技有限公司）
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{BuyerName: "张三"})
	if err != nil {
		t.Fatalf("list by buyer_name: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("buyer_name=张三 rows = %d, want 2", len(rows))
	}

	// 租户 + 订单组合（交集）
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{TenantID: 11, OrderID: 111})
	if err != nil {
		t.Fatalf("list tenant+order: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("tenant=11+order=111 rows = %d, want 1", len(rows))
	}

	// 租户 + 订单组合不命中（租户 11 无订单 222）
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{TenantID: 11, OrderID: 222})
	if err != nil {
		t.Fatalf("list tenant+order mismatch: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("tenant=11+order=222 rows = %d, want 0", len(rows))
	}

	// 状态 + 租户组合
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{TenantID: 11, Status: "rejected"})
	if err != nil {
		t.Fatalf("list tenant+status: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("tenant=11+status=rejected rows = %d, want 1", len(rows))
	}

	// 无过滤返回全部
	rows, err = listInvoiceApplications(ctx, invoiceListFilter{})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("all rows = %d, want 3", len(rows))
	}
}
