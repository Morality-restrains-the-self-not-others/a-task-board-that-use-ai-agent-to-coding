package cloudserverstartauto

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
	"tracelog"
)

// NetworkProvisioner abstracts VPC/VSwitch/SG auto-create (mockable in tests).
type NetworkProvisioner interface {
	Provision(ctx context.Context, accessKey, secretKey string, in aliyun.AutoNetworkInput) (aliyun.AutoNetworkResult, error)
}

type aliyunNetworkProvisioner struct{}

func (aliyunNetworkProvisioner) Provision(ctx context.Context, accessKey, secretKey string, in aliyun.AutoNetworkInput) (aliyun.AutoNetworkResult, error) {
	return aliyun.ProvisionAutoNetworkResources(ctx, accessKey, secretKey, in)
}

// Handler processes CLOUD_SERVER_START_AUTO (resource merge + chain to CLOUD_SERVER_STARTED).
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
	Network   NetworkProvisioner
}

func (h *Handler) provisioner() NetworkProvisioner {
	if h.Network != nil {
		return h.Network
	}
	return aliyunNetworkProvisioner{}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "CLOUD_SERVER_START_AUTO" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	taskID := payload.StrField(data, "task_id")
	companyID, err := payload.Int64Field(data, "company_id")
	if err != nil || taskID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing task_id or company_id")
	}
	regionID := payload.StrField(data, "region_id")
	zoneID := payload.StrField(data, "zone_id")
	if regionID == "" || zoneID == "" {
		return domain.DispatchPermanent, fmt.Errorf("region_id and zone_id required")
	}
	platformType := payload.StrField(data, "cloud_platform_type")
	if cloudcommon.IsLocalSkipCloudPlatform(platformType) {
		tracelog.LogEventConsume(ctx, "auto_network_local_skip", cmd.EventType, map[string]any{
			"task_id":  taskID,
			"platform": platformType,
		})
		return domain.DispatchSuccess, nil
	}
	if platformType != "" && platformType != "aliyun" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported platform %s", platformType)
	}

	autoVPC := boolField(data, "auto_create_vpc") || payload.StrField(data, "vpc_id") == ""
	autoVSwitch := boolField(data, "auto_create_vswitch")
	autoSG := boolField(data, "auto_create_security_group")
	scope := cloudcommon.ScopeFromEventData(data)
	pubScoped := func(statusData map[string]interface{}) {
		_ = cloudcommon.PublishSSE(ctx, h.Publisher, taskID, cloudcommon.ApplyStartupLogScope(statusData, scope))
	}

	pubScoped(map[string]interface{}{
		"status": "processing", "message": "正在处理自动创建资源任务...", "progress": 50, "event_name": "server_status_update",
	})
	tracelog.LogEventConsume(ctx, "auto_network_begin", cmd.EventType, map[string]any{
		"task_id": taskID,
	})

	nextData := copyMap(data)

	if autoVPC || autoVSwitch || autoSG {
		// authorization_id 为平台 SSOT 字符串 ID（cpa_<snowflake>），不可用 Int64Field
		// 解析（strconv.ParseInt 对 "cpa_..." 失败 → 误报缺少 authorization_id，OPT-20260809-026）。
		authID := payload.StrField(data, "authorization_id")
		if authID == "" {
			msg := "缺少 authorization_id"
			pubScoped(map[string]interface{}{
				"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
			})
			return domain.DispatchPermanent, fmt.Errorf("%s", msg)
		}
		auth, err := h.Repo.CloudAuthorizationByID(authID)
		if err != nil {
			pubScoped(map[string]interface{}{
				"status": "error", "message": err.Error(), "progress": 0, "event_name": "server_status_update",
			})
			return domain.DispatchPermanent, err
		}

		clientPublicIP := payload.StrField(data, "client_public_ip")
		extraCIDRs := strSliceField(data, "extra_ingress_cidrs")
		netResult, err := h.provisioner().Provision(ctx, auth.SecretID, auth.SecretKey, aliyun.AutoNetworkInput{
			RegionID:          regionID,
			ZoneID:            zoneID,
			AutoVPC:           autoVPC,
			AutoVSwitch:       autoVSwitch,
			AutoSG:            autoSG,
			ExistingVPCID:     payload.StrField(data, "vpc_id"),
			ExistingVSwitchID: payload.StrField(data, "vswitch_id"),
			ExistingSGID:      payload.StrField(data, "security_group_id"),
			ClientPublicIP:    clientPublicIP,
			ExtraIngressCIDRs: extraCIDRs,
		})
		if err != nil {
			msg := fmt.Sprintf("自动创建网络资源失败: %v", err)
			log.Printf("[cloud_server_start_auto] %s", msg)
			pubScoped(map[string]interface{}{
				"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
			})
			return domain.DispatchRetryable, err
		}
		if autoVPC && netResult.VPCID != "" {
			nextData["vpc_id"] = netResult.VPCID
			log.Printf("[cloud_server_start_auto] 自动创建VPC成功 vpc=%s task=%s", netResult.VPCID, taskID)
			pubScoped(map[string]interface{}{
				"status": "processing", "message": "自动创建VPC成功", "progress": 55, "event_name": "server_status_update",
			})
		}
		if autoVSwitch && netResult.VSwitchID != "" {
			nextData["vswitch_id"] = netResult.VSwitchID
			log.Printf("[cloud_server_start_auto] 自动创建交换机成功 vswitch=%s task=%s", netResult.VSwitchID, taskID)
			pubScoped(map[string]interface{}{
				"status": "processing", "message": "自动创建交换机成功", "progress": 60, "event_name": "server_status_update",
			})
		}
		if autoSG && netResult.SecurityGroupID != "" {
			nextData["security_group_id"] = netResult.SecurityGroupID
			nextData["auto_sg_whitelist"] = true
			if clientPublicIP != "" {
				nextData["client_public_ip"] = clientPublicIP
			}
			if len(extraCIDRs) > 0 {
				nextData["extra_ingress_cidrs"] = extraCIDRs
			}
			log.Printf("[cloud_server_start_auto] 自动创建安全组成功(白名单) sg=%s client_ip=%s extras=%v task=%s",
				netResult.SecurityGroupID, clientPublicIP, extraCIDRs, taskID)
			pubScoped(map[string]interface{}{
				"status": "processing", "message": "自动创建安全组成功（入网白名单）", "progress": 65, "event_name": "server_status_update",
			})
		}
	}

	nextData["auto_create_vpc"] = false
	nextData["auto_create_vswitch"] = false
	nextData["auto_create_security_group"] = false

	pubScoped(map[string]interface{}{
		"status": "processing", "message": "前置资源创建完成，开始启动服务器...", "progress": 70, "event_name": "server_status_update",
	})

	// Ensure chained STARTED keeps per-request event_id for consumer idempotency grain.
	if payload.StrField(nextData, "event_id") == "" {
		if event, err := h.Repo.LatestPendingStartEvent(companyID, taskID); err == nil && event != nil && event.ID != 0 {
			nextData["event_id"] = fmt.Sprint(event.ID)
		}
	}

	tracelog.LogEventConsume(ctx, "chain_cloud_server_started", cmd.EventType, map[string]any{
		"task_id":    taskID,
		"event_id":   payload.StrField(nextData, "event_id"),
		"vpc_id":     payload.StrField(nextData, "vpc_id"),
		"vswitch_id": payload.StrField(nextData, "vswitch_id"),
	})
	if err := h.Publisher.PublishEvent(ctx, "CLOUD_SERVER_STARTED", nextData, taskID); err != nil {
		return domain.DispatchRetryable, err
	}

	if event, err := h.Repo.LatestPendingStartEvent(companyID, taskID); err == nil {
		_ = h.Repo.UpdateCloudServerEventData(event.ID, nextData)
	}
	log.Printf("[cloud_server_start_auto] chained CLOUD_SERVER_STARTED task=%s", taskID)
	return domain.DispatchSuccess, nil
}

func boolField(data map[string]interface{}, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return fmt.Sprint(v) == "true"
	}
}

func copyMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func strSliceField(data map[string]interface{}, key string) []string {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{strings.TrimSpace(t)}
	default:
		return nil
	}
}
