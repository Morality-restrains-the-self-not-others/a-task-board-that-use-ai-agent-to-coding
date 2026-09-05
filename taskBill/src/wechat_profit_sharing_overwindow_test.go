package main

import (
	"strings"
	"testing"
	"time"
)

// OPT-20260823-050：支付超过 30 天窗口仍未成功分账（ps 非 finished）才计入告警；
// 窗口内 pending / 已 finished 均不计入。
func TestQueryOverWindowProfitSharingsCountsOnlyOutsideWindow(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	// 31 天 paid + ps failed → 计入
	order31Failed := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, 96301, order31Failed, "wechat:WX-OW1", "420000ow0001")
	setOrderPaidAt(t, order31Failed, rfc3339DaysAgo(31))
	ps31 := seedProfitSharingRecord(t, order31Failed, 96301)
	setProfitSharingRow(t, ps31, psStatusFailed, rfc3339DaysFromNow(1), "")

	// 26 天 paid + ps pending → 窗口内，不计入
	order26Pending := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, 96302, order26Pending, "wechat:WX-OW2", "420000ow0002")
	setOrderPaidAt(t, order26Pending, rfc3339DaysAgo(26))
	ps26 := seedProfitSharingRecord(t, order26Pending, 96302)
	setProfitSharingRow(t, ps26, psStatusPending, rfc3339DaysFromNow(10), "")

	// 31 天 paid + ps finished → 不计入
	order31Finished := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, 96303, order31Finished, "wechat:WX-OW3", "420000ow0003")
	setOrderPaidAt(t, order31Finished, rfc3339DaysAgo(31))
	ps31f := seedProfitSharingRecord(t, order31Finished, 96303)
	setProfitSharingRow(t, ps31f, psStatusFinished, rfc3339DaysAgo(1), "wx-ow-finished")

	count, items, err := queryOverWindowProfitSharings(time.Now().UTC())
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d want 1（仅 31 天 failed 计入）", count)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].OrderID != order31Failed {
		t.Fatalf("order_id=%d want %d", items[0].OrderID, order31Failed)
	}
	if items[0].Status != psStatusFailed {
		t.Fatalf("status=%s want failed", items[0].Status)
	}
	// 明细字段只含 order_id/status/paid_at，不含 openid
	if strings.Contains(strings.Join([]string{items[0].Status, items[0].PaidAt}, ","), "openid") {
		t.Fatal("over-window row leaked sensitive field")
	}
}

// 边界：paid_at 恰好在 30 天整点时计入（<= 窗口截止）。
func TestQueryOverWindowProfitSharingsIncludesExactly30Days(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	order := generateSnowflakeID()
	seedProfitSharingOrderWithLedger(t, 96304, order, "wechat:WX-OW4", "420000ow0004")
	setOrderPaidAt(t, order, rfc3339DaysAgo(30))
	ps := seedProfitSharingRecord(t, order, 96304)
	setProfitSharingRow(t, ps, psStatusPending, rfc3339DaysFromNow(1), "")

	count, _, err := queryOverWindowProfitSharings(time.Now().UTC())
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d want 1（paid_at 恰 30 天应计入）", count)
	}
}
