package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// ── RunInstances ────────────────────────────────────────────────────────────

func aliyunRunInstances(accessKey, secretKey, regionID string, req *ecsclient.RunInstancesRequest) (*ecsclient.RunInstancesResponseBody, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.RunInstances(req)
	if err != nil {
		return nil, "", err
	}
	rid := ""
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
	}
	return resp.Body, rid, nil
}

// userdataDigest 返回 UserData 内容的紧凑摘要（空串 → 空摘要）。
// 摘要随占位符替换结果变化，驱动 ClientToken 变化以避开阿里云幂等窗口冲突。
func userdataDigest(content string) string {
	if content == "" {
		return ""
	}
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)[:16]
}

// buildClientToken 生成确定性幂等 token。
// 使用 comment_id + hardware snapshot 确保 per-comment 容器启动的唯一性；
// 若 comment_id 为空则回退到 task_id。
// userdataDigest 为最终提交的 UserData 摘要：镜像市场模板经占位符替换后内容变化时，
// token 必须随之变化，否则阿里云幂等窗口内同 token 不同参数返回
// IdempotentParameterMismatch（403），导致修复后的重试被拒（OPT-20260809-024）。
// 格式：SHA256 前 24 位 + id 前缀，最长 64 字符。
func buildClientToken(taskID string, eventData map[string]interface{}, userdataDigest string) string {
	// 优先使用 comment_id（per-container 唯一），回退 task_id
	identityKey := strField(eventData, "comment_id")
	if identityKey == "" {
		identityKey = taskID
	}
	hw := mapValue(eventData, "hardware_config")
	if hw == nil {
		hw = map[string]interface{}{}
	}
	// 与 Django _HARDWARE_CLIENT_TOKEN_KEYS 对齐
	snapshot := map[string]interface{}{
		"id":              identityKey,
		"userdata_digest": userdataDigest,
		"hardware": map[string]interface{}{
			"cpu_cores":                  intFromAny(hw["cpu_cores"], 0),
			"memory_gb":                  intFromAny(hw["memory_gb"], 0),
			"storage_gb":                 intFromAny(hw["storage_gb"], 0),
			"region_id":                  strField(eventData, "region_id"),
			"zone_id":                    strField(eventData, "zone_id"),
			"instance_type":              strField(hw, "instance_type"),
			"system_disk_category":       strField(hw, "system_disk_category"),
			"internet_max_bandwidth_out": intFromAny(hw["internet_max_bandwidth_out"], 0),
			"instance_charge_type":       strField(hw, "instance_charge_type"),
			"spot_strategy":              strField(hw, "spot_strategy"),
			"spot_duration":              intFromAny(hw["spot_duration"], 0),
			"auto_release_minutes":       intFromAny(hw["auto_release_minutes"], 0),
		},
	}
	canonical, _ := json.Marshal(snapshot)
	hash := sha256.Sum256(canonical)
	digest := fmt.Sprintf("%x", hash)[:24]
	prefix := identityKey
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}
	token := prefix + "-" + digest
	if len(token) > 64 {
		token = token[:64]
	}
	return token
}

func buildRunInstancesRequest(eventData map[string]interface{}) *ecsclient.RunInstancesRequest {
	hw := mapValue(eventData, "hardware_config")
	if hw == nil {
		hw = map[string]interface{}{}
	}

	imageID := strField(eventData, "cloud_server_image_id")
	instanceType := strField(hw, "instance_type")
	regionID := strField(eventData, "region_id")
	zoneID := strField(eventData, "zone_id")
	securityGroupID := strField(eventData, "security_group_id")
	vswitchID := strField(eventData, "vswitch_id")
	taskID := strField(eventData, "task_id")
	userdataContent := strField(eventData, "userdata_content")
	// 镜像市场模板含 __TASK2APP_* 运行时占位符，RunInstances 前必须替换，
	// 否则 VM 上 docker pull 拉取字面量占位符失败、cloud-init scripts-user 失败
	//（OPT-20260809-024）。替换规则 SSOT：taskEvents/internal/cloud/userdata/replace.go。
	if userdataContent != "" {
		if replaced, rErr := replaceUserdataRuntimePlaceholders(userdataContent, eventData); rErr != nil {
			logWarn("userdata placeholder replace failed: "+rErr.Error(), taskID)
		} else {
			userdataContent = replaced
		}
	}

	req := &ecsclient.RunInstancesRequest{
		RegionId:        dara.String(regionID),
		ImageId:         dara.String(imageID),
		InstanceType:    dara.String(instanceType),
		SecurityGroupId: dara.String(securityGroupID),
		Amount:          dara.Int32(1),
		MinAmount:       dara.Int32(1),
		ClientToken:     dara.String(buildClientToken(taskID, eventData, userdataDigest(userdataContent))),
	}

	if zoneID != "" {
		req.ZoneId = dara.String(zoneID)
	}
	if vswitchID != "" {
		req.VSwitchId = dara.String(vswitchID)
	}

	commentID := firstNonEmpty(strField(eventData, "parent_comment_id"), strField(eventData, "comment_id"))
	if name := ecsInstanceName(taskID, commentID); name != "" {
		req.InstanceName = dara.String(name)
	}

	// UserData (Base64)
	if userdataContent != "" {
		req.UserData = dara.String(base64.StdEncoding.EncodeToString([]byte(userdataContent)))
	}

	// 硬件配置
	applyHardwareToRunInstances(req, hw)

	// 按量付费
	chargeType := strField(hw, "instance_charge_type")
	if chargeType == "" {
		chargeType = "PostPaid"
	}
	req.InstanceChargeType = dara.String(chargeType)

	// 竞价实例
	spotStrategy := strField(hw, "spot_strategy")
	if spotStrategy != "" {
		req.SpotStrategy = dara.String(spotStrategy)
		if sd := intFromAny(hw["spot_duration"], 0); sd > 0 {
			req.SpotDuration = dara.Int32(int32(sd))
		}
	}

	// 系统盘
	sysDiskCat := strField(hw, "system_disk_category")
	if sysDiskCat == "" {
		sysDiskCat = "cloud_essd"
	}
	storageGB := intFromAny(hw["storage_gb"], 40)
	req.SystemDisk = &ecsclient.RunInstancesRequestSystemDisk{
		Category: dara.String(sysDiskCat),
		Size:     dara.String(fmt.Sprintf("%d", storageGB)),
	}

	// 带宽
	bw := intFromAny(hw["internet_max_bandwidth_out"], 0)
	if bwFromEvent := intFromAny(eventData["bandwidth"], 0); bwFromEvent > 0 {
		bw = bwFromEvent
	}
	if bw > 0 {
		req.InternetMaxBandwidthOut = dara.Int32(int32(bw))
	}

	// 自动释放时间（阿里云格式：UTC ISO8601，秒须为 00）
	if mins := intFromAny(hw["auto_release_minutes"], 0); mins > 0 {
		t := time.Now().UTC().Add(time.Duration(mins) * time.Minute)
		t = t.Truncate(time.Minute)
		req.AutoReleaseTime = dara.String(t.Format("2006-01-02T15:04:05Z"))
	}

	return req
}

func applyHardwareToRunInstances(req *ecsclient.RunInstancesRequest, hw map[string]interface{}) {
	if hw == nil {
		return
	}
	// 数据盘
	if cat := strField(hw, "data_disk_category"); cat != "" {
		size := intFromAny(hw["data_disk_size"], 20)
		if size < 20 {
			size = 20
		}
		req.DataDisk = []*ecsclient.RunInstancesRequestDataDisk{
			{
				Category: dara.String(cat),
				Size:     dara.Int32(int32(size)),
			},
		}
	}
}

// ── 完整 VM 启动（Go 原生）─────────────────────────────────────────────────

func executeStartVmNative(accessKey, secretKey, regionID string, eventData map[string]interface{}) (map[string]interface{}, error) {
	req := buildRunInstancesRequest(eventData)
	body, requestID, err := aliyunRunInstances(accessKey, secretKey, regionID, req)
	if err != nil {
		return nil, err
	}

	instanceID := ""
	if body != nil && body.InstanceIdSets != nil && len(body.InstanceIdSets.InstanceIdSet) > 0 {
		instanceID = derefString(body.InstanceIdSets.InstanceIdSet[0])
	}

	taskID := strField(eventData, "task_id")

	// 立即把 instance_id 落评论 CSC：空 ID 或 persist 失败不得当成功（禁止口头 success SSE）
	if err := bindStartVmInstanceOrError(eventData, instanceID, requestID); err != nil {
		return nil, err
	}

	// 构建 vm_info（与 Django provider.start_vm() 返回值格式一致）
	result := map[string]interface{}{
		"platform":          "aliyun",
		"instance_id":       instanceID,
		"status":            "success",
		"message":           "阿里云服务器创建成功！",
		"region":            regionID,
		"zone_id":           strField(eventData, "zone_id"),
		"security_group_id": strField(eventData, "security_group_id"),
		"vswitch_id":        strField(eventData, "vswitch_id"),
		"launch_request_id": requestID,
		"client_token":      derefString(req.ClientToken),
	}

	// 查询公网 IP（异步：RunInstances 返回时可能尚未分配）
	// 与 Django 一致：最多重试 5 次，间隔 3s
	if instanceID != "" {
		pubIP, serverURL := pollInstancePublicIP(accessKey, secretKey, regionID, instanceID, 5, 3*time.Second)
		if pubIP != "" {
			result["public_ip"] = pubIP
			// 仅 SSE vm_info 不够：server-content / 网关拉容器均读 CSC.public_ip
			persistCloudServerPublicIPByInstanceID(instanceID, pubIP, serverURL)
		} else {
			logWarn(fmt.Sprintf("event=start_vm_public_ip_poll_empty task_id=%s instance_id=%s region=%s",
				taskID, instanceID, regionID), taskID)
		}
		if serverURL != "" {
			result["server_url"] = serverURL
		}
	}

	_ = taskID
	return result, nil
}

func executeStartVmAutoNative(accessKey, secretKey, regionID, zoneID string, eventData map[string]interface{}) (map[string]interface{}, error) {
	taskID := strField(eventData, "task_id")

	// Step 1: 自动创建 VPC/SG/vSwitch
	autoRes, err := autoCreateResources(accessKey, secretKey, regionID, zoneID, taskID, eventData)
	if err != nil {
		return nil, err
	}
	_ = autoRes

	// Step 2: RunInstances
	return executeStartVmNative(accessKey, secretKey, regionID, eventData)
}

// mapValue extracts a nested map from an interface{} map.
func mapValue(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]interface{})
	return v
}

// newECSClientWithRegion is like newECSClient but resolves the custom endpoint config.
func newECSClientWithRegion(accessKey, secretKey, regionID string) (*ecsclient.Client, error) {
	cfg := &openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(regionID),
	}
	applyAliyunECSNetwork(cfg, regionID)
	return ecsclient.NewClient(cfg)
}
