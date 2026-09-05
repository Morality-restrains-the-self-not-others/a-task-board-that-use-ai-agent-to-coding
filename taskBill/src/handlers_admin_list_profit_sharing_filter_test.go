package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260823-045 回归：管理端待分账队列表头列过滤 —
// handleSystemAdminListProfitSharing 支持 order_number LIKE / tenant_id 等值 /
// receiver_user_id 等值组合过滤，且与 status 筛选正交。

func TestHandleSystemAdminListProfitSharingColumnFilters(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	type row struct {
		orderID, tenantID int64
		referrer          string
		status            string
	}
	// 固定 orderID 使 order_number 可预测（ORD<orderID>）；每租户一行避免 billing_account 重复。
	rows := []row{
		{1001, 81001, "referrer-1", psStatusPending},
		{1002, 81002, "referrer-1", psStatusPending},
		{2001, 82001, "referrer-2", psStatusPending},
		{2002, 82002, "referrer-2", psStatusFinished},
	}
	for _, r := range rows {
		seedProfitSharingOrderWithLedger(t, r.tenantID, r.orderID,
			"wechat:WX"+formatID(r.orderID), "420000"+formatID(r.orderID))
		psID := seedProfitSharingRecord(t, r.orderID, r.tenantID)
		if _, err := db.Exec(`UPDATE billing_profit_sharing SET status = ?, referrer_user_id = ? WHERE id = ?`,
			r.status, r.referrer, psID); err != nil {
			t.Fatalf("update ps row: %v", err)
		}
	}

	staffList := func(q string) (int, map[string]interface{}) {
		req := staffAdminOrderReq(t, http.MethodGet, "/api/system-admin/profit-sharing/"+q)
		rec := httptest.NewRecorder()
		handleSystemAdminListProfitSharing(rec, req)
		return rec.Code, decodeJSONMap(t, rec)
	}
	itemOrderNumbers := func(body map[string]interface{}) []string {
		raw, _ := body["items"].([]interface{})
		out := make([]string, 0, len(raw))
		for _, it := range raw {
			m, _ := it.(map[string]interface{})
			out = append(out, m["order_number"].(string))
		}
		return out
	}

	// 订单号 LIKE：%ORD2001% 仅命中 ORD2001
	code, body := staffList("?status=all&order_number=ORD2001")
	if code != http.StatusOK {
		t.Fatalf("order_number status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD2001" {
		t.Fatalf("order_number=ORD2001 got %v want [ORD2001]", got)
	}

	// 租户 ID 等值：81001 仅命中 ORD1001
	code, body = staffList("?status=all&tenant_id=81001")
	if code != http.StatusOK {
		t.Fatalf("tenant status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD1001" {
		t.Fatalf("tenant_id=81001 got %v want [ORD1001]", got)
	}

	// 接收方 user_id 等值：referrer-2 默认 open 状态排除 finished → 仅 ORD2001
	code, body = staffList("?receiver_user_id=referrer-2")
	if code != http.StatusOK {
		t.Fatalf("receiver status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD2001" {
		t.Fatalf("receiver=referrer-2 open got %v want [ORD2001]", got)
	}

	// 接收方 + status=all → ORD2001 与 ORD2002 都命中
	code, body = staffList("?status=all&receiver_user_id=referrer-2")
	if code != http.StatusOK {
		t.Fatalf("receiver all status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 2 {
		t.Fatalf("receiver=referrer-2 all got %v want 2", got)
	}

	// 接收方 + status 组合（交集）：referrer-2 + pending → 仅 ORD2001
	code, body = staffList("?status=pending&receiver_user_id=referrer-2")
	if code != http.StatusOK {
		t.Fatalf("receiver pending status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD2001" {
		t.Fatalf("receiver=pending got %v want [ORD2001]", got)
	}

	// 订单号 + 租户组合（交集）：%ORD1% 命中 ORD1001/ORD1002，tenant=81001 交集 → ORD1001
	code, body = staffList("?status=all&order_number=ORD1&tenant_id=81001")
	if code != http.StatusOK {
		t.Fatalf("combined status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD1001" {
		t.Fatalf("order_number+tenant got %v want [ORD1001]", got)
	}

	// OPT-20260825-023: 商户单号 LIKE — payment_ref 去 wechat: 前缀 = WX<orderID>
	// 命中 o.out_trade_no 或 REPLACE(payment_ref,'wechat:','')，与内部 ORD- 订单号不同。
	code, body = staffList("?status=all&out_trade_no=WX1001")
	if code != http.StatusOK {
		t.Fatalf("out_trade_no status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD1001" {
		t.Fatalf("out_trade_no=WX1001 got %v want [ORD1001]", got)
	}

	// 商户单号 + 订单号组合（交集）：WX10 命中 ORD1001/ORD1002，order_number=ORD1002 交集 → ORD1002
	code, body = staffList("?status=all&out_trade_no=WX10&order_number=ORD1002")
	if code != http.StatusOK {
		t.Fatalf("out_trade_no+order_number status=%d body=%v", code, body)
	}
	if got := itemOrderNumbers(body); len(got) != 1 || got[0] != "ORD1002" {
		t.Fatalf("out_trade_no+order_number got %v want [ORD1002]", got)
	}

	// 非法 tenant_id → 400
	code, body = staffList("?tenant_id=abc")
	if code != http.StatusBadRequest {
		t.Fatalf("invalid tenant_id status=%d want 400 body=%v", code, body)
	}
}
