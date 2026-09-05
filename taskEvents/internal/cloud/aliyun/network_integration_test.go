//go:build integration

package aliyun

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestProvisionAutoNetworkResourcesIntegration(t *testing.T) {
	ak := strings.TrimSpace(os.Getenv("ALIYUN_ACCESS_KEY_ID"))
	sk := strings.TrimSpace(os.Getenv("ALIYUN_ACCESS_KEY_SECRET"))
	region := strings.TrimSpace(os.Getenv("ALIYUN_TEST_REGION"))
	zone := strings.TrimSpace(os.Getenv("ALIYUN_TEST_ZONE"))
	if ak == "" || sk == "" || region == "" || zone == "" {
		t.Skip("set ALIYUN_ACCESS_KEY_ID, ALIYUN_ACCESS_KEY_SECRET, ALIYUN_TEST_REGION, ALIYUN_TEST_ZONE")
	}

	ctx := context.Background()
	first, err := ProvisionAutoNetworkResources(ctx, ak, sk, AutoNetworkInput{
		RegionID: region, ZoneID: zone,
		AutoVPC: true, AutoVSwitch: true, AutoSG: true,
	})
	if err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if first.VPCID == "" || first.VSwitchID == "" || first.SecurityGroupID == "" {
		t.Fatalf("first result incomplete: %+v", first)
	}

	second, err := ProvisionAutoNetworkResources(ctx, ak, sk, AutoNetworkInput{
		RegionID: region, ZoneID: zone,
		AutoVPC: true, AutoVSwitch: true, AutoSG: true,
	})
	if err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if second.VPCID != first.VPCID {
		t.Fatalf("vpc not idempotent: %s vs %s", first.VPCID, second.VPCID)
	}
	if second.VSwitchID != first.VSwitchID {
		t.Fatalf("vswitch not idempotent: %s vs %s", first.VSwitchID, second.VSwitchID)
	}
	if second.SecurityGroupID != first.SecurityGroupID {
		t.Fatalf("sg not idempotent: %s vs %s", first.SecurityGroupID, second.SecurityGroupID)
	}
}
