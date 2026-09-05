package aliyun

import (
	"fmt"
	"sort"
	"strings"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

const maxNoStockZoneFallbackAttempts = 8

// resolveVPCID returns explicit VPC or looks up VPC from an existing VSwitch.
func resolveVPCID(client *ecs.Client, regionID, vpcID, vswitchID string) (string, error) {
	if v := strings.TrimSpace(vpcID); v != "" {
		return v, nil
	}
	vswitchID = strings.TrimSpace(vswitchID)
	if vswitchID == "" {
		return "", fmt.Errorf("vpc_id required for zone fallback")
	}
	req := &ecs.DescribeVSwitchesRequest{
		RegionId:  strPtr(regionID),
		VSwitchId: strPtr(vswitchID),
	}
	resp, err := client.DescribeVSwitches(req)
	if err != nil {
		return "", err
	}
	if resp.Body == nil || resp.Body.VSwitches == nil || len(resp.Body.VSwitches.VSwitch) == 0 {
		return "", fmt.Errorf("vswitch %s not found", vswitchID)
	}
	item := resp.Body.VSwitches.VSwitch[0]
	if item == nil || item.VpcId == nil || strings.TrimSpace(*item.VpcId) == "" {
		return "", fmt.Errorf("vswitch %s has no vpc", vswitchID)
	}
	return strings.TrimSpace(*item.VpcId), nil
}

// listZonesWithInstanceStock returns zone IDs where the instance type is Available.
func listZonesWithInstanceStock(client *ecs.Client, regionID, instanceType string, hw map[string]interface{}) ([]string, error) {
	instanceType = strings.TrimSpace(instanceType)
	if instanceType == "" {
		return nil, fmt.Errorf("instance_type required")
	}
	req := &ecs.DescribeAvailableResourceRequest{
		RegionId:            strPtr(regionID),
		DestinationResource: strPtr("InstanceType"),
		InstanceType:        strPtr(instanceType),
	}
	applyDescribeAvailableResourceFilters(req, hw)
	resp, err := client.DescribeAvailableResource(req)
	if err != nil {
		return nil, err
	}
	zones := []string{}
	seen := map[string]struct{}{}
	if resp.Body == nil || resp.Body.AvailableZones == nil {
		return zones, nil
	}
	for _, az := range resp.Body.AvailableZones.AvailableZone {
		if az == nil || az.ZoneId == nil {
			continue
		}
		zoneID := strings.TrimSpace(*az.ZoneId)
		if zoneID == "" {
			continue
		}
		if !zoneHasAvailableInstance(az) {
			continue
		}
		if _, ok := seen[zoneID]; ok {
			continue
		}
		seen[zoneID] = struct{}{}
		zones = append(zones, zoneID)
	}
	sort.Strings(zones)
	return zones, nil
}

func zoneHasAvailableInstance(az *ecs.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZone) bool {
	if az == nil || az.AvailableResources == nil {
		return false
	}
	for _, ar := range az.AvailableResources.AvailableResource {
		if ar == nil || ar.SupportedResources == nil {
			continue
		}
		for _, sr := range ar.SupportedResources.SupportedResource {
			if sr == nil || sr.Status == nil {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(*sr.Status), "Available") {
				return true
			}
		}
	}
	return false
}

func applyDescribeAvailableResourceFilters(req *ecs.DescribeAvailableResourceRequest, hw map[string]interface{}) {
	if req == nil {
		return
	}
	if hw == nil {
		hw = map[string]interface{}{}
	}
	instanceChargeType := strFromHW(hw, "instance_charge_type")
	if instanceChargeType == "" {
		instanceChargeType = defaultInstanceChargeType
	}
	req.InstanceChargeType = strPtr(instanceChargeType)
	spotStrategy := strFromHW(hw, "spot_strategy")
	if spotStrategy == "" {
		spotStrategy = defaultSpotStrategy
	}
	req.SpotStrategy = strPtr(spotStrategy)
	spotDuration := int32FromHW(hw, "spot_duration", defaultSpotDuration)
	req.SpotDuration = int32Ptr(spotDuration)
	systemDiskCategory := strFromHW(hw, "system_disk_category")
	if systemDiskCategory == "" {
		systemDiskCategory = defaultSystemDiskCategory
	}
	req.SystemDiskCategory = strPtr(systemDiskCategory)
	if cores := intFromHW(hw, "cpu_cores", 0); cores > 0 {
		req.Cores = int32Ptr(int32(cores))
	}
	if mem := intFromHW(hw, "memory_gb", 0); mem > 0 {
		memF := float32(mem)
		req.Memory = &memF
	}
	req.IoOptimized = strPtr("optimized")
	req.NetworkCategory = strPtr("vpc")
}

func alternateZonesForNoStock(client *ecs.Client, regionID, failedZone, instanceType string, hw map[string]interface{}) ([]string, error) {
	zones, err := listZonesWithInstanceStock(client, regionID, instanceType, hw)
	if err != nil {
		return nil, err
	}
	failedZone = strings.TrimSpace(failedZone)
	out := make([]string, 0, len(zones))
	for _, z := range zones {
		if z == failedZone {
			continue
		}
		out = append(out, z)
		if len(out) >= maxNoStockZoneFallbackAttempts {
			break
		}
	}
	return out, nil
}

func resolveVSwitchForZone(client *ecs.Client, regionID, zoneID, vpcID string) (string, error) {
	return getOrCreateVSwitch(client, regionID, zoneID, vpcID, defaultVSwitchName, defaultVSwitchCIDR)
}
