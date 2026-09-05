package main

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"
)

var orderNumberRe = regexp.MustCompile(`^ORD-\d{8}-\d+-\d+$`)

func TestGenerateOrderNumberEmbedsTenantAndSnowflakeID(t *testing.T) {
	const tenantID int64 = 877397588196749312
	const orderID int64 = 877596007691485184
	got, err := generateOrderNumber(tenantID, orderID)
	if err != nil {
		t.Fatalf("generateOrderNumber: %v", err)
	}
	wantDay := time.Now().UTC().Format("20060102")
	want := fmt.Sprintf("ORD-%s-%d-%d", wantDay, tenantID, orderID)
	if got != want {
		t.Fatalf("order_number = %q, want %q", got, want)
	}
	if !orderNumberRe.MatchString(got) {
		t.Fatalf("order_number %q does not match ORD-YYYYMMDD-tenant-id", got)
	}
	parsed, err := ParseResourceOrderNumber(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !parsed.HasTenant || parsed.TenantID != tenantID || parsed.OrderID != orderID {
		t.Fatalf("parsed = %+v, want tenant=%d id=%d", parsed, tenantID, orderID)
	}
}

func TestGenerateOrderNumberRejectsNonPositiveIDs(t *testing.T) {
	if _, err := generateOrderNumber(0, 1); err == nil {
		t.Fatal("expected error for tenantID=0")
	}
	if _, err := generateOrderNumber(-1, 1); err == nil {
		t.Fatal("expected error for tenantID<0")
	}
	if _, err := generateOrderNumber(1, 0); err == nil {
		t.Fatal("expected error for orderID=0")
	}
	if _, err := generateOrderNumber(1, -1); err == nil {
		t.Fatal("expected error for orderID<0")
	}
}

func TestParseResourceOrderNumberFormats(t *testing.T) {
	four, err := ParseResourceOrderNumber("ORD-20260818-877397588196749312-877596007691485184")
	if err != nil {
		t.Fatal(err)
	}
	if !four.HasTenant || four.TenantID != 877397588196749312 || four.OrderID != 877596007691485184 {
		t.Fatalf("four-segment parse: %+v", four)
	}
	legacy, err := ParseResourceOrderNumber("ORD-20260818-877596007691485184")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.HasTenant || legacy.OrderID != 877596007691485184 {
		t.Fatalf("three-segment ADR-0017 parse: %+v", legacy)
	}
	nnn, err := ParseResourceOrderNumber("ORD-20260807-001")
	if err != nil {
		t.Fatal(err)
	}
	if nnn.HasTenant || nnn.OrderID != 1 {
		t.Fatalf("NNN parse: %+v", nnn)
	}
	if _, err := ParseResourceOrderNumber("INV-20260818-1-2"); err == nil {
		t.Fatal("expected error for non-ORD prefix")
	}
}

func TestCreateOrderNumberUniqueAcrossTenantsContainsGene(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantA int64 = 9200000001
	const tenantB int64 = 9200000002
	oa, _, err := createOrder(context.Background(), tenantA, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("tenant A createOrder: %v", err)
	}
	ob, _, err := createOrder(context.Background(), tenantB, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("tenant B createOrder: %v", err)
	}
	if oa.OrderNumber == ob.OrderNumber {
		t.Fatalf("tenants shared order_number %q", oa.OrderNumber)
	}
	pa, err := ParseResourceOrderNumber(oa.OrderNumber)
	if err != nil || !pa.HasTenant || pa.TenantID != tenantA || pa.OrderID != oa.ID {
		t.Fatalf("tenant A number %q parse %+v id=%d err=%v", oa.OrderNumber, pa, oa.ID, err)
	}
	pb, err := ParseResourceOrderNumber(ob.OrderNumber)
	if err != nil || !pb.HasTenant || pb.TenantID != tenantB || pb.OrderID != ob.ID {
		t.Fatalf("tenant B number %q parse %+v id=%d err=%v", ob.OrderNumber, pb, ob.ID, err)
	}
}

func TestLoadOrderRequiresMatchingTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantA int64 = 9200000011
	const tenantB int64 = 9200000012
	oa, _, err := createOrder(context.Background(), tenantA, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if _, _, err := loadOrder(tenantB, oa.ID); err == nil {
		t.Fatal("wrong tenant loadOrder succeeded")
	}
	got, _, err := loadOrder(tenantA, oa.ID)
	if err != nil {
		t.Fatalf("same tenant loadOrder: %v", err)
	}
	if got.ID != oa.ID || got.TenantID != tenantA {
		t.Fatalf("loaded %+v", got)
	}
	byID, _, err := loadOrderByID(oa.ID)
	if err != nil {
		t.Fatalf("loadOrderByID: %v", err)
	}
	if byID.TenantID != tenantA {
		t.Fatalf("loadOrderByID tenant=%d", byID.TenantID)
	}
}

func TestMarkOrderPaidUsesRowTenantWhenCallerOmitsTenant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantID int64 = 9200000021
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`,
		generateSnowflakeID(), tenantID, utcNow(), utcNow()); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	oa, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(context.Background(), oa.ID, "wechat", "ref-omit-tenant", 0); err != nil {
		t.Fatalf("markOrderPaid tenantID=0: %v", err)
	}
	got, _, err := loadOrder(tenantID, oa.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != OrderStatusPaid {
		t.Fatalf("status=%s, want paid", got.Status)
	}
	var quota int64
	if err := db.QueryRow(`SELECT task_post_quota FROM billing_account WHERE tenant_id = ?`, tenantID).Scan(&quota); err != nil {
		t.Fatalf("quota: %v", err)
	}
	if quota != 1 {
		t.Fatalf("task_post_quota=%d, want 1 (credited to row tenant, not 0)", quota)
	}
}

func TestMarkOrderPaidWrongTenantFails(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const tenantA int64 = 9200000031
	const tenantB int64 = 9200000032
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?), (?, ?, 0, ?, ?)`,
		generateSnowflakeID(), tenantA, utcNow(), utcNow(),
		generateSnowflakeID(), tenantB, utcNow(), utcNow()); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}
	oa, _, err := createOrder(context.Background(), tenantA, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(context.Background(), oa.ID, "wechat", "ref-wrong", tenantB); err == nil {
		t.Fatal("wrong tenant markOrderPaid succeeded")
	}
	got, _, err := loadOrder(tenantA, oa.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != OrderStatusPending {
		t.Fatalf("status=%s, want pending", got.Status)
	}
}
