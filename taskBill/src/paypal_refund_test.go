package main

import "testing"

func TestExtractPaypalCaptureID(t *testing.T) {
	data := map[string]interface{}{
		"purchase_units": []interface{}{
			map[string]interface{}{
				"payments": map[string]interface{}{
					"captures": []interface{}{
						map[string]interface{}{
							"id":     "CAP-XYZ",
							"status": "COMPLETED",
						},
					},
				},
			},
		},
	}
	if got := extractPaypalCaptureID(data); got != "CAP-XYZ" {
		t.Fatalf("got %q", got)
	}
}

func TestExecuteProviderRefundMockEnv(t *testing.T) {
	t.Setenv("TASKBILL_REFUND_PROVIDER", "mock")
	ref, err := executeProviderRefund(t.Context(), providerRefundRequest{
		Channel:        "paypal",
		ProviderRef:    "order-1",
		CaptureID:      "cap-1",
		Currency:       "USD",
		RefundPoints:   100,
		OriginalPoints: 100,
		AmountMinor:    100,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ref == "" || ref[:5] != "mock:" {
		t.Fatalf("ref=%q", ref)
	}
}
