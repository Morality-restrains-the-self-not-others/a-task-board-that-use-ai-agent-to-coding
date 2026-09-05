package main

import "testing"

func TestReferralAccrualPointsFromChargeResult(t *testing.T) {
	if got := referralAccrualPointsFromChargeResult(nil); got != 0 {
		t.Fatalf("nil → %d", got)
	}
	if got := referralAccrualPointsFromChargeResult(map[string]interface{}{"cost": int64(0)}); got != 0 {
		t.Fatalf("zero → %d", got)
	}
	if got := referralAccrualPointsFromChargeResult(map[string]interface{}{"cost": int64(10000)}); got != 10000 {
		t.Fatalf("int64 → %d want 10000 (charge.go used to hardcode 0, skipping accrual)", got)
	}
	if got := referralAccrualPointsFromChargeResult(map[string]interface{}{"cost": 250}); got != 250 {
		t.Fatalf("int → %d", got)
	}
}
