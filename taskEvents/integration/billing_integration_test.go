//go:build integration

package integration_test

import (
	"testing"

	"taskEvents/domain"
	"taskEvents/integration"
	"taskEvents/internal/handlers/billing"
)

func TestBillingTransactionCreatedRedisRoundTrip(t *testing.T) {
	h := &billing.Handler{}
	out := integration.DispatchOnce(
		t,
		"BILLING_TRANSACTION_CREATED",
		map[string]interface{}{"transaction_id": "txn-integration-001", "user_id": 1},
		"txn-integration-001",
		"billing_transaction_created",
		h,
		nil,
	)
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestBillingTransactionCreatedIdempotent(t *testing.T) {
	h := &billing.Handler{}
	data := map[string]interface{}{"transaction_id": "txn-integration-idem", "user_id": 1}
	out1 := integration.DispatchOnce(t, "BILLING_TRANSACTION_CREATED", data, "txn-integration-idem", "billing_transaction_created", h, nil)
	out2 := integration.DispatchOnce(t, "BILLING_TRANSACTION_CREATED", data, "txn-integration-idem", "billing_transaction_created", h, nil)
	if out1 != domain.DispatchSuccess || out2 != domain.DispatchSuccess {
		t.Fatalf("outcomes %v %v", out1, out2)
	}
}
