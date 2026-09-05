package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// processPendingProfitSharings 扫描到期 pending，并叠加 25 天窗口兜底。
// OPT-20260816-029：进程内 daemon 已迁出，由 taskEvents billing_profit_sharing_scan timer 触发。
func processPendingProfitSharings(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !wechatLiveOK {
		log.Printf("[taskBill] profit sharing: wechat not live, skipping")
		return nil
	}
	now := time.Now().UTC()
	// OPT-20260823-050：窗口外未成功分账的可观测告警（只读，不影响主流程）。
	logOverWindowProfitSharings(now)
	// OPT-20260823-042：无佣金行微信单解冻失败补偿（依赖微信同号幂等）。
	if err := compensateUnfreezeRemainderForPaidWechatOrders(now); err != nil {
		log.Printf("[taskBill] profit sharing unfreeze compensate scan failed: %v", err)
	}
	marked, err := markUnmarkedFallbackProfitSharings(now)
	if err != nil {
		return err
	}
	// 15 天解冻后改由推荐人手动分账；扫描只跑 25 天窗口兜底，不再执行 due pending。
	fallback, err := queryFallbackProfitSharings(now)
	if err != nil {
		return err
	}
	records := mergeProfitSharingRecords(fallback)
	log.Printf("[taskBill] profit sharing scan: due_skipped=1 fallback=%d unmarked_marked=%d merged=%d",
		len(fallback), marked, len(records))
	if len(records) == 0 {
		return nil
	}

	for _, r := range records {
		rec := r
		if strings.TrimSpace(rec.ReferrerOpenid) == "" {
			if _, err := resolveProfitSharingReceiverOpenid(ctx, &rec); err != nil {
				log.Printf("[taskBill] profit sharing skip wechat empty openid order=%d", rec.OrderID)
				continue
			}
		}
		if err := executeProfitSharing(ctx, rec); err != nil {
			log.Printf("[taskBill] profit sharing failed out_no=%s err=%v", r.OutProfitSharingNo, err)
			markProfitSharingStatus(ctx, r.ID, psStatusFailed, err.Error())
		}
	}
	return nil
}

func queryDuePendingProfitSharings(now time.Time) ([]profitSharingRecord, error) {
	rows, err := db.Query(`
		SELECT id, out_profit_sharing_no, order_id, order_number, tenant_id,
		       referrer_user_id, referrer_openid, commission_yuan_cents, total_yuan_cents
		FROM billing_profit_sharing
		WHERE status = ? AND settle_after <= ?
		ORDER BY created_at ASC
		LIMIT 50`, psStatusPending, now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query pending profit sharings: %w", err)
	}
	defer rows.Close()
	return scanProfitSharingWorkItems(rows)
}
