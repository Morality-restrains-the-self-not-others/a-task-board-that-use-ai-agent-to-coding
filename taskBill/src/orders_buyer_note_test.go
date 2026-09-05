package main

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestResourceRequiresManualFulfillment(t *testing.T) {
	if !resourceRequiresManualFulfillment(ResourceTypeGitlabDisk) {
		t.Fatal("gitlab_disk should require manual fulfillment")
	}
	if resourceRequiresManualFulfillment(ResourceTypeTaskPost) {
		t.Fatal("task_post is auto-delivered")
	}
	if resourceRequiresManualFulfillment(ResourceTypeGitlabTraffic) {
		t.Fatal("gitlab_traffic is auto-delivered")
	}
}

func TestNormalizeBuyerNote(t *testing.T) {
	disk := []orderItemInput{{ResourceType: ResourceTypeGitlabDisk, Quantity: 1}}
	taskOnly := []orderItemInput{{ResourceType: ResourceTypeTaskPost, Quantity: 1}}

	got, err := normalizeBuyerNote("  请开通 team-foo  ", disk)
	if err != nil || got != "请开通 team-foo" {
		t.Fatalf("trim disk note: got %q err=%v", got, err)
	}
	got, err = normalizeBuyerNote("   ", disk)
	if err != nil || got != "" {
		t.Fatalf("blank note: got %q err=%v", got, err)
	}
	if _, err := normalizeBuyerNote("need construction", taskOnly); err == nil {
		t.Fatal("expected error when auto-delivery cart has note")
	}
	long := strings.Repeat("字", maxBuyerNoteRunes+1)
	if utf8.RuneCountInString(long) != maxBuyerNoteRunes+1 {
		t.Fatal("fixture")
	}
	if _, err := normalizeBuyerNote(long, disk); err == nil {
		t.Fatal("expected too-long error")
	}
}

func TestCreateOrderWithNote_GitlabDiskPersists(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000001
	seedVIP1Membership(t, tenantID)
	note := "请把组路径开到 team-foo"
	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	}, note, "user-9301")
	if err != nil {
		t.Fatalf("createOrderWithNote: %v", err)
	}
	if order.BuyerNote != note {
		t.Fatalf("order.BuyerNote=%q", order.BuyerNote)
	}
	loaded, _, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	if loaded.BuyerNote != note {
		t.Fatalf("loaded BuyerNote=%q", loaded.BuyerNote)
	}
	m := orderJSON(loaded, nil)
	if m["buyer_note"] != note {
		t.Fatalf("json buyer_note=%v", m["buyer_note"])
	}
}

func TestCreateOrderWithNote_TaskPostRejectsNote(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000002
	_, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "should not store", "")
	if err == nil {
		t.Fatal("expected error")
	}
	var n int
	if qerr := db.QueryRow(`SELECT COUNT(*) FROM billing_resource_order WHERE tenant_id = ?`, tenantID).Scan(&n); qerr != nil {
		t.Fatalf("count: %v", qerr)
	}
	if n != 0 {
		t.Fatalf("orders inserted=%d", n)
	}
}

// OPT-20260819-027: 非空下单留言须在同事务写入首条 billing_order_comment（author_side=tenant）。
func TestCreateOrderWithNote_WritesFirstOrderComment(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000004
	seedVIP1Membership(t, tenantID)
	note := "请把组路径开到 team-bar"
	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	}, note, "user-9304")
	if err != nil {
		t.Fatalf("createOrderWithNote: %v", err)
	}

	var commentID int64
	var authorSide, content, authorUserID string
	if err := db.QueryRow(`
		SELECT id, author_side, content, author_user_id
		FROM billing_order_comment WHERE order_id = ? AND tenant_id = ?`,
		order.ID, tenantID,
	).Scan(&commentID, &authorSide, &content, &authorUserID); err != nil {
		t.Fatalf("load comment: %v", err)
	}
	if authorSide != "tenant" {
		t.Fatalf("author_side=%q, want tenant", authorSide)
	}
	if content != note {
		t.Fatalf("content=%q, want %q", content, note)
	}
	if authorUserID != "user-9304" {
		t.Fatalf("author_user_id=%q", authorUserID)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_order_comment WHERE order_id = ?`, order.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("comment rows=%d, want 1", n)
	}
}

func TestCreateOrderWithNote_EmptyNoteWritesNoComment(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000005
	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "", "")
	if err != nil {
		t.Fatalf("createOrderWithNote: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_order_comment WHERE order_id = ?`, order.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("comment rows=%d, want 0", n)
	}
}

func TestCreateOrderWithNote_EmptyNoteOnTaskPostOK(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000003
	order, _, err := createOrderWithNote(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "  ", "")
	if err != nil {
		t.Fatalf("empty note should succeed: %v", err)
	}
	if order.BuyerNote != "" {
		t.Fatalf("BuyerNote=%q", order.BuyerNote)
	}
}
