package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var hardwarePassthroughKeys = []string{
	"instance_type", "InstanceType", "spot_strategy", "spot_duration",
	"instance_charge_type", "system_disk_category", "internet_max_bandwidth_out",
	"data_disk_category", "network_category", "generation",
}

func normalizeSelectedInstance(v interface{}) string {
	if m, ok := v.(map[string]interface{}); ok {
		return strField(m, "instance_type")
	}
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func parseAutoReleaseMinutes(body map[string]interface{}) (int, string) {
	if !boolField(body, "auto_release_enabled") {
		return 0, ""
	}
	raw := body["auto_release_minutes"]
	if raw == nil && body["auto_release_hours"] != nil {
		if h, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(body["auto_release_hours"]))); err == nil {
			raw = h * 60
		}
	}
	minutes, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(raw)))
	if err != nil {
		return 0, "自动释放时间（分钟）必须为整数"
	}
	if minutes < 30 || minutes > 1440 {
		return 0, "自动释放时间需在 30～1440 分钟之间（阿里云要求不早于当前时间 30 分钟）"
	}
	return minutes, ""
}

func buildHardwareEventPayload(body map[string]interface{}) map[string]interface{} {
	hw, _ := body["hardware_config"].(map[string]interface{})
	if hw == nil {
		hw = map[string]interface{}{}
	}
	cpu := intFromAny(hw["cpu_cores"], 1)
	mem := intFromAny(hw["memory_gb"], 1)
	disk := intFromAny(hw["storage_gb"], 40)
	out := map[string]interface{}{
		"cpu_cores":  cpu,
		"memory_gb":  mem,
		"storage_gb": disk,
	}
	for _, key := range hardwarePassthroughKeys {
		if key == "InstanceType" {
			continue
		}
		if v := hw[key]; v != nil && strings.TrimSpace(fmt.Sprint(v)) != "" {
			out[key] = v
		}
	}
	if rawIT := hw["InstanceType"]; rawIT != nil {
		if it := strings.TrimSpace(fmt.Sprint(rawIT)); it != "" && strField(out, "instance_type") == "" {
			out["instance_type"] = it
		}
	}
	if fo, ok := body["filter_options"].(map[string]interface{}); ok {
		if strField(out, "spot_strategy") == "" && fo["spot_strategy"] != nil {
			out["spot_strategy"] = fo["spot_strategy"]
		}
		if strField(out, "system_disk_category") == "" && fo["system_disk_category"] != nil {
			out["system_disk_category"] = fo["system_disk_category"]
		}
		if strField(out, "data_disk_category") == "" && fo["data_disk_category"] != nil {
			out["data_disk_category"] = fo["data_disk_category"]
		}
	}
	applySelectedInstanceHardware(out, body["selected_instance"])
	if sel := normalizeSelectedInstance(body["selected_instance"]); sel != "" && strField(out, "instance_type") == "" {
		out["instance_type"] = sel
	}
	if bw := body["bandwidth"]; bw != nil && out["internet_max_bandwidth_out"] == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(bw))); err == nil {
			out["internet_max_bandwidth_out"] = n
		}
	}
	if minutes, msg := parseAutoReleaseMinutes(body); msg != "" {
		return map[string]interface{}{"_error": msg}
	} else if minutes > 0 {
		out["auto_release_minutes"] = minutes
	}
	// 实例规格是 CPU/内存权威源（磁盘可独立）；覆盖前端占位/过滤器漂移值
	applyCachedInstanceTypeSpecToHardware(out)
	return out
}

// applySelectedInstanceHardware 在 selected_instance 为对象时同步 cpu/内存/规格。
func applySelectedInstanceHardware(out map[string]interface{}, selected interface{}) {
	m, ok := selected.(map[string]interface{})
	if !ok || out == nil {
		return
	}
	it := strings.TrimSpace(strField(m, "instance_type"))
	if it == "" {
		it = strings.TrimSpace(strField(m, "instance_type_id"))
	}
	if it != "" && strField(out, "instance_type") == "" {
		out["instance_type"] = it
	}
	cpu := intFromAny(m["cpu_cores"], 0)
	if cpu <= 0 {
		cpu = intFromAny(m["CpuCoreCount"], 0)
	}
	mem := intFromAny(m["memory_gb"], 0)
	if mem <= 0 {
		mem = intFromAny(m["MemorySize"], 0)
	}
	if cpu > 0 {
		out["cpu_cores"] = cpu
	}
	if mem > 0 {
		out["memory_gb"] = mem
	}
}

// applyCachedInstanceTypeSpecToHardware 用缓存的实例规格覆盖 CPU/内存（不改 storage_gb）。
func applyCachedInstanceTypeSpecToHardware(out map[string]interface{}) {
	if out == nil {
		return
	}
	it := strField(out, "instance_type")
	if it == "" {
		return
	}
	spec, ok := getCachedInstanceTypeSpec(it)
	if !ok || spec.cpuCores <= 0 {
		return
	}
	out["cpu_cores"] = spec.cpuCores
	if mem := memoryGBFromInstanceTypeSpec(spec); mem > 0 {
		out["memory_gb"] = mem
	}
}

func memoryGBFromInstanceTypeSpec(spec instanceTypeSpec) int {
	if spec.memoryGB <= 0 {
		return 0
	}
	return int(spec.memoryGB + 0.5)
}

// resolveStartVmRuntimeSource prefers body.runtime_source, then start_reason; default cloud_vm.
func resolveStartVmRuntimeSource(body map[string]interface{}) string {
	if body == nil {
		return "cloud_vm"
	}
	for _, key := range []string{"runtime_source", "start_reason"} {
		if s := strings.TrimSpace(strField(body, key)); s != "" {
			return s
		}
	}
	return "cloud_vm"
}

// alignHardwareWithAuthInstanceType 在缓存未命中时用云厂商 DescribeInstanceTypes 校正 CPU/内存。
func alignHardwareWithAuthInstanceType(auth *cloudAuthRecord, region string, hw map[string]interface{}) {
	if auth == nil || hw == nil {
		return
	}
	it := strField(hw, "instance_type")
	if it == "" {
		return
	}
	if spec, ok := getCachedInstanceTypeSpec(it); ok && spec.cpuCores > 0 {
		hw["cpu_cores"] = spec.cpuCores
		if m := memoryGBFromInstanceTypeSpec(spec); m > 0 {
			hw["memory_gb"] = m
		}
		return
	}
	if strings.TrimSpace(auth.PlatformType) != "aliyun" {
		return
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = "cn-hangzhou"
	}
	rows, err := aliyunDescribeInstanceTypes(auth.SecretID, auth.SecretKey, region, []string{it})
	if err != nil || len(rows) == 0 {
		return
	}
	row := rows[0]
	cpu := intFromAny(row["cpu_cores"], 0)
	mem := intFromAny(row["memory_gb"], 0)
	if cpu <= 0 {
		return
	}
	hw["cpu_cores"] = cpu
	if mem > 0 {
		hw["memory_gb"] = mem
	}
}

func intFromAny(v interface{}, def int) int {
	if v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n + 0.5)
	case float32:
		return int(n + 0.5)
	case int:
		return n
	case int64:
		return int(n)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return def
		}
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f + 0.5)
		}
		return def
	}
}

type startVmFinalizeInput struct {
	TenantID                   string
	WorkspaceID                string
	UserID                     string
	TraceID                    string
	Body                       map[string]interface{}
	CloudAuth                  *cloudAuthRecord
	ResolvedCloudServerImageID string
	ResolvedRegionID           string
	ContainerImageURL          string
}

// startVmAutoFinalizeInput is an alias kept for call sites during Phase 3i cutover.
type startVmAutoFinalizeInput = startVmFinalizeInput

func buildStartVmEventData(in startVmFinalizeInput) (map[string]interface{}, string) {
	body := in.Body
	hw := buildHardwareEventPayload(body)
	if errMsg := strField(hw, "_error"); errMsg != "" {
		return nil, errMsg
	}
	delete(hw, "_error")
	// 自动/模版启动常见占位核内：有云凭证时按实例规格实时核查覆盖
	alignHardwareWithAuthInstanceType(in.CloudAuth, in.ResolvedRegionID, hw)

	vpcID := strField(body, "vpc_id")
	autoCreateVpc := vpcID == "" || vpcID == "auto_create_vpc"
	verificationSecret := generateVerificationSecret()

	externalImageID := ""
	if img, err := getInstalledImage(in.TenantID, strField(body, "container_image_id")); err == nil && img != nil {
		externalImageID = strings.TrimSpace(img.ExternalImageID)
	}
	plain, udErr := fetchAIPublicRuntimeUserdataBody(
		externalImageID,
		in.ResolvedCloudServerImageID,
		in.CloudAuth.PlatformType,
		in.ResolvedRegionID,
	)
	if udErr != nil {
		return nil, "调用镜像市场 UserData API 失败: " + udErr.Error()
	}
	if plain == "" {
		return nil, "未从镜像市场获取到 UserData。请在镜像市场为对应宿主机镜像绑定 UserData 模板，或上传宿主机 UserData，并确认地域与云平台与本次启动一致。"
	}
	verifyScript := buildVerificationShellSnippet(in.TenantID, in.WorkspaceID, strField(body, "task_id"), verificationSecret)
	userdataContent := buildUserdataExecutionWrapperScript(plain, verifyScript)

	runtimeSource := resolveStartVmRuntimeSource(body)
	eventData := map[string]interface{}{
		"task_id":                    strField(body, "task_id"),
		"trace_id":                   strings.TrimSpace(in.TraceID),
		"cloud_server_image_id":      in.ResolvedCloudServerImageID,
		"container_image_id":         strField(body, "container_image_id"),
		"container_image_url":        in.ContainerImageURL,
		"hardware_config":            hw,
		"region_id":                  in.ResolvedRegionID,
		"zone_id":                    strField(body, "zone_id"),
		"vswitch_id":                 body["vswitch_id"],
		"vpc_id":                     nilIfAutoCreateVpc(autoCreateVpc, vpcID),
		"auto_create_vpc":            autoCreateVpc,
		"auto_create_vswitch":        boolField(body, "auto_create_vswitch"),
		"cloud_platform_id":          body["cloud_platform_id"],
		"cloud_platform_type":        in.CloudAuth.PlatformType,
		"authorization_id":           in.CloudAuth.ID,
		"company_id":                 in.TenantID,
		"workspace_id":               in.WorkspaceID,
		"selected_instance":          normalizeSelectedInstance(body["selected_instance"]),
		"security_group_id":          body["security_group_id"],
		"auto_create_security_group": boolField(body, "auto_create_security_group"),
		"bandwidth":                  body["bandwidth"],
		"bandwidth_charging_mode":    body["bandwidth_charging_mode"],
		"verification_secret":        verificationSecret,
		"userdata_task_api_endpoint": userdataVerifyBaseURL(),
		"userdata_content":           userdataContent,
		"runtime_source":             runtimeSource,
	}
	if clientIP := strings.TrimSpace(strField(body, "client_public_ip")); clientIP != "" {
		eventData["client_public_ip"] = clientIP
	}
	if boolField(body, "auto_create_security_group") {
		eventData["auto_sg_whitelist"] = true
	}
	if extras := resolveExtraIngressCIDRs(body); len(extras) > 0 {
		eventData["extra_ingress_cidrs"] = extras
	}
	// Comment / container identity for SSE log_label [实例ID、容器名].
	if cid := strField(body, "comment_id"); cid != "" {
		eventData["comment_id"] = cid
	}
	// 评论级 CSC 行标识：RunInstances 后按 csc_id 精确落 instance_id（孤儿回收防误删窗口）
	if sid := strField(body, "csc_id"); sid != "" {
		eventData["csc_id"] = sid
	}
	if pcid := strField(body, "parent_comment_id"); pcid != "" {
		eventData["parent_comment_id"] = pcid
	}
	if iid := strField(body, "installed_image_id"); iid != "" {
		eventData["installed_image_id"] = iid
	}
	if skill := resolveStartVmImageSkill(in.TenantID,
		firstNonEmpty(strField(body, "installed_image_id"), strField(body, "container_image_id")),
		strField(body, "image_skill")); skill != "" {
		eventData["image_skill"] = skill
	}
	if sv := resolveStartVmSaasInboundSkillVersion(in.TenantID,
		firstNonEmpty(strField(body, "installed_image_id"), strField(body, "container_image_id"))); sv != "" {
		eventData["saas_inbound_skill_version"] = sv
	}
	cname := strField(body, "container_name")
	if cname == "" {
		cname = strField(body, "mock_container_name")
	}
	if cname == "" {
		taskID := strField(body, "task_id")
		cid := firstNonEmpty(strField(body, "parent_comment_id"), strField(body, "comment_id"))
		cname = buildCommentMockContainerName(taskID, cid)
	}
	if cname != "" {
		eventData["container_name"] = cname
	}
	return eventData, ""
}

// resolveExtraIngressCIDRs merges body extras with TASK2APP_SG_EXTRA_INGRESS_CIDRS.
func resolveExtraIngressCIDRs(body map[string]interface{}) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		s := strings.TrimSpace(raw)
		if s == "" || s == "0.0.0.0/0" || s == "::/0" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if body != nil {
		switch v := body["extra_ingress_cidrs"].(type) {
		case []interface{}:
			for _, item := range v {
				add(fmt.Sprint(item))
			}
		case []string:
			for _, item := range v {
				add(item)
			}
		case string:
			for _, part := range strings.FieldsFunc(v, func(r rune) bool {
				return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
			}) {
				add(part)
			}
		}
	}
	for _, part := range strings.FieldsFunc(os.Getenv("TASK2APP_SG_EXTRA_INGRESS_CIDRS"), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	}) {
		add(part)
	}
	// Always include SaaS egress so container probes (layer-graph / heartbeat) are not blocked by whitelist SG.
	add(detectPlatformEgressCIDR())
	return out
}

func nilIfAutoCreateVpc(auto bool, vpcID string) interface{} {
	if auto {
		return nil
	}
	if vpcID == "" {
		return nil
	}
	return vpcID
}
