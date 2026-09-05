package cloudcommon

import "testing"

func TestShouldSkipCloudAPI(t *testing.T) {
	cases := []struct {
		platform, instance string
		skip               bool
	}{
		{"mock", "mock-e2e-1", true},
		{"mock", "", true},
		{"relay-local", "relay-1", true},
		{"aliyun", "mock-e2e-1", true},
		{"aliyun", "i-abc", false},
		{"mock", "i-abc", false},
		{"", "i-abc", false},
	}
	for _, tc := range cases {
		if got := ShouldSkipCloudAPI(tc.platform, tc.instance); got != tc.skip {
			t.Fatalf("ShouldSkipCloudAPI(%q,%q)=%v want %v", tc.platform, tc.instance, got, tc.skip)
		}
	}
}

func TestNormalizeStopPlatform(t *testing.T) {
	if got := NormalizeStopPlatform("mock", "i-abc"); got != "aliyun" {
		t.Fatalf("stale mock on ECS must remap to aliyun, got %q", got)
	}
	if got := NormalizeStopPlatform("mock", "mock-e2e-1"); got != "mock" {
		t.Fatalf("true mock must stay mock, got %q", got)
	}
	if got := NormalizeStopPlatform("", "i-abc"); got != "aliyun" {
		t.Fatalf("empty platform defaults to aliyun, got %q", got)
	}
}
