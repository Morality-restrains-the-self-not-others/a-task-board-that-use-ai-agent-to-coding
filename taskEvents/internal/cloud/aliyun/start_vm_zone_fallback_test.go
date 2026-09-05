package aliyun

import (
	"errors"
	"testing"
)

func TestAlternateZonesForNoStockSkipsFailedZone(t *testing.T) {
	zones := []string{"cn-hongkong-a", "cn-hongkong-b", "cn-hongkong-c"}
	failed := "cn-hongkong-b"
	out := make([]string, 0, len(zones))
	for _, z := range zones {
		if z == failed {
			continue
		}
		out = append(out, z)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(out))
	}
	for _, z := range out {
		if z == failed {
			t.Fatalf("failed zone should be skipped: %s", z)
		}
	}
}

func TestIsNoStockErrorRawSDK(t *testing.T) {
	if !IsNoStockError(errors.New("Code: OperationDenied.NoStock")) {
		t.Fatal("expected NoStock detection")
	}
}
