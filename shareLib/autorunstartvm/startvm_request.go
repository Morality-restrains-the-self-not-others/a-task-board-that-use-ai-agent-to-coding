package autorunstartvm

import (
	"fmt"
	"strconv"
	"strings"
)

// StartVmRequest is the Go port of taskFE buildAutoRunStartVmRequest.
type StartVmRequest struct {
	APIPath string
	Body    map[string]interface{}
}

// BuildAutoRunStartVmRequest mirrors projectRunTemplateUtils.js buildAutoRunStartVmRequest.
func BuildAutoRunStartVmRequest(taskID, containerImageID string, runTemplate map[string]interface{}) (*StartVmRequest, error) {
	shape := runTemplateToServerConfigShape(runTemplate)
	if shape == nil || strings.TrimSpace(taskID) == "" || strings.TrimSpace(containerImageID) == "" {
		return nil, fmt.Errorf("missing task_id, container_image_id, or run template")
	}
	region := strAny(shape["region"])
	platformID := strAny(shape["platform_id"])
	if region == "" || platformID == "" {
		return nil, fmt.Errorf("run template missing region or platform_id")
	}

	vpcID := strAny(shape["vpc_id"])
	vswitchID := strAny(shape["vswitch_id"])
	sgID := strAny(shape["security_group_id"])
	zoneID := strAny(shape["zone_id"])

	autoCreateVPC := vpcID == ""
	autoCreateVSwitch := vswitchID == "" && zoneID != ""
	autoCreateSG := sgID == ""
	apiPath := "start-vm"
	if autoCreateVPC || autoCreateVSwitch || autoCreateSG {
		apiPath = "start-vm-auto"
	}

	instanceType := strAny(runTemplate["selected_instance"])
	hw, _ := shape["hardware_config"].(map[string]interface{})
	if instanceType == "" {
		instanceType = strAny(hw["instance_type"])
	}

	filterOpts, _ := runTemplate["filter_options"].(map[string]interface{})
	if filterOpts == nil {
		filterOpts, _ = shape["filter_options"].(map[string]interface{})
	}
	if filterOpts == nil {
		filterOpts = map[string]interface{}{}
	}

	resolved := resolveHardwareCoresMemory(hw, filterOpts)
	hardwareConfig := map[string]interface{}{
		"cpu_cores":  toNumber(resolved["cpu_cores"]),
		"memory_gb":  toNumber(resolved["memory_gb"]),
		"storage_gb": toNumber(defaultAny(hw["storage_gb"], 40)),
	}
	if instanceType != "" {
		hardwareConfig["instance_type"] = instanceType
	}

	var vpcOut interface{}
	if autoCreateVPC {
		vpcOut = nil
	} else {
		vpcOut = vpcID
	}
	var vswOut interface{}
	if vswitchID == "" {
		vswOut = nil
	} else {
		vswOut = vswitchID
	}
	var sgOut interface{}
	if sgID == "" {
		sgOut = nil
	} else {
		sgOut = sgID
	}
	var zoneOut interface{}
	if zoneID == "" {
		zoneOut = nil
	} else {
		zoneOut = zoneID
	}
	var authOut interface{}
	authID := strAny(shape["authorization_id"])
	if authID == "" {
		authOut = nil
	} else {
		authOut = authID
	}
	var selectedOut interface{}
	if instanceType == "" {
		selectedOut = nil
	} else {
		selectedOut = instanceType
	}

	bandwidth := runTemplate["bandwidth"]
	if bandwidth == nil {
		bandwidth = 1
	}
	bwMode := strAny(runTemplate["bandwidth_charging_mode"])
	if bwMode == "" {
		bwMode = "PayByTraffic"
	}

	body := map[string]interface{}{
		"task_id":                    taskID,
		"container_image_id":         containerImageID,
		"hardware_config":            hardwareConfig,
		"region_id":                  region,
		"zone_id":                    zoneOut,
		"vpc_id":                     vpcOut,
		"vswitch_id":                 vswOut,
		"auto_create_vswitch":        autoCreateVSwitch,
		"cloud_platform_id":          platformID,
		"authorization_id":           authOut,
		"filter_options":             filterOpts,
		"selected_instance":          selectedOut,
		"security_group_id":          sgOut,
		"auto_create_security_group": autoCreateSG,
		"bandwidth":                  bandwidth,
		"bandwidth_charging_mode":    bwMode,
		// 默认按自动运行入口；调用方可 ApplyRuntimeSource 覆盖
		"runtime_source": RuntimeSourceCloudVMAutoRun,
	}
	return &StartVmRequest{APIPath: apiPath, Body: body}, nil
}

// ResolveProjectServerRunTemplateFromProjects mirrors resolveProjectServerRunTemplateFromProjects.js.
func ResolveProjectServerRunTemplateFromProjects(projects []map[string]interface{}, linkedProjectRows []map[string]interface{}) map[string]interface{} {
	if len(projects) == 0 {
		return nil
	}
	linkedIDs := make([]string, 0, len(linkedProjectRows))
	for _, row := range linkedProjectRows {
		id := strings.TrimSpace(strAny(row["project_id"]))
		if id != "" {
			linkedIDs = append(linkedIDs, id)
		}
	}
	ordered := projects
	if len(linkedIDs) > 0 {
		byID := map[string]map[string]interface{}{}
		for _, p := range projects {
			byID[strAny(p["id"])] = p
		}
		ordered = nil
		for _, id := range linkedIDs {
			if p, ok := byID[id]; ok {
				ordered = append(ordered, p)
			}
		}
	}
	for _, project := range ordered {
		tpl, ok := project["server_run_template"].(map[string]interface{})
		if !ok || len(tpl) == 0 {
			continue
		}
		return tpl
	}
	return nil
}

// RunTemplateIsConfigured reports whether a server_run_template has any meaningful fields
// (aligned with taskFE projectHasConfiguredRunTemplate / summarizeRunTemplate !== 未设置).
func RunTemplateIsConfigured(tpl map[string]interface{}) bool {
	if tpl == nil {
		return false
	}
	for _, v := range tpl {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
			continue
		}
		return true
	}
	return false
}

func runTemplateToServerConfigShape(template map[string]interface{}) map[string]interface{} {
	if template == nil {
		return nil
	}
	hw, _ := template["hardware_config"].(map[string]interface{})
	if hw == nil {
		hw = map[string]interface{}{}
	}
	filterOpts, _ := template["filter_options"].(map[string]interface{})
	if filterOpts == nil {
		filterOpts = map[string]interface{}{}
	}
	platformID := strAny(template["cloud_platform_id"])
	if platformID == "" {
		platformID = strAny(template["platform_id"])
	}
	return map[string]interface{}{
		"platform":          strAny(template["platform"]),
		"platform_id":       platformID,
		"region":            strAny(template["region"]),
		"zone_id":           strings.TrimSpace(strAny(template["zone_id"])),
		"vpc_id":            strings.TrimSpace(strAny(template["vpc_id"])),
		"vswitch_id":        strings.TrimSpace(strAny(template["vswitch_id"])),
		"security_group_id": strAny(template["security_group_id"]),
		"authorization_id":  strAny(template["authorization_id"]),
		"hardware_config": map[string]interface{}{
			"cpu_cores":     defaultAny(hw["cpu_cores"], "1"),
			"memory_gb":     defaultAny(hw["memory_gb"], "1"),
			"storage_gb":    defaultAny(hw["storage_gb"], "40"),
			"instance_type": hw["instance_type"],
		},
		"filter_options": filterOpts,
	}
}

func resolveHardwareCoresMemory(hw, filterOptions map[string]interface{}) map[string]interface{} {
	if hw == nil {
		hw = map[string]interface{}{}
	}
	if filterOptions == nil {
		filterOptions = map[string]interface{}{}
	}
	hwCPU := strAny(hw["cpu_cores"])
	hwMem := strAny(hw["memory_gb"])
	if isPlaceholderHardware(hw) {
		filterCPU := filterOptions["cores"]
		filterMem := filterOptions["memory"]
		if filterCPU != nil || filterMem != nil {
			cpu := defaultAny(filterCPU, hwCPU)
			if strAny(cpu) == "" {
				cpu = "1"
			}
			mem := defaultAny(filterMem, hwMem)
			if strAny(mem) == "" {
				mem = "1"
			}
			return map[string]interface{}{"cpu_cores": strAny(cpu), "memory_gb": strAny(mem)}
		}
	}
	cpu := hwCPU
	if cpu == "" {
		cpu = strAny(defaultAny(filterOptions["cores"], "1"))
	}
	mem := hwMem
	if mem == "" {
		mem = strAny(defaultAny(filterOptions["memory"], "1"))
	}
	return map[string]interface{}{"cpu_cores": cpu, "memory_gb": mem}
}

func isPlaceholderHardware(hw map[string]interface{}) bool {
	return strAny(hw["cpu_cores"]) == "1" && strAny(hw["memory_gb"]) == "1"
}

func strAny(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func defaultAny(v, def interface{}) interface{} {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		return def
	}
	return v
}

func toNumber(v interface{}) interface{} {
	s := strAny(v)
	if s == "" {
		return 0
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return v
}
