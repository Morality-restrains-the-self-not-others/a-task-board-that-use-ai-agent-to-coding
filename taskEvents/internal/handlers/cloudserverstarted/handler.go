package cloudserverstarted

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/cloud/userdata"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/cloudconfig"
	"taskEvents/internal/repository/saas"
	"tracelog"
)

// VMStarter abstracts ECS RunInstances (mockable in tests).
type VMStarter interface {
	StartVM(ctx context.Context, accessKey, secretKey string, in aliyun.StartVMInput) (aliyun.StartVMResult, error)
}

// VMStopper abstracts ECS DeleteInstance for superseding an existing binding before start.
type VMStopper interface {
	StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error)
}

type aliyunStarter struct{}

func (aliyunStarter) StartVM(ctx context.Context, accessKey, secretKey string, in aliyun.StartVMInput) (aliyun.StartVMResult, error) {
	return aliyun.StartVM(ctx, accessKey, secretKey, in)
}

type aliyunStopper struct{}

func (aliyunStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	return aliyun.StopVM(accessKey, secretKey, in)
}

// Handler processes CLOUD_SERVER_STARTED.
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
	Starter   VMStarter
	Stopper   VMStopper
}

func (h *Handler) starter() VMStarter {
	if h.Starter != nil {
		return h.Starter
	}
	return aliyunStarter{}
}

func (h *Handler) stopper() VMStopper {
	if h.Stopper != nil {
		return h.Stopper
	}
	return aliyunStopper{}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "CLOUD_SERVER_STARTED" {
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
	platformType := payload.StrField(data, "cloud_platform_type")
	if cloudcommon.IsLocalSkipCloudPlatform(platformType) {
		tracelog.LogEventConsume(ctx, "start_vm_local_skip", cmd.EventType, map[string]any{
			"task_id":  taskID,
			"platform": platformType,
		})
		return domain.DispatchSuccess, nil
	}
	if platformType != "" && platformType != "aliyun" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported platform %s", platformType)
	}

	scope := cloudcommon.ScopeFromEventData(data)
	workspaceIDEarly := payload.StrField(data, "workspace_id")
	if scope.InstanceID == "" && workspaceIDEarly != "" {
		if row, loadErr := cloudconfig.LoadForTask(companyID, workspaceIDEarly, taskID); loadErr == nil && row != nil {
			scope = cloudcommon.WithInstanceID(scope, row.InstanceID)
		}
	}
	var sseEvents []map[string]interface{}
	defer func() {
		_ = cloudcommon.PublishSSEBatch(ctx, h.Publisher, sseEvents)
	}()

	event, err := h.Repo.LatestPendingStartEvent(companyID, taskID)
	if err != nil {
		// OPT-20260818-015：无 pending 事件可能是「该事件已完成/正在被并发消费」后的重放。
		// 按载荷 event_id 查 DB 事件状态，若已是 success 或 processing 则幂等跳过而非投 DLT
		// （processing 表示另一消费者已原子 claim 正在启动，重放不得二次启动 VM）。
		if evID := payload.StrField(data, "event_id"); evID != "" {
			if st, statusErr := h.Repo.CloudServerEventStatusByID(evID); statusErr == nil {
				if st == saas.CloudEventSuccess || st == saas.CloudEventProcessing {
					tracelog.LogEventConsume(ctx, "idempotency skip", cmd.EventType, map[string]any{
						"task_id": taskID,
						"event_id": evID,
						"status":  st,
					})
					return domain.DispatchSuccess, nil
				}
			}
		}
		log.Printf("[cloud_server_started] event lookup: %v", err)
		tracelog.LogEventConsume(ctx, "pending_event_missing", cmd.EventType, map[string]any{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return domain.DispatchPermanent, err
	}
	// OPT-20260818-015 剩余 gap：pending→processing 用 from_status 守卫原子 claim，
	// 并发消费者只会有一方 claim 成功；丢失 claim 的一方幂等跳过（不重复启动 VM）。
	claimed, claimErr := h.Repo.ClaimCloudServerStartEvent(event.ID)
	if claimErr != nil {
		tracelog.LogEventConsume(ctx, "claim_start_event_failed", cmd.EventType, map[string]any{
			"task_id": taskID, "event_id": fmt.Sprint(event.ID), "error": claimErr.Error(),
		})
		return domain.DispatchRetryable, claimErr
	}
	if !claimed {
		tracelog.LogEventConsume(ctx, "claim_lost_concurrent", cmd.EventType, map[string]any{
			"task_id": taskID, "event_id": fmt.Sprint(event.ID),
		})
		return domain.DispatchSuccess, nil
	}
	tracelog.LogEventConsume(ctx, "start_vm_begin", cmd.EventType, map[string]any{
		"task_id":  taskID,
		"event_id": fmt.Sprint(event.ID),
	})

	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id": taskID,
		"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "processing", "message": fmt.Sprintf("%s服务器正在启动中...", defaultPlatform(platformType)),
			"progress": 80, "event_name": "server_on_start_test",
		}, scope),
	})

	// authorization_id 为平台 SSOT 字符串 ID（cpa_<snowflake>），不可用 Int64Field
	// 解析（strconv.ParseInt 对 "cpa_..." 失败 → 误报缺少 authorization_id，OPT-20260809-026）。
	authID := payload.StrField(data, "authorization_id")
	if authID == "" {
		h.failEvent(ctx, event.ID, taskID, &sseEvents, "缺少 authorization_id", scope)
		return domain.DispatchPermanent, fmt.Errorf("missing authorization_id")
	}
	auth, err := h.Repo.CloudAuthorizationByID(authID)
	if err != nil {
		h.failEvent(ctx, event.ID, taskID, &sseEvents, err.Error(), scope)
		return domain.DispatchPermanent, err
	}

	hw, _ := data["hardware_config"].(map[string]interface{})
	regionID := payload.StrField(data, "region_id")
	if regionID == "" && hw != nil {
		regionID = payload.StrField(hw, "region_id")
	}
	zoneID := payload.StrField(data, "zone_id")
	if zoneID == "" && hw != nil {
		zoneID = payload.StrField(hw, "zone_id")
	}
	imageID := payload.StrField(data, "cloud_server_image_id")
	containerImageID := payload.StrField(data, "container_image_id")
	containerImageURL := payload.StrField(data, "container_image_url")
	if containerImageID != "" && containerImageURL == "" {
		msg := "missing container_image_url"
		h.failEvent(ctx, event.ID, taskID, &sseEvents, msg, scope)
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}

	userdataPlain := payload.StrField(data, "userdata_content")
	replacedUserdata, udErr := userdata.ApplyRunInstancesUserdataPlaceholders(
		userdataPlain,
		data,
		payload.StrField,
	)
	if udErr != nil {
		msg := fmt.Sprintf("UserData 占位符替换失败: %v", udErr)
		h.failEvent(ctx, event.ID, taskID, &sseEvents, msg, scope)
		return domain.DispatchPermanent, fmt.Errorf("%s", msg)
	}

	workspaceID := payload.StrField(data, "workspace_id")
	runtimeSource := payload.StrField(data, "runtime_source")
	if runtimeSource == "" {
		runtimeSource = payload.StrField(data, "start_reason")
	}
	sup, err := h.supersedeExistingInstance(ctx, companyID, workspaceID, taskID, regionID, auth.SecretID, auth.SecretKey, runtimeSource)
	if err != nil {
		msg := fmt.Sprintf("释放旧实例失败，取消本次启动: %v", err)
		h.failEvent(ctx, event.ID, taskID, &sseEvents, msg, scope)
		tracelog.LogEventConsume(ctx, "supersede_old_instance_failed", cmd.EventType, map[string]any{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return domain.DispatchRetryable, err
	}
	if sup.ReusedExisting {
		scope = cloudcommon.WithInstanceID(scope, sup.InstanceID)
		_ = h.Repo.UpdateCloudServerEventStatus(event.ID, saas.CloudEventSuccess, "")
		sseEvents = append(sseEvents, map[string]interface{}{
			"task_id": taskID,
			"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
				"status": "success",
				"message": fmt.Sprintf("已附着本任务现有实例 %s（不释放、不二次冷启动）",
					sup.InstanceID),
				"progress": 100, "event_name": "server_status_update",
				"reuse": true, "inflight_attach": true, "instance_id": sup.InstanceID,
				"vm_info": map[string]interface{}{"instance_id": sup.InstanceID},
			}, scope),
		})
		tracelog.LogEventConsume(ctx, "start_vm_reuse_inflight", cmd.EventType, map[string]any{
			"task_id":     taskID,
			"event_id":    fmt.Sprint(event.ID),
			"instance_id": sup.InstanceID,
		})
		return domain.DispatchSuccess, nil
	}

	vmIn := aliyun.StartVMInput{
		RegionID:        regionID,
		ZoneID:          zoneID,
		VPCID:           payload.StrField(data, "vpc_id"),
		ImageID:         imageID,
		InstanceType:    payload.StrField(hw, "instance_type"),
		SecurityGroupID: payload.StrField(data, "security_group_id"),
		VSwitchID:       payload.StrField(data, "vswitch_id"),
		TaskID:          taskID,
		CommentID:       scope.CommentID,
		UserData:        replacedUserdata,
		HardwareConfig:  hw,
	}
	result, err := h.starter().StartVM(ctx, auth.SecretID, auth.SecretKey, vmIn)
	if err != nil {
		msg := fmt.Sprintf("启动服务器失败: %v", aliyun.FormatStartVMError(err, zoneID))
		tracelog.LogEventConsume(ctx, "start_vm_failed", cmd.EventType, map[string]any{
			"task_id":  taskID,
			"event_id": fmt.Sprint(event.ID),
			"error":    msg,
		})
		h.failEvent(ctx, event.ID, taskID, &sseEvents, msg, scope)
		if aliyun.IsPermanentStartVMError(err) {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	scope = cloudcommon.WithInstanceID(scope, result.InstanceID)

	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id": taskID,
		"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "sdk_call", "message": "云厂商SDK调用信息", "event_name": "server_status_update",
			"instance_id": result.InstanceID,
			"sdk_call": map[string]interface{}{
				"method": "start_vm", "request": data,
				"response":    map[string]interface{}{"instance_id": result.InstanceID},
				"request_ids": result.RequestID,
			},
		}, scope),
	})

	vmResult := cloudconfig.StartVMResult{
		Platform:        defaultPlatform(platformType),
		InstanceID:      result.InstanceID,
		SecurityGroupID: vmIn.SecurityGroupID,
		VSwitchID:       firstNonEmpty(result.VSwitchID, vmIn.VSwitchID),
		Region:          regionID,
		ZoneID:          firstNonEmpty(result.ZoneID, zoneID),
		LaunchRequestID: result.LaunchRequestID,
		ClientToken:     result.ClientToken,
		CommentID:       scope.CommentID,
		CSCID:           payload.StrField(data, "csc_id"),
	}
	if err := cloudconfig.UpsertAfterStart(companyID, workspaceID, taskID, fmt.Sprint(authID), payload.StrField(data, "verification_secret"), vmResult); err != nil {
		h.failEvent(ctx, event.ID, taskID, &sseEvents, err.Error(), scope)
		return domain.DispatchRetryable, err
	}
	// 实例规格权威校正 CPU/内存，避免历史卡出现「同规格不同核内」
	aliyun.AlignHardwareCPUMemoryWithInstanceType(auth.SecretID, auth.SecretKey, regionID, hw)
	if err := cloudconfig.InsertHistoryAfterStart(companyID, workspaceID, taskID, fmt.Sprint(authID), hw, vmResult, runtimeSource); err != nil {
		log.Printf("[cloud_server_started] config history: %v", err)
	}
	_ = h.Repo.UpdateCloudServerEventStatus(event.ID, saas.CloudEventSuccess, "")

	publicIP := ""
	if boolField(data, "auto_sg_whitelist") {
		sgID := payload.StrField(data, "security_group_id")
		if sgID == "" {
			sgID = vmIn.SecurityGroupID
		}
		if sgID != "" {
			ip, waitErr := aliyun.WaitInstancePublicIP(ctx, auth.SecretID, auth.SecretKey, regionID, result.InstanceID)
			if waitErr != nil {
				log.Printf("[cloud_server_started] wait public ip for sg whitelist: %v task=%s", waitErr, taskID)
			} else if ip != "" {
				publicIP = ip
				if err := aliyun.AuthorizeServerPublicIPIngress(auth.SecretID, auth.SecretKey, regionID, sgID, ip, strSliceField(data, "extra_ingress_cidrs")...); err != nil {
					log.Printf("[cloud_server_started] authorize server public ip ingress failed: %v sg=%s ip=%s task=%s",
						err, sgID, ip, taskID)
					sseEvents = append(sseEvents, map[string]interface{}{
						"task_id": taskID,
						"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
							"status":   "processing",
							"message":  fmt.Sprintf("服务器已启动，但安全组服务器IP白名单写入失败: %v", err),
							"progress": 95, "event_name": "server_status_update",
						}, scope),
					})
				} else {
					log.Printf("[cloud_server_started] sg whitelist server_ip=%s sg=%s task=%s", ip, sgID, taskID)
				}
			}
		}
	}

	vmInfo := map[string]interface{}{"instance_id": result.InstanceID}
	if publicIP != "" {
		vmInfo["public_ip"] = publicIP
	}
	sseEvents = append(sseEvents, map[string]interface{}{
		"task_id": taskID,
		"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "success", "message": fmt.Sprintf("%s服务器启动成功！", defaultPlatform(platformType)),
			"progress": 100, "event_name": "server_status_update",
			"vm_info": vmInfo,
		}, scope),
	})
	log.Printf("[cloud_server_started] success task=%s instance=%s", taskID, result.InstanceID)
	tracelog.LogEventConsume(ctx, "start_vm_success", cmd.EventType, map[string]any{
		"task_id":     taskID,
		"event_id":    fmt.Sprint(event.ID),
		"instance_id": result.InstanceID,
		"public_ip":   publicIP,
	})
	return domain.DispatchSuccess, nil
}

func (h *Handler) failEvent(ctx context.Context, eventID int64, taskID string, sse *[]map[string]interface{}, msg string, scope cloudcommon.StartupLogScope) {
	_ = h.Repo.UpdateCloudServerEventStatus(eventID, saas.CloudEventError, msg)
	*sse = append(*sse, map[string]interface{}{
		"task_id": taskID,
		"status_data": cloudcommon.ApplyStartupLogScope(map[string]interface{}{
			"status": "error", "message": msg, "progress": 0, "event_name": "server_status_update",
		}, scope),
	})
}

func defaultPlatform(p string) string {
	if p == "" {
		return "云"
	}
	return p
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}
