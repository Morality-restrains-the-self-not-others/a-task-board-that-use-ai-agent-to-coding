package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	native "github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

func persistWechatPayVouchers(orderID int64, outTradeNo, transactionID string) {
	if orderID <= 0 {
		return
	}
	outTradeNo = strings.TrimSpace(outTradeNo)
	transactionID = strings.TrimSpace(transactionID)
	var paymentRef, existingOut string
	_ = db.QueryRow(
		`SELECT COALESCE(payment_ref,''), COALESCE(out_trade_no,'') FROM billing_resource_order WHERE id = ?`,
		orderID,
	).Scan(&paymentRef, &existingOut)
	if outTradeNo == "" {
		outTradeNo = strings.TrimSpace(existingOut)
	}
	if outTradeNo == "" {
		outTradeNo = strings.TrimPrefix(paymentRef, "wechat:")
	}
	if outTradeNo != "" {
		if _, err := db.Exec(`UPDATE billing_resource_order SET out_trade_no = ? WHERE id = ?`, outTradeNo, orderID); err != nil {
			slog.Error("wechat_pay_vouchers_out_trade_no_failed",
				"level", "error", "order_id", formatID(orderID), "err", err.Error())
		}
	}
	if transactionID != "" {
		if _, err := db.Exec(`UPDATE billing_resource_order SET wechat_transaction_id = ? WHERE id = ?`, transactionID, orderID); err != nil {
			slog.Error("wechat_pay_vouchers_txn_id_failed",
				"level", "error", "order_id", formatID(orderID), "err", err.Error())
		}
		if _, err := db.Exec(`
			UPDATE billing_payment_ledger SET provider_capture_id = ?
			WHERE channel = 'wechat' AND provider_ref IN (?, ?, ?)`,
			transactionID, outTradeNo, "wechat:"+outTradeNo, paymentRef); err != nil {
			slog.Error("wechat_pay_vouchers_ledger_failed",
				"level", "error", "order_id", formatID(orderID), "err", err.Error())
		}
	}
	slog.Info("wechat_pay_vouchers_persisted",
		"level", "info",
		"order_id", formatID(orderID),
		"has_out_trade_no", outTradeNo != "",
		"has_txn_id", transactionID != "",
	)
}

func persistWechatPayVouchersFromNotify(orderIDStr, outTradeNo, transactionID string) {
	orderID, err := strconv.ParseInt(strings.TrimSpace(orderIDStr), 10, 64)
	if err != nil || orderID <= 0 {
		return
	}
	persistWechatPayVouchers(orderID, outTradeNo, transactionID)
}

func looksLikeWechatTxnID(q string) bool {
	if len(q) < 18 || len(q) > 32 {
		return false
	}
	for _, r := range q {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// wechatQueryOrderByIdCall 是 Native QueryOrderById 的可注入接缝（OPT-20260823-035）。
var wechatQueryOrderByIdCall = func(ctx context.Context, svc *native.NativeApiService, req native.QueryOrderByIdRequest) (*payments.Transaction, *core.APIResult, error) {
	return svc.QueryOrderById(ctx, req)
}

func queryWechatOutTradeNoByTxnID(txnID string) (string, error) {
	if wechatIsMock() || wechatClient == nil || strings.TrimSpace(wechatCfg.Mchid) == "" {
		return "", nil
	}
	// OPT-20260823-035：限流/熔断，禁止连点或脚本刷微信查单配额。
	if err := wechatQueryGuardInst.acquire(time.Now()); err != nil {
		return "", err
	}
	svc := native.NativeApiService{Client: wechatClient}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tid := strings.TrimSpace(txnID)
	mch := wechatCfg.Mchid
	resp, _, err := wechatQueryOrderByIdCall(ctx, &svc, native.QueryOrderByIdRequest{
		TransactionId: &tid,
		Mchid:         &mch,
	})
	if err != nil {
		wechatQueryGuardInst.recordFailure(time.Now())
		return "", err
	}
	wechatQueryGuardInst.recordSuccess(time.Now())
	if resp == nil || resp.OutTradeNo == nil {
		return "", nil
	}
	return strings.TrimSpace(*resp.OutTradeNo), nil
}

var resolveWechatTxnToOutTradeNo = queryWechatOutTradeNoByTxnID

// recordWechatTransactionIDForOrder 回调回写真实微信支付单号到订单与支付台账。
func recordWechatTransactionIDForOrder(orderIDStr, txnID string) {
	persistWechatPayVouchersFromNotify(orderIDStr, "", txnID)
}

// lookupWechatTransactionIDForOrder 分账所需真实微信支付单号：优先
// billing_payment_ledger.provider_capture_id（回调回写），无则报错，禁止用 out_trade_no 冒充。
func lookupWechatTransactionIDForOrder(orderID int64) (string, error) {
	return lookupWechatTransactionIDOnDB(db, orderID)
}

func lookupWechatTransactionIDOnDB(d *sql.DB, orderID int64) (string, error) {
	if d == nil {
		return "", fmt.Errorf("wechat transaction_id for order %d not found (db nil)", orderID)
	}
	var captureID string
	err := d.QueryRow(`
		SELECT COALESCE(provider_capture_id, '') FROM billing_payment_ledger
		WHERE channel = 'wechat'
		  AND provider_ref = (
			SELECT REPLACE(COALESCE(payment_ref, ''), 'wechat:', '') FROM billing_resource_order WHERE id = ?
		  )
		ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&captureID)
	if err == nil && strings.TrimSpace(captureID) != "" {
		return strings.TrimSpace(captureID), nil
	}
	return "", fmt.Errorf("wechat transaction_id for order %d not found (支付台账未回写真实单号)", orderID)
}
