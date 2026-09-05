package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
)

func TestWechatShareAmountFenCapsRoundHalfUpToFloor(t *testing.T) {
	// 生产单 ORD-...-880009524994408448：55 分 × 5% ROUND_HALF_UP=3，微信按净额×比例向下取整最多 2 分。
	got := wechatShareAmountFen(55, 0, 3, 5)
	if got != 2 {
		t.Fatalf("55分 5%% stored=3 → %d want 2", got)
	}
}

func TestWechatShareAmountFenDeductsRefundsProportionally(t *testing.T) {
	// 1000 分已退 500，台账佣金仍是原单 50 分；微信：floor(500×5%)=25。
	got := wechatShareAmountFen(1000, 500, 50, 5)
	if got != 25 {
		t.Fatalf("refunded half → %d want 25", got)
	}
}

func TestWechatShareAmountFenZeroAfterFullRefund(t *testing.T) {
	got := wechatShareAmountFen(55, 55, 3, 5)
	if got != 0 {
		t.Fatalf("full refund → %d want 0", got)
	}
}

func TestWechatShareAmountFenKeepsStoredWhenAlreadyAtOrBelowCap(t *testing.T) {
	if got := wechatShareAmountFen(1000, 0, 50, 5); got != 50 {
		t.Fatalf("1000×5%% stored=50 → %d want 50", got)
	}
	if got := wechatShareAmountFen(1000, 0, 40, 5); got != 40 {
		t.Fatalf("stored below cap → %d want 40", got)
	}
	if got := wechatShareAmountFen(0, 0, 50, 5); got != 50 {
		t.Fatalf("unknown total must not zero stored commission → %d want 50", got)
	}
}

func TestMinShareAmountWithUnsplit(t *testing.T) {
	if got := minShareAmount(25, 10); got != 10 {
		t.Fatalf("unsplit 10 < 25 → %d", got)
	}
	if got := minShareAmount(25, 100); got != 25 {
		t.Fatalf("unsplit above cap → %d", got)
	}
	if got := minShareAmount(25, -1); got != 25 {
		t.Fatalf("skip unsplit → %d", got)
	}
}

func TestExecuteProfitSharingCapsTinyOrderToWechatFloor(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94110
	orderID := int64(2000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX55CAP", "42000055556666")
	if _, err := db.Exec(`UPDATE billing_resource_order SET total_yuan_cents = 55 WHERE id = ?`, orderID); err != nil {
		t.Fatalf("order amount: %v", err)
	}
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	if _, err := db.Exec(`UPDATE billing_profit_sharing SET total_yuan_cents = 55, commission_yuan_cents = 3 WHERE id = ?`, psID); err != nil {
		t.Fatalf("ps amount: %v", err)
	}

	var gotAmount int64
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(_ context.Context, _ string, _ string, _ string, receivers []profitSharingReceiver) (string, error) {
		if len(receivers) > 0 {
			gotAmount = receivers[0].Amount
		}
		return "wx-ps-capped", nil
	}

	record := profitSharingRecord{
		ID:                  psID,
		OutProfitSharingNo:  "PS" + formatID(psID),
		OrderID:             orderID,
		OrderNumber:         "ORD" + formatID(orderID),
		TenantID:            tenantID,
		ReferrerOpenid:      "openid-1",
		TotalYuanCents:      55,
		CommissionYuanCents: 3,
	}
	if err := executeProfitSharing(context.Background(), record); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotAmount != 2 {
		t.Fatalf("CreateOrder amount=%d want 2 (WeChat floor of 55×5%%)", gotAmount)
	}
	var stored int64
	if err := db.QueryRow(`SELECT commission_yuan_cents FROM billing_profit_sharing WHERE id = ?`, psID).Scan(&stored); err != nil {
		t.Fatalf("stored: %v", err)
	}
	if stored != 2 {
		t.Fatalf("persisted commission=%d want 2", stored)
	}
}

func TestExecuteProfitSharingSkipsWechatWhenShareAmountZero(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	const tenantID int64 = 94111
	orderID := int64(2000 + tenantID)
	seedProfitSharingOrderWithLedger(t, tenantID, orderID, "wechat:WX0SHARE", "42000000001111")
	psID := seedProfitSharingRecord(t, orderID, tenantID)
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_refund_application (
			tenant_id, account_id, applicant_user_id, frozen_points, status, reason,
			reviewer_user_id, review_note, payment_refund_refs, order_id, reviewed_at, created_at, updated_at)
		VALUES (?, ?, 'buyer', 1000, ?, 'full refund', '', '', '[]', ?, NULL, ?, ?)`,
		tenantID, tenantID, refundStatusApproved, orderID, now, now); err != nil {
		t.Fatalf("refund: %v", err)
	}

	calls := 0
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		calls++
		return "wx-should-not-run", nil
	}
	record := profitSharingRecord{
		ID:                  psID,
		OutProfitSharingNo:  "PS" + formatID(psID),
		OrderID:             orderID,
		ReferrerOpenid:      "openid-1",
		TotalYuanCents:      1000,
		CommissionYuanCents: 50,
	}
	err := executeProfitSharing(context.Background(), record)
	if !errors.Is(err, errProfitSharingAmountZero) {
		t.Fatalf("err=%v want errProfitSharingAmountZero", err)
	}
	if calls != 0 {
		t.Fatalf("CreateOrder calls=%d want 0", calls)
	}
}

func TestProfitSharingActionClientErrorRatioExceededIsConflict(t *testing.T) {
	raw := errors.New(`create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额"}`)
	status, msg := profitSharingActionClientError(raw)
	if status != http.StatusConflict {
		t.Fatalf("status=%d want 409", status)
	}
	if msg != "分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额" {
		t.Fatalf("msg=%q", msg)
	}

	status, msg = profitSharingActionClientError(errProfitSharingAmountZero)
	if status != http.StatusConflict {
		t.Fatalf("zero status=%d want 409", status)
	}
	if msg != profitSharingAmountZeroPublic {
		t.Fatalf("zero msg=%q", msg)
	}

	apiErr := &core.APIError{StatusCode: 403, Code: "RULE_LIMIT", Message: "分账金额超出最大分账比例"}
	status, msg = profitSharingActionClientError(wechatSDKResultError("create profit sharing", apiErr))
	if status != http.StatusConflict {
		t.Fatalf("RULE_LIMIT status=%d want 409", status)
	}
	if msg != "分账金额超出最大分账比例" {
		t.Fatalf("RULE_LIMIT msg=%q", msg)
	}
}

func TestHandleSystemAdminShareProfitSharingRatioExceededReturns409(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	_, psID := seedAdminProfitSharingRow(t, psStatusPending, "2026-08-20T00:00:00Z")
	orig := createProfitSharingOrder
	t.Cleanup(func() { createProfitSharingOrder = orig })
	createProfitSharingOrder = func(context.Context, string, string, string, []profitSharingReceiver) (string, error) {
		return "", errors.New(`create profit sharing failed: HTTP 400, {"code":"INVALID_REQUEST","message":"分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额"}`)
	}
	req := adminShareReq(t, psID, "客服复核后补发起分账", "ik-ratio-409", "admin-1", nil)
	rec := httptest.NewRecorder()
	handleSystemAdminShareProfitSharing(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeJSONMap(t, rec)
	msg, _ := body["error"].(string)
	if msg != "分账金额超出最大分账比例，最大可分账金额需等比例扣除退款与补差回退等逆向交易金额" {
		t.Fatalf("error=%q", msg)
	}
}
