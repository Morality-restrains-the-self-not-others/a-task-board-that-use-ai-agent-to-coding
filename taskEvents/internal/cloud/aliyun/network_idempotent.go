package aliyun

import (
	"fmt"
	"net"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

func findVPCIDByName(client *ecs.Client, region, name string) (string, error) {
	req := &ecs.DescribeVpcsRequest{
		RegionId: strPtr(region),
		PageSize: int32Ptr(50),
	}
	resp, err := client.DescribeVpcs(req)
	if err != nil {
		return "", err
	}
	if resp.Body == nil || resp.Body.Vpcs == nil {
		return "", nil
	}
	for _, item := range resp.Body.Vpcs.Vpc {
		if item == nil || item.VpcName == nil || item.VpcId == nil {
			continue
		}
		if strings.TrimSpace(*item.VpcName) == name {
			return strings.TrimSpace(*item.VpcId), nil
		}
	}
	return "", nil
}

func findVSwitchIDByName(client *ecs.Client, region, zoneID, vpcID, name string) (string, error) {
	items, err := describeVSwitchesInVPCZone(client, region, zoneID, vpcID)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item == nil || item.VSwitchName == nil || item.VSwitchId == nil {
			continue
		}
		if strings.TrimSpace(*item.VSwitchName) == name {
			return strings.TrimSpace(*item.VSwitchId), nil
		}
	}
	return "", nil
}

func describeVSwitchesInVPCZone(client *ecs.Client, region, zoneID, vpcID string) ([]*ecs.DescribeVSwitchesResponseBodyVSwitchesVSwitch, error) {
	items, err := describeVSwitchesInVPC(client, region, vpcID)
	if err != nil {
		return nil, err
	}
	if zoneID == "" {
		return items, nil
	}
	var out []*ecs.DescribeVSwitchesResponseBodyVSwitchesVSwitch
	for _, item := range items {
		if item == nil || item.ZoneId == nil {
			continue
		}
		if strings.TrimSpace(*item.ZoneId) == zoneID {
			out = append(out, item)
		}
	}
	return out, nil
}

func describeVSwitchesInVPC(client *ecs.Client, region, vpcID string) ([]*ecs.DescribeVSwitchesResponseBodyVSwitchesVSwitch, error) {
	req := &ecs.DescribeVSwitchesRequest{
		RegionId: strPtr(region),
		VpcId:    strPtr(vpcID),
		PageSize: int32Ptr(50),
	}
	resp, err := client.DescribeVSwitches(req)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil || resp.Body.VSwitches == nil {
		return nil, nil
	}
	return resp.Body.VSwitches.VSwitch, nil
}

// findFirstVSwitchInVPCZone reuses any existing switch in the target VPC/zone
// (aligned with Django _get_vswitch_id) when name lookup misses.
func findFirstVSwitchInVPCZone(client *ecs.Client, region, zoneID, vpcID string) (string, error) {
	items, err := describeVSwitchesInVPCZone(client, region, zoneID, vpcID)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item == nil || item.VSwitchId == nil {
			continue
		}
		if id := strings.TrimSpace(*item.VSwitchId); id != "" {
			return id, nil
		}
	}
	return "", nil
}

func isCidrOverlappedError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "InvalidCidrBlock.Overlapped")
}

func collectVSwitchCIDRs(items []*ecs.DescribeVSwitchesResponseBodyVSwitchesVSwitch) []string {
	var out []string
	for _, item := range items {
		if item == nil || item.CidrBlock == nil {
			continue
		}
		if cidr := strings.TrimSpace(*item.CidrBlock); cidr != "" {
			out = append(out, cidr)
		}
	}
	return out
}

func nextAvailableVSwitchCIDR(used []string, preferred string) (string, error) {
	candidates := []string{preferred}
	if preferred == defaultVSwitchCIDR {
		for i := 2; i <= 254; i++ {
			candidates = append(candidates, fmt.Sprintf("192.168.%d.0/24", i))
		}
	} else {
		candidates = append(candidates, defaultVSwitchCIDR)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		_, candidateNet, err := net.ParseCIDR(candidate)
		if err != nil {
			continue
		}
		overlaps := false
		for _, u := range used {
			_, usedNet, err := net.ParseCIDR(u)
			if err != nil {
				continue
			}
			if candidateNet.Contains(usedNet.IP) || usedNet.Contains(candidateNet.IP) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available vswitch CIDR in VPC")
}

func findSecurityGroupIDByName(client *ecs.Client, region, vpcID, name string) (string, error) {
	req := &ecs.DescribeSecurityGroupsRequest{
		RegionId:   strPtr(region),
		VpcId:      strPtr(vpcID),
		MaxResults: int32Ptr(50),
	}
	resp, err := client.DescribeSecurityGroups(req)
	if err != nil {
		return "", err
	}
	if resp.Body == nil || resp.Body.SecurityGroups == nil {
		return "", nil
	}
	for _, item := range resp.Body.SecurityGroups.SecurityGroup {
		if item == nil || item.SecurityGroupName == nil || item.SecurityGroupId == nil {
			continue
		}
		if strings.TrimSpace(*item.SecurityGroupName) == name {
			return strings.TrimSpace(*item.SecurityGroupId), nil
		}
	}
	return "", nil
}

func getOrCreateVPC(client *ecs.Client, region, name, cidr string) (string, error) {
	if existing, err := findVPCIDByName(client, region, name); err != nil {
		return "", err
	} else if existing != "" {
		return existing, nil
	}
	return createVPC(client, region, name, cidr)
}

func getOrCreateVSwitch(client *ecs.Client, region, zoneID, vpcID, name, cidr string) (string, error) {
	if existing, err := findVSwitchIDByName(client, region, zoneID, vpcID, name); err != nil {
		return "", err
	} else if existing != "" {
		return existing, nil
	}
	if existing, err := findFirstVSwitchInVPCZone(client, region, zoneID, vpcID); err != nil {
		return "", err
	} else if existing != "" {
		return existing, nil
	}
	allInVPC, err := describeVSwitchesInVPC(client, region, vpcID)
	if err != nil {
		return "", err
	}
	usedCIDRs := collectVSwitchCIDRs(allInVPC)
	createCIDR, err := nextAvailableVSwitchCIDR(usedCIDRs, cidr)
	if err != nil {
		return "", err
	}
	id, err := createVSwitch(client, region, zoneID, vpcID, name, createCIDR)
	if err != nil && isCidrOverlappedError(err) {
		if existing, findErr := findFirstVSwitchInVPCZone(client, region, zoneID, vpcID); findErr == nil && existing != "" {
			return existing, nil
		}
		if altCIDR, pickErr := nextAvailableVSwitchCIDR(append(usedCIDRs, createCIDR), cidr); pickErr == nil && altCIDR != createCIDR {
			return createVSwitch(client, region, zoneID, vpcID, name, altCIDR)
		}
	}
	return id, err
}

func getOrCreateSecurityGroup(client *ecs.Client, region, vpcID, name, clientPublicIP string, extraCIDRs []string) (string, error) {
	existing, err := findSecurityGroupIDByName(client, region, vpcID, name)
	if err != nil {
		return "", err
	}
	if existing == "" && name == defaultSGName {
		// Prefer whitelist-named SG; fall back to legacy full-open / short names and tighten.
		for _, legacy := range []string{legacySGNameFullOpen, legacySGName} {
			existing, err = findSecurityGroupIDByName(client, region, vpcID, legacy)
			if err != nil {
				return "", err
			}
			if existing != "" {
				break
			}
		}
	}
	if existing != "" {
		if err := ensureWhitelistIngress(client, region, existing, clientPublicIP, extraCIDRs); err != nil {
			return "", fmt.Errorf("ensure whitelist ingress on %s: %w", existing, err)
		}
		return existing, nil
	}
	return createSecurityGroupWithRules(client, region, vpcID, clientPublicIP, extraCIDRs)
}

func describeFirstVPCInRegion(client *ecs.Client, region string) (string, error) {
	req := &ecs.DescribeVpcsRequest{
		RegionId: strPtr(region),
		PageSize: int32Ptr(10),
	}
	resp, err := client.DescribeVpcs(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil && resp.Body.Vpcs != nil && len(resp.Body.Vpcs.Vpc) > 0 {
		item := resp.Body.Vpcs.Vpc[0]
		if item != nil && item.VpcId != nil {
			return strings.TrimSpace(*item.VpcId), nil
		}
	}
	return "", fmt.Errorf("no vpc in region %s", region)
}
