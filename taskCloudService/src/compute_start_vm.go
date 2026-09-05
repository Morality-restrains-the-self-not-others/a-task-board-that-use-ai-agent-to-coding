package main

import (
	"context"
	"net/http"
	"strings"

	"tracelog"
)

// bindStartVmTraceContext 为本次 start-vm 绑定独立 TraceId。
// 保留合法入站 X-Trace-Id；缺失或等于 task_id 时生成新 ID。禁止用任务 ID 冒充启机链路。
func bindStartVmTraceContext(ctx context.Context, r *http.Request, taskID string) (context.Context, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	header := ""
	if r != nil {
		header = r.Header.Get(tracelog.Header)
	}
	runTrace := tracelog.ResolveTraceID(ctx, header)
	if runTrace == "" || isTraceEqualToTaskID(runTrace, taskID) {
		runTrace = tracelog.NewTraceID()
	}
	corr := tracelog.CorrelationFromContext(ctx)
	if corr.SpanID == "" {
		corr.SpanID = tracelog.NewSpanID()
	}
	corr.TraceID = runTrace
	return tracelog.ContextWithCorrelation(ctx, corr), runTrace
}

func validateStartVmPayload(body map[string]interface{}) string {
	if strField(body, "task_id") == "" {
		return "缺少任务ID"
	}
	if strField(body, "container_image_id") == "" && strField(body, "cloud_server_image_id") == "" {
		return "缺少容器镜像或云服务器镜像"
	}
	return ""
}

// validateStartVmDirectPayload validates POST start-vm (existing network resources, no auto-create).
func validateStartVmDirectPayload(body map[string]interface{}) string {
	if msg := validateStartVmPayload(body); msg != "" {
		return msg
	}
	if strField(body, "region_id") == "" {
		return "缺少 region_id"
	}
	autoCreateVpc := func() bool {
		vpcID := strField(body, "vpc_id")
		return vpcID == "" || vpcID == "auto_create_vpc"
	}()
	if autoCreateVpc || boolField(body, "auto_create_vswitch") || boolField(body, "auto_create_security_group") {
		return "检测到需要自动创建资源，请调用 start-vm-auto 接口"
	}
	sgID := strField(body, "security_group_id")
	if sgID == "" || sgID == "auto_create_security_group" {
		return "security_group_id 为必填参数"
	}
	if strField(body, "vswitch_id") == "" {
		return "缺少 vswitch_id"
	}
	if strField(body, "vpc_id") == "" {
		return "缺少 vpc_id"
	}
	return ""
}

func validateStartVmAutoPayload(body map[string]interface{}) string {
	if msg := validateStartVmPayload(body); msg != "" {
		return msg
	}
	autoCreateVpc := func() bool {
		vpcID := strField(body, "vpc_id")
		return vpcID == "" || vpcID == "auto_create_vpc"
	}()
	autoCreateVswitch := boolField(body, "auto_create_vswitch")
	autoCreateSecurityGroup := boolField(body, "auto_create_security_group")
	if !autoCreateVpc && !autoCreateVswitch && !autoCreateSecurityGroup {
		return "未检测到需要自动创建的资源，请调用 start-vm 接口"
	}
	return ""
}

func boolField(body map[string]interface{}, key string) bool {
	v, ok := body[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true") || t == "1"
	default:
		return false
	}
}

func enrichStartVmPayloadFromInstalledImage(tenantID string, body map[string]interface{}) {
	if strField(body, "container_image_url") != "" {
		return
	}
	imageID := strField(body, "container_image_id")
	if imageID == "" || tenantID == "" {
		return
	}
	img, err := getInstalledImage(tenantID, imageID)
	if err != nil || img == nil {
		return
	}
	ref := mergeInstalledImageRef(img)
	if ref == "" {
		return
	}
	body["container_image_url"] = ref
}
