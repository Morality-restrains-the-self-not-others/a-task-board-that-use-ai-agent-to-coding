package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// ---- 订单 CRUD ----

func handlePayOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	order, items, err := loadOrder(tid, orderID)
	if err != nil {
		writeOrderLookupMiss(w, r, tid, orderID)
		return
	}
	if order.Status != OrderStatusPending {
		writeErrorJSON(w, http.StatusBadRequest, "订单状态不是待支付", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	paymentMethod := stringField(body, "payment_method")
	userID := r.Header.Get("X-User-Id")
	consentID := stringField(body, "consent_id")
	// 微信 Native amount.total 单位为分，原样传订单分金额；KYC 限额接口仍按整数元上取整。
	amountFen := order.TotalYuanCents
	kycYuan := kycAmountYuanCeil(order.TotalYuanCents)

	// 支付服务条款签署门禁（有生效条款时必须已签署）
	resolvedConsentID, cerr := checkPaymentTermsConsentGate(r.Context(), userID, consentID)
	if cerr != nil {
		writeConsentGateDenied(w, cerr)
		return
	}
	if resolvedConsentID != "" {
		if aerr := attachConsentToOrder(r.Context(), orderID, resolvedConsentID); aerr != nil {
			slog.WarnContext(r.Context(), "attach_order_consent_failed",
				"level", "warn",
				"order_id", formatID(orderID),
				"error", aerr.Error(),
			)
		}
	}

	// KYC 门禁：身份等级 + AML + 单笔/单日限额
	if err := checkKycPaymentGate(r.Context(), userID, kycYuan); err != nil {
		writeKycGateDenied(w, err)
		return
	}
	// SMS 门禁：手机号短信验证
	if err := checkSmsPaymentGate(r.Context(), userID); err != nil {
		writeSmsGateDenied(w, err)
		return
	}

	desc := fmt.Sprintf("资源购买-%s", order.OrderNumber)

	switch paymentMethod {
	case "wechat":
		if !wechatConfigured() {
			writeErrorJSON(w, http.StatusServiceUnavailable, "微信支付未配置", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		notifyURL := wechatCfg.NotifyURL
		outTradeNo, codeURL, err := wechatPrepay(r.Context(), tid, userID, amountFen, desc, notifyURL)
		if err != nil {
			// OPT-20260823-034: 剥离 WeChat HTTP dump（含签名头）后再写浏览器 JSON。
			_, msg := paymentActionClientError(err)
			writeErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("微信支付下单失败: %s", msg), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		// 关联订单到微信支付 pending 状态
		linkWechatPendingToOrder(outTradeNo, orderID, tid)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"order_id":       formatID(order.ID),
			"order_number":   order.OrderNumber,
			"total_yuan":     centsToYuanStr(order.TotalYuanCents),
			"payment_method": "wechat",
			"out_trade_no":   outTradeNo,
			"code_url":       codeURL,
			"mode":           wechatCfg.Mode,
		})

	case "paypal":
		paypalResult, err := createPaypalForOrder(r.Context(), order, tid, userID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, fmt.Sprintf("PayPal 下单失败: %v", err), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, paypalResult)

	default:
		writeErrorJSON(w, http.StatusBadRequest, "payment_method 须为 wechat 或 paypal", tracelog.TraceIDFromContext(r.Context()))
	}

	_ = items // 行项信息在前端订单详情中展示
}

// linkWechatPendingToOrder 将微信支付的 outTradeNo 关联到资源订单（支付回调时用于资源发放）。
// v64: 同步持久化 billing_payment_pending 关联 + 订单 user_id（下单会话归属）。
func linkWechatPendingToOrder(outTradeNo string, orderID, tenantID int64) {
	wechatMu.Lock()
	p, ok := wechatPending[outTradeNo]
	if ok {
		p.OrderID = orderID
		p.TenantID = tenantID
		wechatPending[outTradeNo] = p
	}
	wechatMu.Unlock()
	if !ok {
		log.Printf("[taskBill] link wechat payment %s: pending not in memory", outTradeNo)
		return
	}
	if _, err := db.Exec(`
		UPDATE billing_payment_pending SET order_id = ?, tenant_id = ?
		WHERE out_trade_no = ?`, orderID, tenantID, outTradeNo); err != nil {
		log.Printf("[taskBill] persist pending order link %s failed: %v", outTradeNo, err)
	}
	if p.UserID != "" {
		if _, err := db.Exec(`
			UPDATE billing_resource_order SET user_id = ? WHERE id = ?`, p.UserID, orderID); err != nil {
			log.Printf("[taskBill] persist order user_id %d failed: %v", orderID, err)
		}
	}
	log.Printf("[taskBill] linked wechat payment %s to order %d", outTradeNo, orderID)
}

// createPaypalForOrder 为资源订单创建 PayPal 支付
func createPaypalForOrder(ctx interface{}, order *ResourceOrder, tenantID int64, userID string) (map[string]interface{}, error) {
	// PayPal 支付本地不实现，返回参数让前端调 PayPal SDK
	return map[string]interface{}{
		"order_id":       formatID(order.ID),
		"order_number":   order.OrderNumber,
		"total_yuan":     centsToYuanStr(order.TotalYuanCents),
		"payment_method": "paypal",
		"amount_yuan":    centsToYuanStr(order.TotalYuanCents),
		"description":    fmt.Sprintf("资源购买-%s", order.OrderNumber),
	}, nil
}

// ---- 支付回调 ----

// handleOrderPaymentCallback POST /api/billing/orders/{orderId}/callback/
// 用于支付网关回调时标记订单已支付并发放资源
func handleOrderPaymentCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	paymentMethod := stringField(body, "payment_method")
	paymentRef := stringField(body, "payment_ref")

	if err := markOrderPaid(r.Context(), orderID, paymentMethod, paymentRef, 0); err != nil {
		// tenant_id=0 让 markOrderPaid 自己从订单中获取
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	order, items, err := loadOrderByID(orderID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "支付成功，资源已发放",
		"order":   orderJSON(order, items),
	})
}

// ---- 资源配额查询 ----

// handleResourceQuotas GET /api/tenant/{id}/billing/quotas/
func handleResourceQuotas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeResourceQuotas(w, r, tid)
}

func writeResourceQuotas(w http.ResponseWriter, r *http.Request, tid int64) {
	if err := ensureTaskPostPurchaseLots(r.Context(), tid); err != nil {
		slog.WarnContext(r.Context(), "task_post_purchase_lots_backfill_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"error", err.Error(),
		)
	}

	taskQuota := int64(0)
	gifted := int64(0)
	purchased := int64(0)
	if split, err := getTaskPostQuotaSplit(tid); err == nil {
		taskQuota = split.Total
		gifted = split.Gifted
		purchased = split.Purchased
	}

	gitlabView, err := gitlabResourceView(tid)
	if err != nil {
		gitlabView = map[string]interface{}{}
	}

	// OPT-20260818-022：按区列表（多区域时首页按区展示磁盘/流量），旧聚合字段保留兼容。
	gitlabResources, listErr := listGitlabResourceViews(tid, requestIsTester(r))
	if listErr != nil {
		slog.Warn("gitlab_quotas_list_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"error", listErr.Error(),
		)
		gitlabResources = []map[string]interface{}{}
	}

	taskPrice, _ := getUnitPriceCents(ResourceTypeTaskPost)
	diskPrice, _ := getUnitPriceCents(ResourceTypeGitlabDisk)
	trafficPrice, _ := getUnitPriceCents(ResourceTypeGitlabTraffic)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_post_quota":                 taskQuota,
		"task_post_quota_gifted":          gifted,
		"task_post_quota_purchased":       purchased,
		"gitlab_resources":                gitlabResources,
		"gitlab_disk_gb":                  gitlabView["disk_gb"],
		"gitlab_disk_months":              gitlabView["disk_months"],
		"gitlab_disk_expires_at":          gitlabView["disk_expires_at"],
		"gitlab_disk_used_gb":             gitlabView["disk_used_gb"],
		"gitlab_traffic_prepaid_gb":       gitlabView["traffic_prepaid_gb"],
		"gitlab_traffic_used_gb":          gitlabView["traffic_used_gb"],
		"task_post_unit_price_cents":      taskPrice,
		"gitlab_disk_unit_price_cents":    diskPrice,
		"gitlab_traffic_unit_price_cents": trafficPrice,
	})
}

// ---- 工具函数 ----

// parseOrderIDFromPath 从 URL 路径中提取 order ID
func parseOrderIDFromPath(path, segment string) (int64, error) {
	idx := strings.Index(path, segment)
	if idx < 0 {
		return 0, fmt.Errorf("无效的订单路径")
	}
	rest := path[idx+len(segment):]
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("缺少 order_id")
	}
	return strconv.ParseInt(parts[0], 10, 64)
}

// handleOrderCancel POST /api/tenant/{id}/billing/orders/{orderId}/cancel/
func handleOrderCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	order, _, err := loadOrder(tid, orderID)
	if err != nil {
		writeOrderLookupMiss(w, r, tid, orderID)
		return
	}
	if order.Status != OrderStatusPending {
		writeErrorJSON(w, http.StatusBadRequest, "只能取消待支付订单", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	now := utcNow()
	_, err = db.Exec(`UPDATE billing_resource_order SET status = 'cancelled', cancelled_at = ? WHERE id = ? AND tenant_id = ?`, now, orderID, tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	// OPT-20260820-018: 取消订单时把同订单的 pending 二维码行一并标 cancelled。
	if err := closePaymentPendingForOrder(r.Context(), db, orderID, "paid", "cancelled"); err != nil {
		slog.WarnContext(r.Context(), "close_payment_pending_on_cancel_failed",
			"level", "warn",
			"order_id", formatID(orderID),
			"error", err.Error(),
		)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "message": "订单已取消"})
}

// handleInternalFulfillOrder POST /api/internal/taskbill/orders/{id}/fulfill/
func handleInternalFulfillOrder(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseIDField(body["order_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid order_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	paymentMethod := stringField(body, "payment_method")
	paymentRef := stringField(body, "payment_ref")
	tenantID, _ := parseIDField(body["tenant_id"])

	if err := markOrderPaid(r.Context(), orderID, paymentMethod, paymentRef, tenantID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	order, items, err := loadOrderByID(orderID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "订单已支付，资源已发放",
		"order":   orderJSON(order, items),
	})
}

// ---- 传统充值兼容（已废弃，保留以兼容旧版） ----
// handleLegacyRechargeRedirect 将旧充值请求重定向到提示用户使用新订单流程
// 不执行此操作，仅保留函数签名供编译
var _ = func() int { return 0 }
var _ = json.Marshal
