package aliyun

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

const (
	defaultSpotStrategy                         = "SpotAsPriceGo"
	defaultSpotDuration                   int32 = 0
	defaultSystemDiskCategory                   = "cloud_essd"
	defaultInstanceChargeType                   = "PostPaid"
	runInstancesFixedHostName                   = "task2app"
	maxIdempotentParameterMismatchRetries       = 5
	maxIdempotentProcessingRetries              = 45
	idempotentProcessingWaitSec                 = 2
)

// StartVMInput is the RunInstances payload for CLOUD_SERVER_STARTED.
type StartVMInput struct {
	RegionID        string
	ZoneID          string
	VPCID           string
	ImageID         string
	InstanceType    string
	SecurityGroupID string
	VSwitchID       string
	TaskID          string
	CommentID       string
	UserData        string
	HardwareConfig  map[string]interface{}
}

// StartVMResult is returned after RunInstances.
type StartVMResult struct {
	InstanceID      string
	RequestID       string
	ClientToken     string
	LaunchRequestID string
	ZoneID          string
	VSwitchID       string
}

// StartVM runs ECS RunInstances with hardware parity to Django start_aliyun_vm.
func StartVM(ctx context.Context, accessKey, secretKey string, in StartVMInput) (StartVMResult, error) {
	if in.RegionID == "" {
		return StartVMResult{}, fmt.Errorf("region_id required")
	}
	if in.ImageID == "" || in.SecurityGroupID == "" || in.VSwitchID == "" {
		return StartVMResult{}, fmt.Errorf("image_id, security_group_id, vswitch_id required")
	}

	hw := in.HardwareConfig
	if hw == nil {
		hw = map[string]interface{}{}
	}
	tokenSnapshot := hardwareSnapshotForClientToken(in, hw)
	tokenIdentity := ECSInstanceName(in.TaskID, in.CommentID)
	if tokenIdentity == "" {
		return StartVMResult{}, fmt.Errorf("comment_id required")
	}
	clientToken := ComputeClientTokenFromTaskIDAndHardware(tokenIdentity, tokenSnapshot)

	client, err := newECSClient(accessKey, secretKey, in.RegionID)
	if err != nil {
		return StartVMResult{}, err
	}

	zoneID := strings.TrimSpace(in.ZoneID)
	if zoneID == "" {
		zoneID = strFromHW(hw, "zone_id")
	}
	vswitchID := strings.TrimSpace(in.VSwitchID)
	instanceType := strings.TrimSpace(in.InstanceType)
	if instanceType == "" {
		instanceType = strFromHW(hw, "instance_type")
	}

	resp, usedZone, usedVSwitch, usedToken, err := runInstancesOnce(ctx, client, in, hw, zoneID, vswitchID, clientToken)
	if err == nil {
		return buildStartVMResult(resp, usedZone, usedVSwitch, usedToken)
	}
	// NoStock: fail fast on the user-selected zone only — do not silently retry other zones
	// (VSwitch/SG are zone-bound; cross-zone fallback can leave resources in the wrong AZ).
	if IsNoStockError(err) {
		return StartVMResult{}, &NoStockError{ZoneID: zoneID, InstanceType: instanceType}
	}
	return StartVMResult{}, FormatStartVMError(err, zoneID)
}

func runInstancesOnce(
	ctx context.Context,
	client *ecs.Client,
	in StartVMInput,
	hw map[string]interface{},
	zoneID, vswitchID, clientToken string,
) (*ecs.RunInstancesResponse, string, string, string, error) {
	attemptIn := in
	attemptIn.ZoneID = zoneID
	attemptIn.VSwitchID = vswitchID

	var (
		resp                     *ecs.RunInstancesResponse
		lastRequestParams        map[string]interface{}
		idempotentAttempt        int
		idempotentProcessingWait int
		currentClientToken       = clientToken
		err                      error
	)

	for {
		req, requestParams := buildRunInstancesRequest(attemptIn, hw, currentClientToken)
		lastRequestParams = requestParams
		resp, err = client.RunInstances(req)
		if err == nil {
			return resp, zoneID, vswitchID, currentClientToken, nil
		}
		if IsIdempotentProcessingAliyunError(err) && idempotentProcessingWait < maxIdempotentProcessingRetries {
			idempotentProcessingWait++
			select {
			case <-ctx.Done():
				return nil, zoneID, vswitchID, currentClientToken, ctx.Err()
			case <-time.After(idempotentProcessingWaitSec * time.Second):
			}
			continue
		}
		if IsIdempotentParameterMismatchAliyunError(err) &&
			idempotentAttempt < maxIdempotentParameterMismatchRetries &&
			lastRequestParams != nil {
			idempotentAttempt++
			idempotentProcessingWait = 0
			currentClientToken = ComputeRunInstancesClientTokenAfterMismatch(
				currentClientToken,
				lastRequestParams,
				idempotentAttempt,
			)
			continue
		}
		return nil, zoneID, vswitchID, currentClientToken, err
	}
}

func buildStartVMResult(resp *ecs.RunInstancesResponse, zoneID, vswitchID, clientToken string) (StartVMResult, error) {
	out := StartVMResult{
		ClientToken: clientToken,
		ZoneID:      zoneID,
		VSwitchID:   vswitchID,
	}
	if resp != nil && resp.Body != nil {
		if resp.Body.RequestId != nil {
			out.RequestID = *resp.Body.RequestId
			out.LaunchRequestID = *resp.Body.RequestId
		}
		if resp.Body.InstanceIdSets != nil && len(resp.Body.InstanceIdSets.InstanceIdSet) > 0 {
			out.InstanceID = *resp.Body.InstanceIdSets.InstanceIdSet[0]
		}
	}
	if out.InstanceID == "" {
		return out, fmt.Errorf("RunInstances returned no instance id")
	}
	return out, nil
}

func buildRunInstancesRequest(in StartVMInput, hw map[string]interface{}, clientToken string) (*ecs.RunInstancesRequest, map[string]interface{}) {
	instanceType := strings.TrimSpace(in.InstanceType)
	if instanceType == "" {
		instanceType = strFromHW(hw, "instance_type")
	}
	instanceChargeType := strFromHW(hw, "instance_charge_type")
	if instanceChargeType == "" {
		instanceChargeType = defaultInstanceChargeType
	}
	spotStrategy := strFromHW(hw, "spot_strategy")
	if spotStrategy == "" {
		spotStrategy = defaultSpotStrategy
	}
	spotDuration := int32FromHW(hw, "spot_duration", defaultSpotDuration)
	systemDiskCategory := strFromHW(hw, "system_disk_category")
	if systemDiskCategory == "" {
		systemDiskCategory = defaultSystemDiskCategory
	}
	systemDiskSize := int32FromHW(hw, "storage_gb", 40)
	bandwidth := int32FromHW(hw, "internet_max_bandwidth_out", 5)
	if bandwidth <= 0 {
		bandwidth = 5
	}

	zoneID := strings.TrimSpace(in.ZoneID)
	if zoneID == "" {
		zoneID = strFromHW(hw, "zone_id")
	}
	useZone := zoneID != ""
	currentZoneID := ""
	var currentVSwitchID *string
	if useZone {
		currentZoneID = zoneID
		currentVSwitchID = strPtr(in.VSwitchID)
	}

	autoReleaseTime := ComputeAutoReleaseTimeISO(hw)
	if autoReleaseTime != "" && instanceChargeType != "PostPaid" {
		autoReleaseTime = ""
	}

	instanceName := ECSInstanceName(in.TaskID, in.CommentID)
	req := &ecs.RunInstancesRequest{
		RegionId:                strPtr(in.RegionID),
		ImageId:                 strPtr(in.ImageID),
		SecurityGroupIds:        []*string{strPtr(in.SecurityGroupID)},
		InstanceType:            strPtr(defaultInstanceType(instanceType)),
		Amount:                  int32Ptr(1),
		HostName:                strPtr(runInstancesFixedHostName),
		InternetMaxBandwidthOut: int32Ptr(bandwidth),
		InstanceChargeType:      strPtr(instanceChargeType),
		SpotStrategy:            strPtr(spotStrategy),
		SpotDuration:            int32Ptr(spotDuration),
		ClientToken:             strPtr(clientToken),
		SystemDisk: &ecs.RunInstancesRequestSystemDisk{
			Category: strPtr(systemDiskCategory),
			Size:     strPtr(fmt.Sprint(systemDiskSize)),
		},
	}
	if instanceName != "" {
		req.InstanceName = strPtr(instanceName)
	}
	if currentVSwitchID != nil {
		req.VSwitchId = currentVSwitchID
	}
	if useZone {
		req.ZoneId = strPtr(currentZoneID)
	}
	if in.UserData != "" {
		req.UserData = strPtr(encodeUserDataBase64(in.UserData))
	}
	if autoReleaseTime != "" {
		req.AutoReleaseTime = strPtr(autoReleaseTime)
	}

	requestParams := map[string]interface{}{
		"InstanceName":            instanceName,
		"ClientToken":             clientToken,
		"InstanceType":            defaultInstanceType(instanceType),
		"ImageId":                 in.ImageID,
		"InstanceChargeType":      instanceChargeType,
		"InternetMaxBandwidthOut": bandwidth,
		"SecurityGroupId":         in.SecurityGroupID,
		"SystemDisk.Category":     systemDiskCategory,
		"SystemDisk.Size":         systemDiskSize,
		"RegionId":                in.RegionID,
		"Amount":                  1,
		"VSwitchId":               derefStr(currentVSwitchID),
		"HostName":                runInstancesFixedHostName,
		"SpotStrategy":            spotStrategy,
		"SpotDuration":            spotDuration,
	}
	if currentZoneID != "" {
		requestParams["ZoneId"] = currentZoneID
	}
	if autoReleaseTime != "" {
		requestParams["AutoReleaseTime"] = autoReleaseTime
	}
	return req, requestParams
}

func hardwareSnapshotForClientToken(in StartVMInput, hw map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"cpu_cores":                  intFromHW(hw, "cpu_cores", 1),
		"memory_gb":                  intFromHW(hw, "memory_gb", 1),
		"storage_gb":                 intFromHW(hw, "storage_gb", 40),
		"region_id":                  firstNonEmpty(in.RegionID, strFromHW(hw, "region_id")),
		"zone_id":                    firstNonEmpty(in.ZoneID, strFromHW(hw, "zone_id")),
		"instance_type":              firstNonEmpty(in.InstanceType, strFromHW(hw, "instance_type")),
		"system_disk_category":       strFromHW(hw, "system_disk_category"),
		"internet_max_bandwidth_out": intFromHW(hw, "internet_max_bandwidth_out", 5),
		"instance_charge_type":       strFromHW(hw, "instance_charge_type"),
		"spot_strategy":              strFromHW(hw, "spot_strategy"),
		"spot_duration":              intFromHW(hw, "spot_duration", 0),
		"auto_release_minutes":       hw["auto_release_minutes"],
		"auto_release_hours":         hw["auto_release_hours"],
	}
}

func strFromHW(hw map[string]interface{}, key string) string {
	if hw == nil {
		return ""
	}
	v, ok := hw[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func intFromHW(hw map[string]interface{}, key string, def int) int {
	if hw == nil {
		return def
	}
	v, ok := hw[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return def
		}
		var i int
		if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
			return i
		}
		return def
	}
}

func int32FromHW(hw map[string]interface{}, key string, def int32) int32 {
	return int32(intFromHW(hw, key, int(def)))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func defaultInstanceType(t string) string {
	if t != "" {
		return t
	}
	return "ecs.t5-lc1m1.small"
}

func int32Ptr(n int32) *int32 { return &n }

// encodeUserDataBase64 matches Django _aliyun_user_data_b64: UTF-8 plain text → standard Base64 (no newlines).
func encodeUserDataBase64(plain string) string {
	return base64.StdEncoding.EncodeToString([]byte(plain))
}
