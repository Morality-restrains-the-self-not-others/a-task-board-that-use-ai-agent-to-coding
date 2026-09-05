package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// markOrderPaid 标记订单已支付并发放资源配额（幂等: 已 paid 直接成功，微信重试不重复发放）。
// tenantID>0 时按分片键加载；tenantID<=0（内部确认支付）先按主键加载，再用行上 tenant_id 发放配额。
func markOrderPaid(ctx context.Context, orderID int64, paymentMethod, paymentRef string, tenantID int64) error {
	var order *ResourceOrder
	var items []ResourceOrderItem
	var err error
	if tenantID > 0 {
		order, items, err = loadOrder(tenantID, orderID)
	} else {
		order, items, err = loadOrderByID(orderID)
	}
	if err != nil {
		return err
	}
	if order.Status == OrderStatusPaid {
		log.Printf("[taskBill] markOrderPaid idempotent: order %d already paid", orderID)
		return nil
	}
	if order.Status != OrderStatusPending {
		return fmt.Errorf("订单状态不是待支付: %s", order.Status)
	}
	if tenantID > 0 && order.TenantID != tenantID {
		return fmt.Errorf("订单不存在")
	}
	tenantID = order.TenantID
	if tenantID <= 0 {
		return fmt.Errorf("订单不存在")
	}

	now := utcNow()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 标记订单已支付
	_, err = tx.Exec(`
		UPDATE billing_resource_order SET status = 'paid', payment_method = ?, payment_ref = ?, paid_at = ?
		WHERE id = ? AND tenant_id = ? AND status = 'pending'`,
		paymentMethod, paymentRef, now, orderID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 发放资源配额
	for _, item := range items {
		switch item.ResourceType {
		case ResourceTypeTaskPost:
			// 增加任务帖预购配额并建立购买批次
			_, err = tx.Exec(
				`UPDATE billing_account SET task_post_quota = task_post_quota + ?, updated_at = ? WHERE tenant_id = ?`,
				item.Quantity, now, tenantID,
			)
			if err == nil {
				err = insertTaskPostPurchaseLotTx(tx, tenantID, orderID, item.Quantity, now)
			}
		case ResourceTypeGitlabDisk:
			regionSlug := strings.TrimSpace(item.Region)
			if regionSlug == "" {
				return fmt.Errorf("region required")
			}
			// OPT-20260903-010: 用订单行月数写 disk_months，不再写死 1。
			diskMonths := item.DiskMonths
			if diskMonths <= 0 {
				// 077 迁移前旧订单行无 disk_months，按 1 兜底（与旧实扣语义一致）
				diskMonths = 1
			}
			currentExpiresAt := ""
			if err := tx.QueryRow(
				`SELECT COALESCE(disk_expires_at, '') FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`, tenantID, regionSlug,
			).Scan(&currentExpiresAt); err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("查询现有磁盘资源失败: %w", err)
			}
			// OPT-20260903-012: 到期日从「开通日起算」。
			// 首次购买（无既有到期时间，pending_admin 待管理员开通）→ disk_expires_at 留空，
			// 由 handleAdminProvisionGitlabResource 开通成功时按 disk_months 计算；
			// 已开通/历史行已有到期时间（active 续费加购）→ 按现有到期时间顺延，管理员无需再开通。
			expiresAt := ""
			if currentExpiresAt != "" {
				expiresAt = diskExpiresAtFromMonths(diskMonths, currentExpiresAt)
			}
			_, err = tx.Exec(`
				INSERT INTO billing_tenant_gitlab_resource (
					tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at, provisioning_status, created_at, updated_at
				) VALUES (?, ?, ?, 0, ?, ?, 'pending_admin', ?, ?)
				ON DUPLICATE KEY UPDATE
					disk_gb = disk_gb + VALUES(disk_gb),
					disk_months = VALUES(disk_months),
					disk_expires_at = VALUES(disk_expires_at),
					updated_at = VALUES(updated_at)`,
				tenantID, regionSlug, item.Quantity, diskMonths, expiresAt, now, now,
			)
		case ResourceTypeGitlabTraffic:
			regionSlug := strings.TrimSpace(item.Region)
			if regionSlug == "" {
				return fmt.Errorf("region required")
			}
			_, err = tx.Exec(`
				INSERT INTO billing_tenant_gitlab_resource (
					tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at, created_at, updated_at
				) VALUES (?, ?, 0, ?, 0, '', ?, ?)
				ON DUPLICATE KEY UPDATE
					traffic_prepaid_gb = traffic_prepaid_gb + VALUES(traffic_prepaid_gb),
					updated_at = VALUES(updated_at)`,
				tenantID, regionSlug, item.Quantity, now, now,
			)
		}
		if err != nil {
			return fmt.Errorf("发放资源配额失败 (%s): %w", item.ResourceType, err)
		}
	}

	// 记录交易流水（视为购买消费，但没有 balance 扣减）
	tid := generateSnowflakeID()
	buyerID := billingBuyerUserID(order.UserID)
	if buyerID == "" {
		// OPT-20260821-032: 存量订单创建时未写 user_id 时，从支付会话
		// billing_payment_pending 回填真实买家，并同步写回订单行。
		if uid := lookupOrderBuyerUserIDFromPending(orderID); uid != "" {
			buyerID = uid
			if _, uerr := tx.Exec(`UPDATE billing_resource_order SET user_id = ? WHERE id = ?`, uid, orderID); uerr != nil {
				return fmt.Errorf("回填订单买家失败: %w", uerr)
			}
		}
	}
	sourceTxnID := orderReferralSourceTxnID(order.OrderNumber)
	_, err = tx.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, description, transaction_id, created_at, usage_amount,
			user_id, related_order_id
		) VALUES (?, (SELECT id FROM billing_account WHERE tenant_id = ?), 'consumption', ?,
			(SELECT balance FROM billing_account WHERE tenant_id = ?),
			(SELECT balance FROM billing_account WHERE tenant_id = ?),
			'resource_purchase', ?, ?, ?, 0, ?, ?)`,
		tid, tenantID, order.TotalYuanCents, tenantID, tenantID,
		fmt.Sprintf("订单 %s 资源购买", order.OrderNumber),
		sourceTxnID, now, nullStr(buyerID), order.ID,
	)
	if err != nil {
		return fmt.Errorf("记录交易流水失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	publishGitlabManualNodeFulfillmentIfNeeded(ctx, tenantID, orderID, items)

	// Ensure payment ledger for WeChat/PayPal so order refunds can原路退回.
	order.PaymentMethod = paymentMethod
	order.PaymentRef = paymentRef
	order.Status = OrderStatusPaid
	if channel, _, ok := orderRefundChannel(paymentMethod, paymentRef); ok && (channel == "wechat" || channel == "paypal") {
		acc, _, aerr := getOrCreateBillingAccount(tenantID, false)
		if aerr == nil && acc != nil {
			if _, lerr := ensureOrderPaymentLedger(context.Background(), db, tenantID, acc.ID, order, now); lerr != nil {
				log.Printf("[taskBill] ensureOrderPaymentLedger order=%d err=%v", orderID, lerr)
			}
		}
	}

	// 标记推荐佣金分账（异步，失败不阻塞订单完成）。
	// 无佣金行时再解冻剩余资金。不解冻放在 already-paid 重放路径，
	// 以免并发回调在 insert 佣金行之前把资金解冻。
	go func() {
		if err := markOrderForProfitSharing(context.Background(), orderID, tenantID); err != nil {
			log.Printf("[taskBill] mark profit sharing order=%d err=%v", orderID, err)
		}
		maybeUnfreezeWechatRemainderIfNoReceiver(context.Background(), orderID, paymentMethod)
	}()

	if buyerID != "" && order.TotalYuanCents > 0 {
		tryAccrueReferralFromConsumption(
			context.Background(), buyerID, sourceTxnID, tid, order.TotalYuanCents, now,
		)
	}

	// OPT-20260808-016: 支付落地后经 taskSSE 推送 order_paid（异步，失败不阻塞）。
	// user_id 为空（未 link 微信）时 publishOrderPaidSSE 内部跳过。
	if order.UserID != "" {
		go publishOrderPaidSSEFn(context.Background(), order)
	}

	// 同步会员累计消费并检查自动升级（支付完成后）
	if _, err := syncMembershipConsumption(context.Background(), tenantID); err != nil {
		log.Printf("[taskBill] syncMembershipConsumption tenant=%d err=%v", tenantID, err)
	}

	return nil
}

// lookupOrderBuyerUserIDFromPending 支付回调兜底：订单 user_id 为空时，从
// billing_payment_pending（下单支付会话写入）按 order_id 取最近一条真实买家。
// 无记录/无 user_id 返回空串。
func lookupOrderBuyerUserIDFromPending(orderID int64) string {
	var uid string
	err := db.QueryRow(`
		SELECT user_id FROM billing_payment_pending
		WHERE order_id = ? AND user_id != 0
		ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&uid)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(uid)
}
