package main

// OPT-20260819-014: 交易流水账按 region 回放 GitLab 磁盘/流量瞬时配额剩余。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// seedGitlabRegionForReplay 创建区域元数据 + 租户资源 + 可选赠送批次。
func seedGitlabRegionForReplay(t *testing.T, tenantID int64, region string, diskGB, trafficGB, diskUsedBytes int64, trafficUsedGB float64) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 99, 500, 5000, 0, 0, 'test-token', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = VALUES(updated_at)`,
		generateSnowflakeID(), "测试区 "+region, region, now, now,
	); err != nil {
		t.Fatalf("seed region %s: %v", region, err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, 1, '2099-12-31', ?, ?, 'active', ?, ?)
		ON DUPLICATE KEY UPDATE
			disk_gb = VALUES(disk_gb), traffic_prepaid_gb = VALUES(traffic_prepaid_gb),
			disk_used_bytes = VALUES(disk_used_bytes), traffic_used_gb = VALUES(traffic_used_gb),
			updated_at = VALUES(updated_at)`,
		tenantID, region, diskGB, trafficGB, diskUsedBytes, trafficUsedGB, now, now,
	); err != nil {
		t.Fatalf("seed tenant resource %s: %v", region, err)
	}
}

// seedGitlabDiskPurchase 创建一笔 gitlab_disk 购买订单 + resource_purchase 流水。
func seedGitlabDiskPurchase(t *testing.T, tenantID int64, accID int64, region string, qty int64, createdAt string) (orderID int64) {
	t.Helper()
	orderID = generateSnowflakeID()
	orderNum := "OPT-ORDER-DISK-" + region
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents, payment_method, created_at, paid_at
		) VALUES (?, ?, ?, 'paid', 100, 'wechat', ?, ?)`,
		orderID, tenantID, orderNum, createdAt, createdAt,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_resource_order_item (
			id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, created_at
		) VALUES (?, ?, 'gitlab_disk', ?, 100, 100, ?, ?)`,
		generateSnowflakeID(), orderID, qty, region, createdAt,
	); err != nil {
		t.Fatalf("insert order item: %v", err)
	}
	tid := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, description, transaction_id, created_at, usage_amount, related_order_id
		) VALUES (?, ?, 'consumption', 0, 0, 0, 'resource_purchase', '购买 GitLab 磁盘', ?, ?, 0, ?)`,
		tid, accID, "order:"+orderNum, createdAt, orderID,
	); err != nil {
		t.Fatalf("insert purchase txn: %v", err)
	}
	return orderID
}

// seedGitlabTrafficGrant 创建一笔 gitlab_traffic 后台赠送流水（带 FK）。
func seedGitlabTrafficGrant(t *testing.T, tenantID int64, accID int64, region string, qty int64, createdAt string) {
	t.Helper()
	tid := generateSnowflakeID()
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, user_id, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'recharge', 0, 0, 0, 'admin_grant', 'user-1', '管理员后台赠送：GitLab 流量 +3 GB', ?, ?, 0)`,
		tid, accID, "admin_grant:"+formatID(tenantID)+":"+formatID(generateSnowflakeID()), createdAt,
	); err != nil {
		t.Fatalf("insert grant txn: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (
			id, tenant_id, resource_type, quantity, remaining, reason, expires_at, created_at,
			billing_transaction_id, source_kind, region
		) VALUES (?, ?, 'gitlab_traffic', ?, ?, '', '2099-12-31 23:59:59', ?, ?, 'gift', ?)`,
		generateSnowflakeID(), tenantID, qty, qty, createdAt, tid, region,
	); err != nil {
		t.Fatalf("insert grant row: %v", err)
	}
}

// TestTransactionsListGitlabRegionRemainingReplay 校验购买 + 赠送两行均回放出区域配额剩余。
func TestTransactionsListGitlabRegionRemainingReplay(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9410000006)
	const region = "tencent-shanghai-5"
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}

	// 当前配额：磁盘 5 GB（已用 1 GB），流量 10 GB（已用 2 GB）
	seedGitlabRegionForReplay(t, tenantID, region, 5, 10, int64(1.0*1024*1024*1024), 2.0)

	// 时间倒序：购买较新，赠送较旧
	grantTime := "2026-08-01 10:00:00"
	purchaseTime := "2026-08-02 10:00:00"
	seedGitlabTrafficGrant(t, tenantID, acc.ID, region, 3, grantTime)
	seedGitlabDiskPurchase(t, tenantID, acc.ID, region, 1, purchaseTime)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000006/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("len=%d body=%s", len(list), rr.Body.String())
	}

	// 最新行（购买）应含磁盘剩余；旧行（赠送）应含流量剩余。
	// 回放起点：磁盘 5 − 已用 1 = 4；流量 10 − 已用 2 = 8。
	purchaseRow := list[0]
	grantRow := list[1]

	snap, _ := purchaseRow["ledger_snapshot"].(map[string]interface{})
	purchaseDisplay, _ := snap["display"].(string)
	if !strings.Contains(purchaseDisplay, "GitLab 磁盘剩余 4 GB（"+region+"）") {
		t.Fatalf("purchase display=%q want GitLab 磁盘剩余 4 GB（%s）", purchaseDisplay, region)
	}

	snap2, _ := grantRow["ledger_snapshot"].(map[string]interface{})
	grantDisplay, _ := snap2["display"].(string)
	if !strings.Contains(grantDisplay, "GitLab 流量剩余 8 GB（"+region+"）") {
		t.Fatalf("grant display=%q want GitLab 流量剩余 8 GB（%s）", grantDisplay, region)
	}

	// 独立字段也须存在
	if rem, ok := purchaseRow["gitlab_region_remaining"].([]interface{}); !ok || len(rem) == 0 {
		t.Fatalf("purchase gitlab_region_remaining=%v", purchaseRow["gitlab_region_remaining"])
	}
	if rem, ok := grantRow["gitlab_region_remaining"].([]interface{}); !ok || len(rem) == 0 {
		t.Fatalf("grant gitlab_region_remaining=%v", grantRow["gitlab_region_remaining"])
	}
}

// TestTransactionsListGitlabRegionRemainingNoRows 无 GitLab 资源时流水不追加额外行。
func TestTransactionsListGitlabRegionRemainingNoRows(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID = int64(9410000007)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES (?, ?, 'consumption', 30, 100, 70, 'consumption', ?, 1, '启动消耗', 'txn-ledger-cash', ?)`,
		generateSnowflakeID(), acc.ID, unitID, utcNow(),
	); err != nil {
		t.Fatalf("insert txn: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9410000007/billing/transactions/", nil)
	rr := httptest.NewRecorder()
	handleTransactionsList(rr, req, false)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d body=%s", len(list), rr.Body.String())
	}
	if _, ok := list[0]["gitlab_region_remaining"]; ok {
		t.Fatalf("unexpected gitlab_region_remaining=%v", list[0]["gitlab_region_remaining"])
	}
	snap, _ := list[0]["ledger_snapshot"].(map[string]interface{})
	display, _ := snap["display"].(string)
	if strings.Contains(display, "GitLab") {
		t.Fatalf("unexpected GitLab in display=%q", display)
	}
}
