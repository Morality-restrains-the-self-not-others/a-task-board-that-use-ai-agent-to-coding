package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func stubWechatPayIdentity(t *testing.T, ident wechatPayIdentity, lookupErr error) {
	t.Helper()
	orig := lookupWechatPayIdentity
	t.Cleanup(func() { lookupWechatPayIdentity = orig })
	lookupWechatPayIdentity = func(context.Context, string, string) (wechatPayIdentity, error) {
		return ident, lookupErr
	}
}

func TestExecuteProfitSharingRejectsEmptyOpenidWithoutCallingWechat(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubWechatPayIdentity(t, wechatPayIdentity{}, nil)

	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = '' WHERE id = ?`, psID); err != nil {
		t.Fatalf("clear openid: %v", err)
	}
	var rec profitSharingRecord
	if err := db.QueryRow(`
		SELECT id, out_profit_sharing_no, order_id, order_number, tenant_id,
		       referrer_user_id, referrer_openid, commission_yuan_cents, total_yuan_cents,
		       status, COALESCE(fail_reason, '')
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid, &rec.CommissionYuanCents, &rec.TotalYuanCents,
		&rec.Status, &rec.FailReason,
	); err != nil {
		t.Fatalf("load: %v", err)
	}

	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-should-not-run", nil
	}

	err := executeProfitSharing(context.Background(), rec)
	if !errors.Is(err, errReferrerOpenidMissing) {
		t.Fatalf("err=%v want errReferrerOpenidMissing", err)
	}
	if calls != 0 {
		t.Fatalf("CreateOrder calls=%d want 0", calls)
	}
}

func TestExecuteProfitSharingUsesRegisteredReceiverOpenid(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubWechatPayIdentity(t, wechatPayIdentity{}, errors.New("identity must not be consulted"))

	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = '', referrer_user_id = 'referrer-live' WHERE id = ?`, psID); err != nil {
		t.Fatalf("clear openid: %v", err)
	}
	if err := upsertProfitSharingReceiverRow("referrer-live", "wxapp", "oLIVE-openid", psReceiverRegistered, "", true); err != nil {
		t.Fatalf("receiver: %v", err)
	}
	var rec profitSharingRecord
	if err := db.QueryRow(`
		SELECT id, out_profit_sharing_no, order_id, order_number, tenant_id,
		       referrer_user_id, referrer_openid, commission_yuan_cents, total_yuan_cents
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid, &rec.CommissionYuanCents, &rec.TotalYuanCents,
	); err != nil {
		t.Fatalf("load: %v", err)
	}

	var gotAccount string
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(_ context.Context, _ string, _ string, _ string, receivers []profitSharingReceiver) (string, error) {
		if len(receivers) > 0 {
			gotAccount = receivers[0].Account
		}
		return "wx-ps-live", nil
	}

	if err := executeProfitSharing(context.Background(), rec); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotAccount != "oLIVE-openid" {
		t.Fatalf("account=%q want oLIVE-openid", gotAccount)
	}
	var stored string
	if err := db.QueryRow(`SELECT referrer_openid FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&stored); err != nil {
		t.Fatalf("stored: %v", err)
	}
	if stored != "oLIVE-openid" {
		t.Fatalf("persisted openid=%q", stored)
	}
}

func TestExecuteProfitSharingPrefersReceiverOverStaleSnapshot(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubWechatPayIdentity(t, wechatPayIdentity{}, errors.New("identity must not be consulted"))

	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = ?, referrer_user_id = ? WHERE id = ?`,
		"osshz2WnJeZXaLGYtxwVmZwyf0Xg", "referrer-live", psID); err != nil {
		t.Fatalf("stale snapshot: %v", err)
	}
	if err := upsertProfitSharingReceiverRow("referrer-live", "wx31273ca77c89dffe", "oNsrS1BPzk9Lr7IvyVl9lHIQKwB8", psReceiverRegistered, "", true); err != nil {
		t.Fatalf("receiver: %v", err)
	}
	var rec profitSharingRecord
	if err := db.QueryRow(`
		SELECT id, out_profit_sharing_no, order_id, order_number, tenant_id,
		       referrer_user_id, referrer_openid, commission_yuan_cents, total_yuan_cents
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid, &rec.CommissionYuanCents, &rec.TotalYuanCents,
	); err != nil {
		t.Fatalf("load: %v", err)
	}

	var gotAccount string
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(_ context.Context, _ string, _ string, _ string, receivers []profitSharingReceiver) (string, error) {
		if len(receivers) > 0 {
			gotAccount = receivers[0].Account
		}
		return "wx-ps-pair", nil
	}

	if err := executeProfitSharing(context.Background(), rec); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotAccount != "oNsrS1BPzk9Lr7IvyVl9lHIQKwB8" {
		t.Fatalf("account=%q want mp openid, not web snapshot", gotAccount)
	}
	var stored string
	if err := db.QueryRow(`SELECT referrer_openid FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&stored); err != nil {
		t.Fatalf("stored: %v", err)
	}
	if stored != "oNsrS1BPzk9Lr7IvyVl9lHIQKwB8" {
		t.Fatalf("persisted openid=%q want corrected mp pair", stored)
	}
}

func TestExecuteProfitSharingUsesIdentityLookupWhenSnapshotEmpty(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	stubWechatPayIdentity(t, wechatPayIdentity{AppID: "wxapp", OpenID: "oAUTH-openid"}, nil)

	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET referrer_openid = '' WHERE id = ?`, psID); err != nil {
		t.Fatalf("clear openid: %v", err)
	}
	var rec profitSharingRecord
	if err := db.QueryRow(`
		SELECT id, out_profit_sharing_no, order_id, order_number, tenant_id,
		       referrer_user_id, referrer_openid, commission_yuan_cents, total_yuan_cents
		FROM billing_profit_sharing WHERE id = ?`, psID).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid, &rec.CommissionYuanCents, &rec.TotalYuanCents,
	); err != nil {
		t.Fatalf("load: %v", err)
	}

	var gotAccount string
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(_ context.Context, _ string, _ string, _ string, receivers []profitSharingReceiver) (string, error) {
		if len(receivers) > 0 {
			gotAccount = receivers[0].Account
		}
		return "wx-ps-auth", nil
	}

	if err := executeProfitSharing(context.Background(), rec); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotAccount != "oAUTH-openid" {
		t.Fatalf("account=%q want oAUTH-openid", gotAccount)
	}
}

func TestProfitSharingActionClientErrorEmptyAccount(t *testing.T) {
	status, msg := profitSharingActionClientError(errReferrerOpenidMissing)
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", status)
	}
	if !strings.Contains(msg, "未绑定微信收款账号") {
		t.Fatalf("msg=%s", msg)
	}
	raw := errors.New(`create profit sharing failed: HTTP 400, {"code":"PARAM_ERROR","detail":{"location":"body","value":0},"message":"输入源“/body/receivers/0/account”映射到值字段“分账接收方帐号”字符串规则校验失败，字符数 0，小于最小值 1"}`)
	status, msg = profitSharingActionClientError(raw)
	if status != http.StatusBadRequest {
		t.Fatalf("param-error status=%d want 400", status)
	}
	if strings.Contains(msg, "PARAM_ERROR") || strings.Contains(msg, "HTTP 400") {
		t.Fatalf("leaked provider dump: %s", msg)
	}
	if !strings.Contains(msg, "未绑定微信收款账号") {
		t.Fatalf("msg=%s", msg)
	}
}
