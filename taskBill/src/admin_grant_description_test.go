package main

import (
	"context"
	"testing"
)

func TestAdminGrantWritesConcreteResourceDescription(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9400000001)
	_, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 10},
	}, "admin-1", "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	var desc, src string
	if err := db.QueryRow(`
		SELECT description, points_source_type
		FROM billing_transaction
		WHERE account_id = (SELECT id FROM billing_account WHERE tenant_id = ?)
		  AND transaction_type = 'recharge'
		ORDER BY created_at DESC LIMIT 1`, tenantID).Scan(&desc, &src); err != nil {
		t.Fatalf("query txn: %v", err)
	}
	if src != "admin_grant" {
		t.Fatalf("points_source_type=%q want admin_grant", src)
	}
	if desc != "管理员后台赠送：任务帖 +10 帖" {
		t.Fatalf("description=%q", desc)
	}
}

func TestAdminGrantWritesMultiResourceDescription(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9400000002)
	_, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 2, Reason: "补偿"},
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 8, Region: "tencent-sh-1"},
	}, "admin-1", "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	var desc string
	if err := db.QueryRow(`
		SELECT description FROM billing_transaction
		WHERE account_id = (SELECT id FROM billing_account WHERE tenant_id = ?)
		ORDER BY created_at DESC LIMIT 1`, tenantID).Scan(&desc); err != nil {
		t.Fatalf("query txn: %v", err)
	}
	want := "管理员后台赠送：任务帖 +2 帖；GitLab 磁盘 +8 GB（tencent-sh-1）（补偿）"
	if desc != want {
		t.Fatalf("description=%q want %q", desc, want)
	}
}
