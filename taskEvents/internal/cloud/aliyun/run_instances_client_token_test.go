package aliyun

import (
	"errors"
	"strings"
	"testing"
)

func TestIsIdempotentParameterMismatchPositive(t *testing.T) {
	err := errors.New("Error: IdempotentParameterMismatch code: 403, RequestId: abc")
	if !IsIdempotentParameterMismatchAliyunError(err) {
		t.Fatal("expected true")
	}
}

func TestIsIdempotentParameterMismatchNegativeWithout403(t *testing.T) {
	err := errors.New("Error: IdempotentParameterMismatch code: 400")
	if IsIdempotentParameterMismatchAliyunError(err) {
		t.Fatal("expected false")
	}
}

func TestIsIdempotentProcessingPositive(t *testing.T) {
	err := errors.New("Error: IdempotentProcessing code: 403, still processing")
	if !IsIdempotentProcessingAliyunError(err) {
		t.Fatal("expected true")
	}
}

func TestComputeClientTokenStable(t *testing.T) {
	snap := map[string]interface{}{
		"cpu_cores": 2, "memory_gb": 4, "storage_gb": 40,
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-a",
		"instance_type": nil, "system_disk_category": "cloud_essd",
		"internet_max_bandwidth_out": 10, "instance_charge_type": "PostPaid",
		"auto_release_minutes": nil, "auto_release_hours": nil,
	}
	a := ComputeClientTokenFromTaskIDAndHardware("task-hw", snap)
	b := ComputeClientTokenFromTaskIDAndHardware("task-hw", snap)
	if a != b {
		t.Fatalf("token unstable: %q vs %q", a, b)
	}
	if len(a) > 64 {
		t.Fatalf("token too long: %d", len(a))
	}
}

func TestComputeClientTokenDiffersWhenCPUChanges(t *testing.T) {
	base := map[string]interface{}{
		"cpu_cores": 2, "memory_gb": 4, "storage_gb": 40,
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-a",
		"instance_type": nil, "system_disk_category": "cloud_essd",
		"internet_max_bandwidth_out": 10, "instance_charge_type": "PostPaid",
		"auto_release_minutes": nil, "auto_release_hours": nil,
	}
	a := ComputeClientTokenFromTaskIDAndHardware("t", mergeSnap(base, "cpu_cores", 2))
	b := ComputeClientTokenFromTaskIDAndHardware("t", mergeSnap(base, "cpu_cores", 4))
	if a == b {
		t.Fatal("expected different tokens when cpu changes")
	}
}

func TestComputeRunInstancesClientTokenAfterMismatchStable(t *testing.T) {
	params := map[string]interface{}{"ImageId": "img-1", "RegionId": "cn-hangzhou"}
	a := ComputeRunInstancesClientTokenAfterMismatch("base", params, 1)
	b := ComputeRunInstancesClientTokenAfterMismatch("base", params, 1)
	if a != b {
		t.Fatalf("unstable mismatch token: %q vs %q", a, b)
	}
	if len(a) > 64 {
		t.Fatalf("token too long")
	}
}

func TestComputeRunInstancesClientTokenAfterMismatchDiffersByParams(t *testing.T) {
	a := ComputeRunInstancesClientTokenAfterMismatch("base", map[string]interface{}{"ImageId": "a"}, 1)
	b := ComputeRunInstancesClientTokenAfterMismatch("base", map[string]interface{}{"ImageId": "b"}, 1)
	if a == b {
		t.Fatal("expected different tokens")
	}
}

func TestComputeAutoReleaseTimeISO(t *testing.T) {
	got := ComputeAutoReleaseTimeISO(map[string]interface{}{"auto_release_minutes": 30})
	if got == "" {
		t.Fatal("expected non-empty auto release time")
	}
	if !strings.HasSuffix(got, "Z") {
		t.Fatalf("expected UTC Z suffix: %q", got)
	}
}

func mergeSnap(base map[string]interface{}, key string, val interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base))
	for k, v := range base {
		out[k] = v
	}
	out[key] = val
	return out
}
