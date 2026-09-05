package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestInternalUserAccountDeletionBlockersRequiresSecret(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/users/123/account-deletion-blockers/", nil)
	w := httptest.NewRecorder()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	handleInternalUserAccountDeletionBlockers(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", w.Code)
	}
}

func TestInternalUserAccountDeletionBlockersBadPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/users/123/other/", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", "test-secret")
	w := httptest.NewRecorder()
	prev := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	handleInternalUserAccountDeletionBlockers(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", w.Code)
	}
}

func insertDeletionPaymentPending(t *testing.T, outTradeNo, userID string, tenantID, orderID int64, payStatus string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO billing_payment_pending (
			out_trade_no, user_id, tenant_id, order_id, amount_fen, status, created_at
		) VALUES (?, ?, ?, ?, 100, ?, ?)`,
		outTradeNo, userID, tenantID, orderID, payStatus, utcNow(),
	)
	if err != nil {
		t.Fatalf("insert billing_payment_pending: %v", err)
	}
}

func insertDeletionOrder(t *testing.T, orderID, tenantID int64, status string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO billing_resource_order (
			id, tenant_id, order_number, status, total_yuan_cents,
			payment_method, payment_ref, created_at
		) VALUES (?, ?, ?, ?, 100, 'wechat', '', ?)`,
		orderID, tenantID, fmt.Sprintf("ORD-DEL-%d", orderID), status, utcNow(),
	)
	if err != nil {
		t.Fatalf("insert billing_resource_order: %v", err)
	}
}

func TestListPendingPaymentsForUserSkipsTerminalOrders(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := generateSnowflakeID()

	cases := []struct {
		name      string
		payStatus string
		orderID   int64
		orderStat string
		wantN     int
	}{
		{
			name:      "pending pay linked to refunded order must not block",
			payStatus: "pending",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusRefunded,
			wantN:     0,
		},
		{
			name:      "pending pay linked to paid order must not block",
			payStatus: "pending",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusPaid,
			wantN:     0,
		},
		{
			name:      "pending pay linked to cancelled order must not block",
			payStatus: "pending",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusCancelled,
			wantN:     0,
		},
		{
			name:      "pending pay linked to expired order must not block",
			payStatus: "pending",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusExpired,
			wantN:     0,
		},
		{
			name:      "paid payment row must not block",
			payStatus: "paid",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusPending,
			wantN:     0,
		},
		{
			name:      "refunded payment row must not block",
			payStatus: "refunded",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusPending,
			wantN:     0,
		},
		{
			name:      "pending pay with pending order is a real blocker",
			payStatus: "pending",
			orderID:   generateSnowflakeID(),
			orderStat: OrderStatusPending,
			wantN:     1,
		},
		{
			name:      "pending pay without order (recharge) is a real blocker",
			payStatus: "pending",
			orderID:   0,
			wantN:     1,
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID := strconv.FormatInt(generateSnowflakeID(), 10)
			outTradeNo := fmt.Sprintf("WX-DEL-%d", i)
			if tc.orderID > 0 {
				insertDeletionOrder(t, tc.orderID, tenantID, tc.orderStat)
			}
			insertDeletionPaymentPending(t, outTradeNo, userID, tenantID, tc.orderID, tc.payStatus)

			got, err := listPendingPaymentsForUser(t.Context(), userID)
			if err != nil {
				t.Fatalf("listPendingPaymentsForUser: %v", err)
			}
			if len(got) != tc.wantN {
				t.Fatalf("got %d pending payments %#v, want %d", len(got), got, tc.wantN)
			}
			if tc.wantN == 1 && got[0].OutTradeNo != outTradeNo {
				t.Fatalf("out_trade_no=%s want %s", got[0].OutTradeNo, outTradeNo)
			}
		})
	}
}

func TestListPendingPaymentsForUserMatchesReportedRefundedOrder(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	const (
		userID   = "877397583960502272"
		tenantID = int64(877397588196749312)
		orderID  = int64(877596007691485184)
		outTrade = "WX877596018835750912"
	)
	insertDeletionOrder(t, orderID, tenantID, OrderStatusRefunded)
	insertDeletionPaymentPending(t, outTrade, userID, tenantID, orderID, "pending")

	got, err := listPendingPaymentsForUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("listPendingPaymentsForUser: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("refunded order still listed as pending payment: %#v", got)
	}
}
