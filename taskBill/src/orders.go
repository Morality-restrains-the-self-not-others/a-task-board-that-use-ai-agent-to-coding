package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"tracelog"
)

// ResourceOrder 资源订单
type ResourceOrder struct {
	ID                  int64
	TenantID            int64
	OrderNumber         string
	Status              string // pending | paid | cancelled | expired
	TotalYuanCents      int64
	PaymentMethod       string
	PaymentRef          string
	OutTradeNo          string // 微信支付商户订单号
	WechatTransactionID string // 微信支付单号
	UserID              string // SSE 用户维度推送（linkWechatPayment 时回写）
	CreatedAt           string
	PaidAt              sql.NullString
	CancelledAt         sql.NullString
	BuyerNote           string // 下单施工留言；自动发货订单须为空
}

// ResourceOrderItem 订单行项
type ResourceOrderItem struct {
	ID                 int64
	OrderID            int64
	ResourceType       string // task_post | gitlab_disk | gitlab_traffic
	Quantity           int64
	UnitPriceYuanCents int64
	SubtotalYuanCents  int64
	Region             string
	DiskMonths         int64 // gitlab_disk: 购买月数（API 契约 1..36），其余资源 0
	CreatedAt          string
}

// 资源类型常量
const (
	ResourceTypeTaskPost      = "task_post"
	ResourceTypeGitlabDisk    = "gitlab_disk"
	ResourceTypeGitlabTraffic = "gitlab_traffic"
)

// 订单维度 gitlab_disk 可购月数范围（购买时长设计 2026-07-18：API 允许 1～36）。
const (
	gitlabOrderDiskMonthsMin = int64(1)
	gitlabOrderDiskMonthsMax = int64(36)
)

// 订单状态
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
	OrderStatusExpired   = "expired"
	OrderStatusRefunded  = "refunded"
)

// isDuplicateKeyError 判断错误是否为唯一键冲突，兼容各驱动错误文案：
// MySQL: Error 1062 (23000): Duplicate entry '...' for key '...'
// SQLite: UNIQUE constraint failed: ...
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

// maxOrderInsertRetries 订单主记录 UNIQUE 冲突重试上限（id 或 order_number）。
// Snowflake 正常不碰撞；保留重试以防 MACHINE_ID 冲突等极端情况。
const maxOrderInsertRetries = 5

type orderItemInput struct {
	ResourceType string `json:"resource_type"`
	Quantity     int64  `json:"quantity"`
	DiskMonths   int64  `json:"disk_months,omitempty"` // gitlab_disk 专用：购买月数
	Region       string `json:"region,omitempty"`      // gitlab_disk / gitlab_traffic 必填
}

const maxBuyerNoteRunes = 2000

// orderBuyerUserID 将买家 X-User-Id 转换为 billing_resource_order.user_id（BIGINT）。
// 空串/非数字/非正数一律回退 0，避免把 "user-9301" 之类的占位写入 BIGINT 列。
func orderBuyerUserID(raw string) int64 {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// resourceRequiresManualFulfillment 资源支付后不能自动开通、需平台施工。
// 当前仅 GitLab 磁盘（provisioning_status=pending_admin）；任务帖/流量为配额直发。
func resourceRequiresManualFulfillment(resourceType string) bool {
	return resourceType == ResourceTypeGitlabDisk
}

func orderItemsRequireManualFulfillment(items []orderItemInput) bool {
	for _, it := range items {
		if resourceRequiresManualFulfillment(it.ResourceType) {
			return true
		}
	}
	return false
}

func normalizeBuyerNote(note string, items []orderItemInput) (string, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return "", nil
	}
	if utf8.RuneCountInString(note) > maxBuyerNoteRunes {
		return "", fmt.Errorf("留言不得超过 %d 字", maxBuyerNoteRunes)
	}
	if !orderItemsRequireManualFulfillment(items) {
		return "", fmt.Errorf("仅需人工履约的资源（如 GitLab 磁盘）可添加留言")
	}
	return note, nil
}

// insertOrderWithRetry 插入订单主记录，撞号（1062/UNIQUE）时重新生成订单号与 ID 重试。
// exec 接受事务或直连执行器（*sql.Tx.Exec 与 *sql.DB.Exec 签名一致）；
// buildArgs 在每次尝试前调用，传入新 orderID/orderNumber 构造 INSERT 语句与参数。
// 与 createOrder 共用撞号重试语义（含 resource_order_number_collision 观测），
// 管理端补建路径（backfillGrantOrders/adminGrantResources）对齐。
// 订单号由 Snowflake id 派生，生成不再走 SELECT MAX，不额外占用连接池。
func insertOrderWithRetry(ctx context.Context, exec func(query string, args ...any) (sql.Result, error), tenantID int64, maxRetries int, buildArgs func(orderID int64, orderNumber string) (string, []any)) (int64, string, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		orderID := generateSnowflakeID()
		orderNumber, err := generateOrderNumberFn(tenantID, orderID)
		if err != nil {
			return 0, "", fmt.Errorf("生成订单号失败: %w", err)
		}
		query, args := buildArgs(orderID, orderNumber)
		if _, err := exec(query, args...); err != nil {
			if isDuplicateKeyError(err) {
				lastErr = err
				tracelog.LogForwardStage(ctx, "resource_order_number_collision", map[string]any{
					"tenant_id":    formatID(tenantID),
					"order_number": orderNumber,
					"attempt":      attempt,
				})
				continue
			}
			return 0, "", err
		}
		return orderID, orderNumber, nil
	}
	return 0, "", fmt.Errorf("创建订单失败(已重试%d次): %w", maxRetries, lastErr)
}

// createOrder 创建订单（无施工留言）。
func createOrder(ctx context.Context, tenantID int64, items []orderItemInput) (*ResourceOrder, []ResourceOrderItem, error) {
	return createOrderWithNote(ctx, tenantID, items, "", "")
}

// createOrderWithNote 创建订单，buyerNote 仅在含人工履约资源时可非空。
// authorUserID 为下单人 user id，用于把留言写入首条订单评论线程（可为空，
// 为空时以 tenant:{id} 占位）。
func createOrderWithNote(ctx context.Context, tenantID int64, items []orderItemInput, buyerNote, authorUserID string) (*ResourceOrder, []ResourceOrderItem, error) {
	buyerNote, err := normalizeBuyerNote(buyerNote, items)
	if err != nil {
		return nil, nil, err
	}
	for i := range items {
		if items[i].ResourceType == ResourceTypeGitlabDisk || items[i].ResourceType == ResourceTypeGitlabTraffic {
			region, err := getGitlabRegionBySlug(items[i].Region)
			if err != nil {
				return nil, nil, err
			}
			if !CanUseGitlabRegion(region.AccessMode, testerFromContext(ctx)) {
				slog.WarnContext(ctx, "gitlab_region_dev_mode_forbidden",
					"level", "warn",
					"region", region.Slug,
					"access_mode", region.AccessMode,
					"is_tester", testerFromContext(ctx),
				)
				return nil, nil, errGitlabRegionDevModeForbidden
			}
			items[i].Region = region.Slug
		}
	}
	// 确保计费账户存在
	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		return nil, nil, err
	}

	// VIP 会员等级权限校验：检查每个资源类型是否允许当前会员等级购买
	for _, item := range items {
		if err := checkMembershipPurchasePermission(tenantID, item.ResourceType); err != nil {
			return nil, nil, err
		}
	}

	// 读取资源定价约束（最低消费门槛、前置依赖）
	pricing, _ := getCurrentResourcePricing()

	// 根据 billing_unit 当前实时价格逐项计价
	var orderItems []ResourceOrderItem
	totalCents := int64(0)
	for _, item := range items {
		unitPriceCents, err := getUnitPriceCents(item.ResourceType)
		if err != nil {
			return nil, nil, fmt.Errorf("读取 %s 单价失败: %w", item.ResourceType, err)
		}
		if item.Quantity <= 0 {
			return nil, nil, fmt.Errorf("%s 数量须 >= 1", item.ResourceType)
		}
		// GitLab 流量费：须先购买 GitLab 磁盘
		if item.ResourceType == ResourceTypeGitlabTraffic && pricing != nil && pricing.GitlabTrafficRequiresUnitType == "gitlab_disk" {
			hasDisk, err := tenantHasGitlabDisk(tenantID)
			if err != nil {
				return nil, nil, fmt.Errorf("查询 GitLab 磁盘购买状态失败: %w", err)
			}
			if !hasDisk {
				return nil, nil, fmt.Errorf("须先购买 GitLab 磁盘，才可购买 GitLab 流量费")
			}
		}
		months := int64(1)
		if item.ResourceType == ResourceTypeGitlabDisk {
			if err := validateGitlabDiskPurchaseQuantity(item.Quantity); err != nil {
				slog.WarnContext(ctx, "gitlab_disk_below_min_purchase",
					"level", "warn",
					"quantity", item.Quantity,
					"min_gb", GitlabDiskMinPurchaseGB,
				)
				return nil, nil, err
			}
			// OPT-20260903-010: gitlab_disk 小计 = 单价 × GB × 购买月数，缺省 1，校验 1..36。
			// 否则 OrderCreate 展示金额（price×GB×months）与实扣（price×GB）分叉。
			if item.DiskMonths > 0 {
				months = item.DiskMonths
			}
			if months < gitlabOrderDiskMonthsMin || months > gitlabOrderDiskMonthsMax {
				return nil, nil, fmt.Errorf("gitlab_disk disk_months 须在 %d～%d 之间", gitlabOrderDiskMonthsMin, gitlabOrderDiskMonthsMax)
			}
		}
		subtotal := unitPriceCents * item.Quantity * months
		orderItems = append(orderItems, ResourceOrderItem{
			ResourceType:       item.ResourceType,
			Quantity:           item.Quantity,
			UnitPriceYuanCents: unitPriceCents,
			SubtotalYuanCents:  subtotal,
			Region:             strings.TrimSpace(item.Region),
			DiskMonths:         months,
		})
		totalCents += subtotal
	}

	now := utcNow()

	// 重试循环：UNIQUE(id/order_number) 冲突时换新 Snowflake 再插入。
	// 订单号由 id 派生，不再 SELECT MAX，可在拿到 orderID 后立即生成。
	var lastErr error
	var orderID int64
	var orderNumber string
	for attempt := 0; attempt < maxOrderInsertRetries; attempt++ {
		orderID = generateSnowflakeID()
		orderNumber, err = generateOrderNumberFn(tenantID, orderID)
		if err != nil {
			return nil, nil, fmt.Errorf("生成订单号失败: %w", err)
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, nil, err
		}
		defer tx.Rollback()

		// OPT-20260821-032: 下单即写入买家 user_id（X-User-Id），
		// 使 markOrderPaid 入账的 resource_purchase 流水始终可关联购买人。
		_, err = tx.Exec(`
			INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, buyer_note, user_id, created_at)
			VALUES (?, ?, ?, 'pending', ?, ?, ?, ?)`,
			orderID, tenantID, orderNumber, totalCents, buyerNote, orderBuyerUserID(authorUserID), now,
		)
		if err != nil {
			tx.Rollback()
			if isDuplicateKeyError(err) {
				lastErr = err
				tracelog.LogForwardStage(ctx, "resource_order_number_collision", map[string]any{
					"tenant_id":    formatID(tenantID),
					"order_number": orderNumber,
					"attempt":      attempt,
				})
				continue
			}
			return nil, nil, fmt.Errorf("创建订单失败: %w", err)
		}

		for _, oi := range orderItems {
			itemID := generateSnowflakeID()
			_, err = tx.Exec(`
				INSERT INTO billing_resource_order_item (id, order_id, resource_type, quantity, unit_price_yuan_cents, subtotal_yuan_cents, region, disk_months, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				itemID, orderID, oi.ResourceType, oi.Quantity, oi.UnitPriceYuanCents, oi.SubtotalYuanCents, oi.Region, oi.DiskMonths, now,
			)
			if err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("创建订单行项失败: %w", err)
			}
		}

		// OPT-20260819-027: 下单施工留言写入首条订单评论线程，施工人员只看评论区也能看到。
		if buyerNote != "" {
			authorID := strings.TrimSpace(authorUserID)
			if authorID == "" {
				authorID = fmt.Sprintf("tenant:%d", tenantID)
			}
			if _, err := tx.Exec(`
				INSERT INTO billing_order_comment
					(id, tenant_id, order_id, author_user_id, author_side, content, created_at)
				VALUES (?, ?, ?, ?, 'tenant', ?, ?)`,
				generateSnowflakeID(), tenantID, orderID, authorID, buyerNote, now,
			); err != nil {
				tx.Rollback()
				return nil, nil, fmt.Errorf("写入下单施工留言评论失败: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			if isDuplicateKeyError(err) {
				lastErr = err
				continue
			}
			return nil, nil, fmt.Errorf("提交订单失败: %w", err)
		}

		lastErr = nil
		break
	}
	if lastErr != nil {
		tracelog.LogForwardStage(ctx, "resource_order_create_failed", map[string]any{
			"tenant_id": formatID(tenantID),
			"error":     lastErr.Error(),
			"attempts":  maxOrderInsertRetries,
		})
		return nil, nil, fmt.Errorf("创建订单失败(已重试%d次): %w", maxOrderInsertRetries, lastErr)
	}

	order := &ResourceOrder{
		ID:             orderID,
		TenantID:       tenantID,
		OrderNumber:    orderNumber,
		Status:         OrderStatusPending,
		TotalYuanCents: totalCents,
		UserID:         billingBuyerUserID(authorUserID),
		CreatedAt:      now,
		BuyerNote:      buyerNote,
	}

	gitlabRegions := make([]string, 0, len(orderItems))
	for _, oi := range orderItems {
		if oi.Region != "" {
			gitlabRegions = append(gitlabRegions, oi.ResourceType+":"+oi.Region)
		}
	}
	tracelog.LogForwardStage(ctx, "resource_order_created", map[string]any{
		"order_id":       formatID(orderID),
		"order_number":   orderNumber,
		"tenant_id":      formatID(tenantID),
		"total_cents":    totalCents,
		"item_count":     len(orderItems),
		"regions":        gitlabRegions,
		"buyer_note_len": utf8.RuneCountInString(buyerNote),
	})

	return order, orderItems, nil
}

// tenantOrderFocusOffset 计算目标订单在「created_at DESC, id DESC」列表中的 0-based 下标，
// 再对齐到 pageSize 的页起始 offset。订单不存在或不属于该租户时 ok=false。
func tenantOrderFocusOffset(tenantID, orderID int64, statusFilter string, pageSize int) (offset int, ok bool, err error) {
	if pageSize <= 0 {
		pageSize = 15
	}
	var createdAt string
	var status string
	err = db.QueryRow(
		`SELECT created_at, status FROM billing_resource_order WHERE id = ? AND tenant_id = ?`,
		orderID, tenantID,
	).Scan(&createdAt, &status)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if statusFilter != "" && status != statusFilter {
		// 当前状态筛选会把目标订单滤掉，无法定位
		return 0, false, nil
	}

	var rank int64
	if statusFilter != "" {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM billing_resource_order
			 WHERE tenant_id = ? AND status = ?
			   AND (created_at > ? OR (created_at = ? AND id > ?))`,
			tenantID, statusFilter, createdAt, createdAt, orderID,
		).Scan(&rank)
	} else {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM billing_resource_order
			 WHERE tenant_id = ?
			   AND (created_at > ? OR (created_at = ? AND id > ?))`,
			tenantID, createdAt, createdAt, orderID,
		).Scan(&rank)
	}
	if err != nil {
		return 0, false, err
	}
	aligned := int(rank/int64(pageSize)) * pageSize
	return aligned, true, nil
}

// adminOrderFocusOffset 跨租户订单列表的 order_id 对齐（系统管理员用）：
// 与 tenantOrderFocusOffset 同款排序规则（created_at DESC, id DESC）下，
// 计算目标订单的全局 rank 并对齐到所在页。
func adminOrderFocusOffset(orderID int64, statusFilter string, pageSize int) (offset int, ok bool, err error) {
	if pageSize <= 0 {
		pageSize = 15
	}
	var createdAt string
	var status string
	err = db.QueryRow(
		`SELECT created_at, status FROM billing_resource_order WHERE id = ?`,
		orderID,
	).Scan(&createdAt, &status)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if statusFilter != "" && status != statusFilter {
		// 当前状态筛选会把目标订单滤掉，无法定位
		return 0, false, nil
	}

	var rank int64
	if statusFilter != "" {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM billing_resource_order
			 WHERE status = ?
			   AND (created_at > ? OR (created_at = ? AND id > ?))`,
			statusFilter, createdAt, createdAt, orderID,
		).Scan(&rank)
	} else {
		err = db.QueryRow(
			`SELECT COUNT(*) FROM billing_resource_order
			 WHERE (created_at > ? OR (created_at = ? AND id > ?))`,
			createdAt, createdAt, orderID,
		).Scan(&rank)
	}
	if err != nil {
		return 0, false, err
	}
	aligned := int(rank/int64(pageSize)) * pageSize
	return aligned, true, nil
}
