package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// OPT-20260903-012: 到期日从管理员开通日起算。
// 首次购买（pending_admin）支付时不写 disk_expires_at；已开通续费按现有到期时间顺延。

func TestMarkOrderPaidGitlabDiskRenewalExtendsExistingExpiry(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9610000121
	seedVIP1Membership(t, tenantID)
	ctx := context.Background()
	region := "tencent-sh-1"
	now := utcNow()
	oldExpiry := diskExpiresAtFromMonths(1, "") // ~now+1月，模拟已开通资源

	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			provisioning_status, created_at, updated_at
		) VALUES (?, ?, 10, 0, 1, ?, 'active', ?, ?)`,
		tenantID, region, oldExpiry, now, now,
	); err != nil {
		t.Fatalf("seed active row: %v", err)
	}

	// 已开通租户加购 2 个月：状态保持 active，到期日在现有到期时间上顺延 2 个月
	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: region, DiskMonths: 2},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-gl-disk-renew", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	var diskGB, diskMonths int64
	var expiresAt, status string
	if err := db.QueryRow(`
		SELECT disk_gb, disk_months, disk_expires_at, provisioning_status
		FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, region,
	).Scan(&diskGB, &diskMonths, &expiresAt, &status); err != nil {
		t.Fatalf("scan resource row: %v", err)
	}
	if diskGB != GitlabDiskMinPurchaseGB+10 {
		t.Fatalf("disk_gb=%d want %d", diskGB, GitlabDiskMinPurchaseGB+10)
	}
	if status != "active" {
		t.Fatalf("provisioning_status=%q want active（续费不被打回待开通）", status)
	}
	oldT, err := time.Parse("2006-01-02 15:04:05.000000", oldExpiry)
	if err != nil {
		t.Fatalf("parse old expiry: %v", err)
	}
	newT, err := time.Parse("2006-01-02 15:04:05.000000", expiresAt)
	if err != nil {
		t.Fatalf("parse new expiry %q: %v", expiresAt, err)
	}
	lo := oldT.AddDate(0, 2, -4)
	hi := oldT.AddDate(0, 2, 4)
	if newT.Before(lo) || newT.After(hi) {
		t.Fatalf("renewed expiry %v not ~old(%v)+2months", newT, oldT)
	}
}

func TestGitlabProvisionExpiresAtHelper(t *testing.T) {
	// 已有到期时间（历史旧逻辑已计时/续费保留）→ 沿用
	existing := "2026-12-31 00:00:00.000000"
	if got := gitlabProvisionExpiresAt(2, existing); got != existing {
		t.Fatalf("existing=%q got %q want preserved", existing, got)
	}
	// 空到期 + disk_months>0 → now+months（开通日起算）
	got := gitlabProvisionExpiresAt(2, "")
	if got == "" {
		t.Fatal("empty expiry with months=2 must compute now+2months")
	}
	exp, err := time.Parse("2006-01-02 15:04:05.000000", got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	now := time.Now().UTC()
	lo := now.AddDate(0, 2, -4)
	hi := now.AddDate(0, 2, 4)
	if exp.Before(lo) || exp.After(hi) {
		t.Fatalf("provision expiry %v not ~now+2months", exp)
	}
	// 无购买月数（管理员赠送等）→ 无到期日
	if got := gitlabProvisionExpiresAt(0, ""); strings.TrimSpace(got) != "" {
		t.Fatalf("months=0 expiry=%q want empty", got)
	}
}
