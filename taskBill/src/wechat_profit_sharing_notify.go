package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"tracelog"
)

// profitSharingReturnOrderCall 是分账回退的 SDK 可注入接缝。
var profitSharingReturnOrderCall = func(ctx context.Context, svc *profitsharing.ReturnOrdersApiService, req profitsharing.CreateReturnOrderRequest) (*profitsharing.ReturnOrdersEntity, *core.APIResult, error) {
	return svc.CreateReturnOrder(ctx, req)
}

// profitSharingChangeContent 微信分账动账通知解密后的 resource（profitsharing）。
// 官方文档：https://pay.weixin.qq.com （分账动账通知，2026-01-28）
type profitSharingChangeContent struct {
	Mchid         string `json:"mchid"`
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
	OutOrderNo    string `json:"out_order_no"`
	State         string `json:"state"`
	Receiver      struct {
		Type        string `json:"type"`
		Account     string `json:"account"`
		Amount      int64  `json:"amount"`
		Description string `json:"description"`
		SuccessTime string `json:"success_time"`
	} `json:"receiver"`
	SuccessTime string `json:"success_time"`
}

// handleProfitSharingNotify 处理微信分账动账/结果通知
// POST /api/billing/profitsharing/change-notify/ 与存量 /api/billing/profitsharing/notify/
func handleProfitSharingNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"code": "FAIL", "message": "method"})
		return
	}
	notifyID, eventType, content, err := parseProfitSharingChangeNotify(r)
	if err != nil {
		slog.WarnContext(r.Context(), "profit_sharing_change_notify_parse_failed",
			"level", "warn",
			"error", err.Error(),
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "verify"})
		return
	}
	if err := applyProfitSharingChangeNotify(r.Context(), notifyID, eventType, content); err != nil {
		slog.ErrorContext(r.Context(), "profit_sharing_change_notify_apply_failed",
			"level", "error",
			"error", err.Error(),
			"notify_id", notifyID,
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "FAIL", "message": "apply"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS"})
}

func parseProfitSharingChangeNotify(r *http.Request) (notifyID, eventType string, content profitSharingChangeContent, err error) {
	if wechatIsMock() {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			return "", "", content, readErr
		}
		var flat struct {
			ID            string `json:"id"`
			EventType     string `json:"event_type"`
			OutOrderNo    string `json:"out_order_no"`
			TransactionID string `json:"transaction_id"`
			OrderID       string `json:"order_id"`
			State         string `json:"state"`
			SuccessTime   string `json:"success_time"`
		}
		if uErr := json.Unmarshal(body, &flat); uErr != nil {
			return "", "", content, uErr
		}
		content.OutOrderNo = flat.OutOrderNo
		content.TransactionID = flat.TransactionID
		content.OrderID = flat.OrderID
		content.State = flat.State
		content.SuccessTime = flat.SuccessTime
		return strings.TrimSpace(flat.ID), strings.TrimSpace(flat.EventType), content, nil
	}
	if wechatNotifyH == nil {
		return "", "", content, fmt.Errorf("wechat notify handler not ready")
	}
	var parsed profitSharingChangeContent
	nreq, pErr := wechatNotifyH.ParseNotifyRequest(r.Context(), r, &parsed)
	if pErr != nil {
		return "", "", content, pErr
	}
	id := ""
	et := ""
	if nreq != nil {
		id = strings.TrimSpace(nreq.ID)
		et = strings.TrimSpace(nreq.EventType)
	}
	return id, et, parsed, nil
}

func applyProfitSharingChangeNotify(ctx context.Context, notifyID, eventType string, content profitSharingChangeContent) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	if strings.TrimSpace(notifyID) == "" {
		return fmt.Errorf("missing notify id")
	}
	now := time.Now().UTC()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT IGNORE INTO billing_profit_sharing_change_notify
		  (notify_id, out_order_no, wechat_transaction_id, wechat_order_id, event_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		notifyID, content.OutOrderNo, content.TransactionID, content.OrderID, eventType, now)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		slog.InfoContext(ctx, "profit_sharing_change_notify_duplicate",
			"level", "info",
			"notify_id", notifyID,
			"out_order_no", content.OutOrderNo,
		)
		return tx.Commit()
	}

	outNo := strings.TrimSpace(content.OutOrderNo)
	if outNo == "" {
		slog.WarnContext(ctx, "profit_sharing_change_notify_missing_out_order_no",
			"level", "warn",
			"notify_id", notifyID,
		)
		return tx.Commit()
	}

	var psID, orderID int64
	err = tx.QueryRow(`
		SELECT id, order_id FROM billing_profit_sharing
		WHERE out_profit_sharing_no = ? FOR UPDATE`, outNo).Scan(&psID, &orderID)
	if err == sql.ErrNoRows {
		slog.WarnContext(ctx, "profit_sharing_change_notify_unknown_out_order_no",
			"level", "warn",
			"notify_id", notifyID,
			"out_order_no", outNo,
		)
		return tx.Commit()
	}
	if err != nil {
		return err
	}

	wechatPSID := strings.TrimSpace(content.OrderID)
	txnID := strings.TrimSpace(content.TransactionID)
	state := strings.ToUpper(strings.TrimSpace(content.State))
	isReturn := strings.Contains(content.Receiver.Description, "回退") || strings.Contains(eventType, "REFUND")
	nowRFC := now.UTC().Format(time.RFC3339)

	failTID := tracelog.TraceIDFromContext(ctx)
	switch {
	case isReturn:
		_, err = tx.Exec(`
			UPDATE billing_profit_sharing
			SET wechat_profit_sharing_id = CASE WHEN ? = '' THEN wechat_profit_sharing_id ELSE ? END,
			    status = ?, fail_reason = '微信分账回退', fail_trace_id = ?, updated_at = ?
			WHERE id = ?`, wechatPSID, wechatPSID, psStatusReturned, failTID, nowRFC, psID)
	case state == "CLOSED":
		_, err = tx.Exec(`
			UPDATE billing_profit_sharing
			SET status = ?, fail_reason = '微信侧关闭', fail_trace_id = ?, updated_at = ?
			WHERE id = ?`, psStatusFailed, failTID, nowRFC, psID)
	default:
		_, err = tx.Exec(`
			UPDATE billing_profit_sharing
			SET status = ?, wechat_profit_sharing_id = CASE WHEN ? = '' THEN wechat_profit_sharing_id ELSE ? END,
			    settled_at = ?, updated_at = ?
			WHERE id = ?`,
			psStatusFinished, wechatPSID, wechatPSID, nowRFC, nowRFC, psID)
	}
	if err != nil {
		return err
	}
	if txnID != "" && orderID > 0 {
		if _, err := tx.Exec(`
			UPDATE billing_resource_order
			SET wechat_transaction_id = ?
			WHERE id = ? AND (wechat_transaction_id IS NULL OR wechat_transaction_id = '')`,
			txnID, orderID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	slog.InfoContext(ctx, "profit_sharing_change_notify_applied",
		"level", "info",
		"notify_id", notifyID,
		"out_order_no", outNo,
		"profit_sharing_id", psID,
		"event_type", eventType,
	)
	return nil
}

// decryptWechatNotify 发票回调仍走明文/已解密 body；分账动账通知请用 ParseNotifyRequest。
func decryptWechatNotify(body []byte) ([]byte, error) {
	return body, nil
}

// verifyWechatNotification 保留给发票等其它路径的轻量探测；分账动账走 ParseNotifyRequest。
func verifyWechatNotification(headers http.Header, body []byte) bool {
	if wechatIsMock() {
		return true
	}
	if wechatNotifyH == nil {
		return false
	}
	req := &http.Request{Header: headers}
	_, err := wechatNotifyH.ParseNotifyRequest(context.Background(), req, &notify.ContentMap{})
	return err == nil
}

// —— 退款场景：分账取消/回退 ——

// cancelProfitSharingForOrder 订单退款时取消/回退关联的分账记录
// 由退款审批通过后调用
func cancelProfitSharingForOrder(ctx context.Context, orderID int64) error {
	// 查找该订单的分账记录
	row := db.QueryRow(`
		SELECT id, out_profit_sharing_no, status, wechat_profit_sharing_id, commission_yuan_cents
		FROM billing_profit_sharing WHERE order_id = ?`, orderID)

	var rec struct {
		ID                  int64
		OutNo               string
		Status              string
		WechatOrderID       string
		CommissionYuanCents int64
	}
	err := row.Scan(&rec.ID, &rec.OutNo, &rec.Status, &rec.WechatOrderID, &rec.CommissionYuanCents)
	if err != nil {
		// 无分账记录，无需处理
		return nil
	}

	switch rec.Status {
	case psStatusPending:
		// 分账尚未发起，直接取消本地记录
		return voidProfitSharingRecord(ctx, rec.ID, "订单退款取消分账")

	case psStatusProcessing:
		// 分账处理中，标记待回退
		return voidProfitSharingRecord(ctx, rec.ID, "订单退款-分账处理中需人工确认")

	case psStatusFinished:
		// 分账已完成，需要调用微信分账回退 API
		return rollbackWechatProfitSharing(ctx, rec.OutNo, rec.WechatOrderID, rec.CommissionYuanCents, rec.ID)

	case psStatusFailed:
		// 分账失败，直接取消
		return voidProfitSharingRecord(ctx, rec.ID, "订单退款-分账已失败取消")

	default:
		return nil
	}
}

// voidProfitSharingRecord 作废本地分账记录
func voidProfitSharingRecord(ctx context.Context, id int64, reason string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE billing_profit_sharing
		SET status = 'voided', fail_reason = ?, fail_trace_id = ?, updated_at = ?
		WHERE id = ?`, reason, tracelog.TraceIDFromContext(ctx), now, id)
	if err != nil {
		return fmt.Errorf("void profit sharing %d: %w", id, err)
	}
	log.Printf("[taskBill] profit sharing voided: id=%d reason=%s", id, reason)
	return nil
}

// rollbackWechatProfitSharing 调用微信分账回退 API 追回已分账佣金
// POST /v3/profitsharing/return-orders（sdk ReturnOrdersApiService.CreateReturnOrder）
func rollbackWechatProfitSharing(ctx context.Context, outProfitSharingNo, wechatOrderID string, amountYuanCents int64, localID int64) error {
	if !wechatConfigured() {
		return fmt.Errorf("wechat pay not configured, cannot rollback profit sharing")
	}
	if wechatClient == nil {
		return fmt.Errorf("wechat pay client not initialized")
	}

	outReturnNo := fmt.Sprintf("PSR%d", generateSnowflakeID())
	req := profitsharing.CreateReturnOrderRequest{
		OrderId:     core.String(wechatOrderID),
		OutOrderNo:  core.String(outProfitSharingNo),
		OutReturnNo: core.String(outReturnNo),
		ReturnMchid: core.String(wechatCfg.Mchid),
		Amount:      core.Int64(amountYuanCents),
		Description: core.String("订单退款-分账回退"),
	}
	svc := profitsharing.ReturnOrdersApiService{Client: wechatClient}
	_, _, err := profitSharingReturnOrderCall(ctx, &svc, req)
	if err != nil {
		// 回退失败，标记需人工处理
		detail := wechatSDKResultError("wechat profit sharing return", err)
		voidProfitSharingRecord(ctx, localID, fmt.Sprintf("分账回退失败需人工处理: %v", detail))
		return fmt.Errorf("wechat profit sharing return failed: %w", err)
	}

	// 回退成功，标记分账记录为已回退
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`
		UPDATE billing_profit_sharing
		SET status = 'returned', fail_reason = '订单退款-分账已回退', fail_trace_id = ?, updated_at = ?
		WHERE id = ?`, tracelog.TraceIDFromContext(ctx), now, localID)

	log.Printf("[taskBill] profit sharing returned: local_id=%d out_no=%s amount=%d分",
		localID, outProfitSharingNo, amountYuanCents)
	return nil
}

// cancelPendingProfitSharingsForTenant 退款时取消租户待分账记录（FIFO 顺序）
// 按创建时间从早到晚取消，直到累计金额达到退款金额
func cancelPendingProfitSharingsForTenant(ctx context.Context, tenantID, refundPoints int64) {
	rows, err := db.Query(`
		SELECT id, order_id, commission_yuan_cents, status
		FROM billing_profit_sharing
		WHERE tenant_id = ? AND status IN (?, ?)
		ORDER BY created_at ASC`, tenantID, psStatusPending, psStatusProcessing)
	if err != nil {
		log.Printf("[taskBill] query profit sharing for refund rollback: %v", err)
		return
	}
	defer rows.Close()

	var accumulated int64
	for rows.Next() {
		var id, orderID, status string
		var comm int64
		if err := rows.Scan(&id, &orderID, &comm, &status); err != nil {
			continue
		}
		if status == psStatusPending || status == psStatusProcessing {
			if err := voidProfitSharingRecord(ctx, parseInt64Str(id), "退款取消分账"); err != nil {
				log.Printf("[taskBill] void profit sharing for refund: %v", err)
			}
			accumulated += comm
		}
		if accumulated >= refundPoints {
			break
		}
	}
	if accumulated > 0 {
		log.Printf("[taskBill] profit sharing rollback for refund: tenant=%d voided=%d分", tenantID, accumulated)
	}
}

func parseInt64Str(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
