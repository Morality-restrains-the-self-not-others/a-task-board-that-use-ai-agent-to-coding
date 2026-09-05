package main

import "testing"

func TestRemainingCapacity(t *testing.T) {
	if got := remainingCapacity(100, 30); got != 70 {
		t.Fatalf("remainingCapacity(100,30)=%d", got)
	}
	if got := remainingCapacity(10, 50); got != 0 {
		t.Fatalf("remainingCapacity underflow=%d", got)
	}
}

func TestClampRemainingBandwidth(t *testing.T) {
	if got := clampRemainingBandwidth(200, 80); got != 80 {
		t.Fatalf("got %d", got)
	}
	if got := clampRemainingBandwidth(200, 999); got != 200 {
		t.Fatalf("over total got %d", got)
	}
	if got := clampRemainingBandwidth(200, -5); got != 0 {
		t.Fatalf("negative got %d", got)
	}
}

func TestAttachDeployInfo_IncludesBandwidthShareAndRemaining(t *testing.T) {
	regions := []GitlabRegion{{
		ID:                     1,
		Name:                   "腾讯上海一区",
		Slug:                   "tencent-sh-1",
		IsActive:               true,
		BandwidthShared:        true,
		TotalBandwidthMbps:     200,
		RemainingBandwidthMbps: 999,
		TotalDiskGB:            100,
		AllocatedDiskGB:        10,
		TotalTrafficGB:         50,
		AllocatedTrafficGB:     5,
	}}
	out := attachDeployInfo(regions)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	row := out[0]
	shared, _ := row["bandwidth_shared"].(bool)
	if !shared {
		t.Fatalf("bandwidth_shared=%v", row["bandwidth_shared"])
	}
	if row["total_bandwidth_mbps"] != int64(200) {
		t.Fatalf("total=%v", row["total_bandwidth_mbps"])
	}
	if row["remaining_bandwidth_mbps"] != int64(200) {
		t.Fatalf("remaining should clamp to total, got %v", row["remaining_bandwidth_mbps"])
	}
}

func TestAttachDeployInfo_DedicatedBandwidthUnconfigured(t *testing.T) {
	out := attachDeployInfo([]GitlabRegion{{
		Slug:                   "tencent-shanghai-5",
		BandwidthShared:        false,
		TotalBandwidthMbps:     0,
		RemainingBandwidthMbps: 0,
	}})
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0]["bandwidth_shared"] != false {
		t.Fatalf("shared=%v", out[0]["bandwidth_shared"])
	}
	if out[0]["total_bandwidth_mbps"] != int64(0) {
		t.Fatalf("total=%v", out[0]["total_bandwidth_mbps"])
	}
	if out[0]["remaining_bandwidth_mbps"] != int64(0) {
		t.Fatalf("remaining=%v", out[0]["remaining_bandwidth_mbps"])
	}
}
