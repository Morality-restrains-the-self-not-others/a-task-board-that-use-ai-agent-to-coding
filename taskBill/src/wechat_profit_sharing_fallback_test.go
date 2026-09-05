package main

import (
	"context"
	"testing"
	"time"
)

// context is used by stubPaytimeQualification callback.

func rfc3339DaysAgo(days int) string {
	return time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
}

func rfc3339DaysFromNow(days int) string {
	return time.Now().UTC().AddDate(0, 0, days).Format(time.RFC3339)
}

func enableWechatLive(t *testing.T) {
	t.Helper()
	orig := wechatLiveOK
	wechatLiveOK = true
	t.Cleanup(func() { wechatLiveOK = orig })
}

type createPSProbe struct {
	calls int
	outNo []string
}

func stubCreateProfitSharingOK(t *testing.T) *createPSProbe {
	t.Helper()
	p := &createPSProbe{}
	orig := createProfitSharingOrder
	createProfitSharingOrder = func(_ context.Context, outOrderNo, _ string, _ string, _ []profitSharingReceiver) (string, error) {
		p.calls++
		p.outNo = append(p.outNo, outOrderNo)
		return "wx-ps-fallback-1", nil
	}
	t.Cleanup(func() { createProfitSharingOrder = orig })
	return p
}

func setOrderPaidAt(t *testing.T, orderID int64, paidAt string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE billing_resource_order SET paid_at = ? WHERE id = ?`, paidAt, orderID); err != nil {
		t.Fatalf("set paid_at: %v", err)
	}
}

func setProfitSharingRow(t *testing.T, id int64, status, settleAfter, wechatID string) {
	t.Helper()
	if _, err := db.Exec(`
		UPDATE billing_profit_sharing
		SET status = ?, settle_after = ?, wechat_profit_sharing_id = ?, updated_at = ?
		WHERE id = ?`, status, settleAfter, wechatID, utcNow(), id); err != nil {
		t.Fatalf("update profit sharing: %v", err)
	}
}

func profitSharingStatus(t *testing.T, id int64) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM billing_profit_sharing WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("load status: %v", err)
	}
	return status
}

func profitSharingOutNo(t *testing.T, id int64) string {
	t.Helper()
	var outNo string
	if err := db.QueryRow(`SELECT out_profit_sharing_no FROM billing_profit_sharing WHERE id = ?`, id).Scan(&outNo); err != nil {
		t.Fatalf("load out_no: %v", err)
	}
	return outNo
}

func TestProcessPendingProfitSharingsFallbackPendingFutureSettleAfter26Days(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96201)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB1", "420000fb0001")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(26))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusPending, rfc3339DaysFromNow(10), "")

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 1 {
		t.Fatalf("create calls=%d want 1", probe.calls)
	}
	if profitSharingStatus(t, psID) != psStatusFinished {
		t.Fatalf("status=%s want finished", profitSharingStatus(t, psID))
	}
}

func TestProcessPendingProfitSharingsSkipsPending24DaysFutureSettleAfter(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96202)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB2", "420000fb0002")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(24))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusPending, rfc3339DaysFromNow(10), "")

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 0 {
		t.Fatalf("create calls=%d want 0", probe.calls)
	}
	if profitSharingStatus(t, psID) != psStatusPending {
		t.Fatalf("status=%s want pending", profitSharingStatus(t, psID))
	}
}

func TestProcessPendingProfitSharingsRetriesFailed26DaysSameOutNo(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96203)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB3", "420000fb0003")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(26))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusFailed, rfc3339DaysFromNow(10), "")
	wantOut := profitSharingOutNo(t, psID)

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 1 {
		t.Fatalf("create calls=%d want 1", probe.calls)
	}
	if probe.outNo[0] != wantOut {
		t.Fatalf("out_no=%s want %s", probe.outNo[0], wantOut)
	}
	if profitSharingStatus(t, psID) != psStatusFinished {
		t.Fatalf("status=%s want finished", profitSharingStatus(t, psID))
	}
}

func TestProcessPendingProfitSharingsSkipsFinished26Days(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96204)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB4", "420000fb0004")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(26))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusFinished, rfc3339DaysAgo(1), "wx-already")

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 0 {
		t.Fatalf("create calls=%d want 0", probe.calls)
	}
}

func TestProcessPendingProfitSharingsReplayDoesNotCreateTwice(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96205)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB5", "420000fb0005")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(26))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusPending, rfc3339DaysFromNow(10), "")

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process 1: %v", err)
	}
	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process 2: %v", err)
	}
	if probe.calls != 1 {
		t.Fatalf("create calls=%d want 1", probe.calls)
	}
}

func TestProcessPendingProfitSharingsSkipsOlderThanWechatWindow(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96206)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB6", "420000fb0006")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(31))
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	setProfitSharingRow(t, psID, psStatusFailed, rfc3339DaysFromNow(1), "")

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 0 {
		t.Fatalf("create calls=%d want 0", probe.calls)
	}
}

func TestProcessPendingProfitSharingsMarksUnmarkedEligibleOrderAndExecutes(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)
	stubPaytimeQualification(t, func(ctx context.Context, referrerUserID string) bool { return true })

	tenantID := int64(96207)
	orderID := generateSnowflakeID()
	buyerID := generateSnowflakeID()
	referrerID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB7", "420000fb0007")
	if _, err := db.Exec(`UPDATE billing_resource_order SET user_id = ?, paid_at = ? WHERE id = ?`,
		buyerID, rfc3339DaysAgo(26), orderID); err != nil {
		t.Fatalf("set user/paid_at: %v", err)
	}
	if err := upsertReferralEdge(formatID(referrerID), formatID(buyerID), tenantID, utcNow(), "CH", true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE billing_referral_edge SET referrer_openid = ? WHERE referred_user_id = ?`,
		"openid-fallback-7", formatID(buyerID)); err != nil {
		t.Fatalf("set openid: %v", err)
	}

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 1 {
		t.Fatalf("create calls=%d want 1", probe.calls)
	}
	var n int
	var status string
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(status),'') FROM billing_profit_sharing WHERE order_id = ?`,
		orderID).Scan(&n, &status); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows=%d want 1", n)
	}
	if status != psStatusFinished {
		t.Fatalf("status=%s want finished", status)
	}
}

func TestProcessPendingProfitSharingsSkipsDueBeforeFallbackWindow(t *testing.T) {
	// 2026-08-23-referrer-manual-profit-sharing-design.md：15 天解冻后改由推荐人手动分账，
	// 扫描不再自动执行 due pending；仅 25 天窗口兜底（TestProcessPendingProfitSharingsFallback25Days 覆盖）。
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	enableWechatLive(t)
	probe := stubCreateProfitSharingOK(t)

	tenantID := int64(96208)
	orderID := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX-FB8", "420000fb0008")
	setOrderPaidAt(t, orderID, rfc3339DaysAgo(10))
	psID := seedProfitSharingRecord(t, orderID, tenantID)

	if err := processPendingProfitSharings(context.Background()); err != nil {
		t.Fatalf("process: %v", err)
	}
	if probe.calls != 0 {
		t.Fatalf("create calls=%d want 0 (10d due, manual share only)", probe.calls)
	}
	if profitSharingStatus(t, psID) != psStatusPending {
		t.Fatalf("status=%s want pending (unchanged)", profitSharingStatus(t, psID))
	}
}

func TestMergeProfitSharingRecordsDedupsByID(t *testing.T) {
	a := []profitSharingRecord{{ID: 1, OutProfitSharingNo: "PS1"}, {ID: 2, OutProfitSharingNo: "PS2"}}
	b := []profitSharingRecord{{ID: 2, OutProfitSharingNo: "PS2-dup"}, {ID: 3, OutProfitSharingNo: "PS3"}}
	got := mergeProfitSharingRecords(a, b)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}
	if got[1].OutProfitSharingNo != "PS2" {
		t.Fatalf("id 2 kept first out_no=%s", got[1].OutProfitSharingNo)
	}
}
