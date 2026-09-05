package main

import (
	"context"
	"fmt"
	"log/slog"
)

const zeroAmountInvoiceRejectNote = "零额不可开票"

// pendingZeroAmountInvoiceAppIDs 盘点 status=pending 且订单金额 ≤0 的开票申请（只读）。
func pendingZeroAmountInvoiceAppIDs(ctx context.Context) ([]int64, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT a.id
		FROM billing_invoice_application a
		INNER JOIN billing_resource_order o ON o.id = a.order_id
		WHERE a.status = ? AND o.total_yuan_cents <= 0
		ORDER BY a.id`,
		invoiceAppPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// rejectPendingZeroAmountInvoiceApplications 用既有 reject 路径拒绝存量零额 pending 申请。
func rejectPendingZeroAmountInvoiceApplications(ctx context.Context, reviewerUserID string) (int, error) {
	ids, err := pendingZeroAmountInvoiceAppIDs(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, id := range ids {
		if _, err := rejectInvoiceApplication(ctx, id, reviewerUserID, zeroAmountInvoiceRejectNote); err != nil {
			return n, fmt.Errorf("reject application %d: %w", id, err)
		}
		n++
		slog.InfoContext(ctx, "invoice_application_rejected_zero_amount_pending",
			"level", "info",
			"application_id", formatID(id),
			"reviewer_user_id", reviewerUserID,
		)
	}
	return n, nil
}
