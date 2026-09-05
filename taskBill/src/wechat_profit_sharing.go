package main

import (
	"bytes"
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
	"github.com/wechatpay-apiv3/wechatpay-go/core/consts"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"tracelog"
)

// —— 分账配置 ——

// profitSharingDelayDays 订单完成后延迟 N 天才允许分账（与推荐页冻结窗口对齐，默认 15 天）
const defaultProfitSharingDelayDays = 15

// —— 分账记录状态 ——
const (
	psStatusPending    = "pending"
	psStatusProcessing = "processing"
	psStatusFinished   = "finished"
	psStatusFailed     = "failed"
)

// —— 微信分账 API ——

// —— 微信分账 SDK 可注入接缝（单测可替换，避免打到微信） ——

var profitSharingCreateOrderCall = func(ctx context.Context, svc *profitsharing.OrdersApiService, req profitsharing.CreateOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
	return svc.CreateOrder(ctx, req)
}

var profitSharingQueryOrderCall = func(ctx context.Context, svc *profitsharing.OrdersApiService, req profitsharing.QueryOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
	return svc.QueryOrder(ctx, req)
}

var profitSharingUnfreezeOrderCall = func(ctx context.Context, svc *profitsharing.OrdersApiService, req profitsharing.UnfreezeOrderRequest) (*profitsharing.OrdersEntity, *core.APIResult, error) {
	return svc.UnfreezeOrder(ctx, req)
}

// wechatSDKResultError 把 SDK 调用错误格式化为带 HTTP 状态与应答体摘要的 error，
// 便于 Loki 检索；取不到 APIError 时原样包裹。
func wechatSDKResultError(prefix string, err error) error {
	if apiErr, ok := err.(*core.APIError); ok && apiErr != nil {
		body := strings.TrimSpace(apiErr.Body)
		if body == "" {
			body = fmt.Sprintf("code=%s message=%s", apiErr.Code, apiErr.Message)
		}
		return fmt.Errorf("%s failed: HTTP %d, %s: %w", prefix, apiErr.StatusCode, truncateBytes([]byte(body), 500), apiErr)
	}
	return fmt.Errorf("%s failed: %w", prefix, err)
}

// deleteProfitSharingReceiver 删除分账接收方
// POST /v3/profitsharing/receivers/delete（sdk ReceiversApiService.DeleteReceiver）
func deleteProfitSharingReceiver(ctx context.Context, appid, receiverOpenid string) error {
	if !wechatLiveOK {
		return fmt.Errorf("wechat pay not in live mode")
	}
	if wechatClient == nil {
		return fmt.Errorf("wechat pay client not initialized")
	}
	req := profitsharing.DeleteReceiverRequest{
		Appid:   core.String(appid),
		Type:    profitsharing.RECEIVERTYPE_PERSONAL_OPENID.Ptr(),
		Account: core.String(receiverOpenid),
	}
	svc := profitsharing.ReceiversApiService{Client: wechatClient}
	_, _, err := profitSharingDeleteReceiverCall(ctx, &svc, req)
	if err != nil {
		return wechatSDKResultError("delete profit sharing receiver", err)
	}
	return nil
}

// createProfitSharingOrder 创建分账订单
// POST /v3/profitsharing/orders
var createProfitSharingOrder = createProfitSharingOrderImpl

func createProfitSharingOrderImpl(ctx context.Context, outOrderNo, wechatTransactionID string, outProfitSharingNo string, receivers []profitSharingReceiver) (wechatOrderID string, err error) {
	if !wechatConfigured() {
		return "", fmt.Errorf("wechat pay not configured")
	}
	if wechatClient == nil {
		return "", fmt.Errorf("wechat pay client not initialized")
	}
	recvList := make([]profitsharing.CreateOrderReceiver, len(receivers))
	for i, r := range receivers {
		if strings.TrimSpace(r.Account) == "" {
			return "", errReferrerOpenidMissing
		}
		recvList[i] = profitsharing.CreateOrderReceiver{
			Type:        core.String("PERSONAL_OPENID"),
			Account:     core.String(r.Account),
			Amount:      core.Int64(r.Amount),
			Description: core.String(r.Description),
		}
		if name := strings.TrimSpace(r.Name); name != "" {
			recvList[i].Name = core.String(name)
		}
	}
	req := profitsharing.CreateOrderRequest{
		Appid:           core.String(wechatCfg.Appid),
		TransactionId:   core.String(wechatTransactionID),
		OutOrderNo:      core.String(outProfitSharingNo),
		Receivers:       recvList,
		UnfreezeUnsplit: core.Bool(true),
	}
	svc := profitsharing.OrdersApiService{Client: wechatClient}
	resp, _, err := profitSharingCreateOrderCall(ctx, &svc, req)
	if err != nil {
		return "", wechatSDKResultError("create profit sharing", err)
	}
	if resp == nil || resp.OrderId == nil || *resp.OrderId == "" {
		return "", fmt.Errorf("create profit sharing failed: empty order_id")
	}
	return *resp.OrderId, nil
}

// queryProfitSharingOrder 查询分账结果
// GET /v3/profitsharing/orders/{out_order_no}（sdk OrdersApiService.QueryOrder）
func queryProfitSharingOrder(ctx context.Context, outOrderNo, wechatTransactionID string) (status string, err error) {
	if wechatClient == nil {
		return "", fmt.Errorf("wechat pay client not initialized")
	}
	req := profitsharing.QueryOrderRequest{
		TransactionId: core.String(wechatTransactionID),
		OutOrderNo:    core.String(outOrderNo),
	}
	svc := profitsharing.OrdersApiService{Client: wechatClient}
	resp, _, err := profitSharingQueryOrderCall(ctx, &svc, req)
	if err != nil {
		return "", wechatSDKResultError("query profit sharing", err)
	}
	if resp == nil || resp.State == nil {
		return "", fmt.Errorf("query profit sharing failed: empty state")
	}
	return string(*resp.State), nil
}

// unfreezeProfitSharing 解冻剩余资金
// POST /v3/profitsharing/orders/unfreeze（sdk OrdersApiService.UnfreezeOrder）
func unfreezeProfitSharing(ctx context.Context, outOrderNo, wechatTransactionID, description string) error {
	if wechatClient == nil {
		return fmt.Errorf("wechat pay client not initialized")
	}
	req := profitsharing.UnfreezeOrderRequest{
		TransactionId: core.String(wechatTransactionID),
		OutOrderNo:    core.String(outOrderNo),
		Description:   core.String(description),
	}
	svc := profitsharing.OrdersApiService{Client: wechatClient}
	_, _, err := profitSharingUnfreezeOrderCall(ctx, &svc, req)
	if err != nil {
		return wechatSDKResultError("unfreeze profit sharing", err)
	}
	return nil
}

// —— 分账本地逻辑 ——

type profitSharingReceiver struct {
	Account     string `json:"account"`
	Amount      int64  `json:"amount"` // 分（单位），5% × 订单金额
	Description string `json:"description"`
	Name        string `json:"name,omitempty"`
}

type profitSharingRecord struct {
	ID                    int64
	OutProfitSharingNo    string
	OrderID               int64
	OrderNumber           string
	TenantID              int64
	ReferrerUserID        string
	ReferrerOpenid        string
	TotalYuanCents        int64
	CommissionYuanCents   int64
	Status                string
	WechatProfitSharingID string
	SettleAfter           string
	SettledAt             string
	FailReason            string
}

// markOrderForProfitSharing 在订单支付完成后标记该订单需分账给推荐人
// 由 markOrderPaid 在支付回调时调用
func markOrderForProfitSharing(ctx context.Context, orderID, tenantID int64) error {
	// 捕获一次全局 db：markOrderPaid 以异步 goroutine 调用本函数，
	// 测试在 cleanup 中会把全局 db 置 nil，若此处反复读全局可能在其中读到 nil
	// 导致后续 findReferrerForOrder 的 db.QueryRow nil 解引用崩溃（nightly 060001 复现）。
	d := db
	if d == nil {
		return nil // 无数据库连接（测试清理后异步兜底），跳过分账标记
	}
	var existing int
	if err := d.QueryRow(`SELECT COUNT(*) FROM billing_profit_sharing WHERE order_id = ?`, orderID).Scan(&existing); err != nil {
		return fmt.Errorf("check profit sharing: %w", err)
	}
	if existing > 0 {
		return nil
	}
	row, err := findReferrerForOrder(ctx, d, orderID, tenantID)
	if err != nil {
		return err
	}
	if row == nil {
		return nil // 无有效推荐关系，不分账
	}
	outNo := fmt.Sprintf("PS%d", generateSnowflakeID())

	// 落库金额用推荐政策比例，不因微信 max_ratio 暂不可用而丢掉管理员可见记录。
	commissionRate := getReferralRatePercent()
	if commissionRate < 1 {
		commissionRate = referralCommissionRateNum
	}
	totalCents := row.TotalYuanCents
	commissionCents := (totalCents*commissionRate + 50) / 100 // ROUND_HALF_UP

	delayDays := getProfitSharingDelayDays()
	settleAfter := time.Now().UTC().AddDate(0, 0, int(delayDays)).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = d.Exec(`
		INSERT INTO billing_profit_sharing
		(out_profit_sharing_no, order_id, order_number, tenant_id, referrer_user_id, referrer_openid,
		 total_yuan_cents, commission_yuan_cents, status, settle_after, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		outNo, orderID, row.OrderNumber, row.TenantID, row.ReferrerUserID, row.ReferrerOpenid,
		totalCents, commissionCents, psStatusPending, settleAfter, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert profit sharing record: %w", err)
	}
	log.Printf("[taskBill] profit sharing marked: order=%d referrer=%s commission=%d分 settle_after=%s",
		orderID, row.ReferrerUserID, commissionCents, settleAfter)
	return nil
}

type referrerOrderInfo struct {
	ReferrerUserID string
	ReferrerOpenid string
	TotalYuanCents int64
	OrderNumber    string
	TenantID       int64
}

// findReferrerForOrder 按订单买家匹配推荐边。买家租户 ≠ 推荐人租户是常态，禁止用 referrer_tenant_id = order.tenant_id。
// 分账门禁是支付时刻现查推荐人资格（ADR-0033），不读边上 commission_eligible 快照。
func findReferrerForOrder(ctx context.Context, d *sql.DB, orderID, _ int64) (*referrerOrderInfo, error) {
	row := d.QueryRow(`
		SELECT re.referrer_user_id, COALESCE(re.referrer_openid, ''), o.total_yuan_cents,
		       o.order_number, o.tenant_id
		FROM billing_resource_order o
		JOIN billing_referral_edge re
		  ON re.referred_user_id = (CAST(o.user_id AS CHAR) COLLATE utf8mb4_unicode_ci)
		WHERE o.id = ?
		  AND o.user_id > 0
	`, orderID)

	var info referrerOrderInfo
	err := row.Scan(&info.ReferrerUserID, &info.ReferrerOpenid, &info.TotalYuanCents, &info.OrderNumber, &info.TenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(info.ReferrerUserID) == "" {
		return nil, nil
	}
	if !lookupReferrerPaytimeQualification(ctx, info.ReferrerUserID) {
		return nil, nil
	}
	return &info, nil
}

func executeProfitSharing(ctx context.Context, r profitSharingRecord) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// OPT-20260821-035: 用真实微信支付单号（回调时回写 provider_capture_id），
	// 不再用 "wechat:ORD...TODO" 假值；找不到单号则失败落库，不静默当成功。
	wechatTransactionID, err := lookupWechatTransactionIDForOrder(r.OrderID)
	if err != nil {
		return err
	}

	openid, err := resolveProfitSharingReceiverOpenid(ctx, &r)
	if err != nil {
		return err
	}

	shareAmount, err := resolveOutboundShareAmount(ctx, r, wechatTransactionID)
	if err != nil {
		return err
	}

	// 标记为处理中
	markProfitSharingStatus(ctx, r.ID, psStatusProcessing, "")

	// 出站金额：floor(净额×比例)，禁止把 ROUND_HALF_UP 台账原样交给微信。
	receivers := []profitSharingReceiver{{
		Account:     openid,
		Amount:      shareAmount,
		Description: fmt.Sprintf("推荐佣金-订单%s", r.OrderNumber),
		Name:        lookupReceiverLegalName(r.ReferrerUserID),
	}}

	wechatOrderID, err := createProfitSharingOrder(ctx,
		r.OutProfitSharingNo,
		wechatTransactionID,
		r.OutProfitSharingNo,
		receivers,
	)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		UPDATE billing_profit_sharing
		SET status = ?, wechat_profit_sharing_id = ?, settled_at = ?, commission_yuan_cents = ?, updated_at = ?
		WHERE id = ?`,
		psStatusFinished, wechatOrderID, now, shareAmount, now, r.ID,
	)
	if err != nil {
		return fmt.Errorf("update profit sharing status: %w", err)
	}
	log.Printf("[taskBill] profit sharing success out_no=%s wechat_id=%s commission=%d分",
		r.OutProfitSharingNo, wechatOrderID, shareAmount)
	return nil
}

// handleInternalProcessPendingProfitSharings 一次性触发待分账扫描（OPT-20260816-029）。
// 由 taskEvents billing_profit_sharing_scan timer worker 调用，替代进程内 runProfitSharingDaemon。
// 微信未 live 时返回 200 + skipped=true（与旧 daemon 行为一致）。
func handleInternalProcessPendingProfitSharings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSONMap(w, r, http.StatusMethodNotAllowed, map[string]string{"status": "method not allowed"})
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSONMap(w, r, http.StatusForbidden, map[string]string{"status": "forbidden"})
		return
	}
	skipped := "false"
	if !wechatLiveOK {
		skipped = "true"
	}
	if err := processPendingProfitSharings(r.Context()); err != nil {
		writeErrorJSONMap(w, r, http.StatusInternalServerError, map[string]string{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "skipped": skipped})
}

func markProfitSharingStatus(ctx context.Context, id int64, status, failReason string) {
	if ctx == nil {
		ctx = context.Background()
	}
	tid := ""
	if strings.TrimSpace(failReason) != "" {
		tid = tracelog.TraceIDFromContext(ctx)
	}
	_, err := db.Exec(`
		UPDATE billing_profit_sharing SET status = ?, fail_reason = ?, fail_trace_id = ?, updated_at = ? WHERE id = ?`,
		status, failReason, tid, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		slog.ErrorContext(ctx, "profit_sharing_status_update_failed",
			"level", "error",
			"profit_sharing_id", id,
			"error", err.Error(),
		)
	}
}

// —— 配置读取 ——

// getCommissionRate 微信打款比例：min(固定 5%, 微信商户 max_ratio)。
// 微信比例不可用时返回 0，禁止用 conf 单独打款。
func getCommissionRate() int64 {
	info := resolveCommissionRate(context.Background())
	if !info.ok() {
		return 0
	}
	configured := getReferralRatePercent()
	if configured > 0 && configured < info.Percent {
		return configured
	}
	return info.Percent
}

func getProfitSharingDelayDays() int64 {
	d := int64(getReferralSettleDelayDays())
	if d < profitSharingFreezeDays {
		return profitSharingFreezeDays
	}
	return d
}

// —— 微信 V3 API HTTP 调用 ——
// 分账 live 路径已全部改走 sdk profitsharing service（OPT-20260822-053）；
// 以下仅电子发票（/v3/new-tax-control-fapiao/）仍用带签名的通用调用。

func wechatV3PostWithResp(ctx context.Context, path string, payload map[string]interface{}) (int, []byte, error) {
	body, _ := json.Marshal(payload)
	reqURL := consts.WechatPayAPIServer + path
	if wechatClient != nil {
		// OPT-20260821-035: 走签名 client，未签名裸 HTTP 会被微信拒收。
		result, err := wechatClient.Post(ctx, reqURL, payload)
		if err != nil {
			return 0, nil, err
		}
		if result == nil || result.Response == nil {
			return 0, nil, fmt.Errorf("wechat v3 post empty response")
		}
		defer result.Response.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(result.Response.Body, 1<<20))
		return result.Response.StatusCode, respBody, nil
	}
	// wechatClient 未初始化（非 live / 测试）：兜底裸 HTTP
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := wechatProfitSharingHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, respBody, nil
}

func wechatV3Get(ctx context.Context, path string) (int, []byte, error) {
	reqURL := consts.WechatPayAPIServer + path
	if wechatClient != nil {
		result, err := wechatClient.Get(ctx, reqURL)
		if err != nil {
			return 0, nil, err
		}
		if result == nil || result.Response == nil {
			return 0, nil, fmt.Errorf("wechat v3 get empty response")
		}
		defer result.Response.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(result.Response.Body, 1<<20))
		return result.Response.StatusCode, respBody, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := wechatProfitSharingHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, respBody, nil
}

func truncateBytes(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

var wechatProfitSharingHTTP = &http.Client{Timeout: 30 * time.Second}
