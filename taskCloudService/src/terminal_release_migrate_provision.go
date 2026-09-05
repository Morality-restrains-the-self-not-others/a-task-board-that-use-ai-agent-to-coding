package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Hooks for tests / alternate provisioners.
var (
	provisionMigratedRealECSFn           = provisionMigratedRealECS
	startMigratedContainerFn             = startMigratedContainer
	publishContainerMigrateAwaitReadyFn  = publishContainerMigrateAwaitReady
)

type migrateOffResult struct {
	InstanceID              string
	Via                     string
	ContainerStartTriggered bool
	ContainerImageID        string
}

func extractContainerImageHints(companyID, taskID string) (imageID, imageURL string) {
	ev, err := loadLatestStartEvent(companyID, taskID, "", "")
	if err != nil || ev == nil || ev.EventData == nil {
		return "", ""
	}
	return strings.TrimSpace(strField(ev.EventData, "container_image_id")),
		strings.TrimSpace(strField(ev.EventData, "container_image_url"))
}

func clearOwnerBindingKeepingMeta(owner *CloudServerConfig) error {
	if owner == nil {
		return nil
	}
	owner.InstanceID = ""
	owner.PublicIP = ""
	owner.ServerURL = ""
	owner.BusinessAPIEndpoint = ""
	owner.ContainerVscodeURL = ""
	owner.LastRuntimeStatus = ""
	owner.ErrorReason = ""
	if err := upsertCloudServerConfig(*owner); err != nil {
		return err
	}
	return clearConfigIdleSince(owner.CompanyID, owner.WorkspaceID, owner.TaskID)
}

// provisionMigratedRealECS publishes CLOUD_SERVER_START_AUTO rebuilt from the task's latest start event.
// UserData on the new ECS boots the container image — no separate container start needed for this path.
func provisionMigratedRealECS(companyID, workspaceID, ownerTaskID, forbidInstanceID, containerImageID string, owner *CloudServerConfig) (string, error) {
	if owner == nil {
		return "", fmt.Errorf("owner config required")
	}
	authID := strings.TrimSpace(owner.AuthorizationID)
	if authID == "" {
		return "", fmt.Errorf("owner authorization_id required for real ECS migrate")
	}
	cloudAuth, err := loadCloudAuth(companyID, authID)
	if err != nil || cloudAuth == nil {
		return "", fmt.Errorf("load cloud auth: %v", err)
	}

	body := map[string]interface{}{
		"task_id":                    ownerTaskID,
		"authorization_id":           authID,
		"forbid_reuse_instance_id":   forbidInstanceID,
		"auto_create_security_group": true,
		"region_id":                  strings.TrimSpace(owner.Region),
		"zone_id":                    strings.TrimSpace(owner.ZoneID),
	}
	if ev, eerr := loadLatestStartEvent(companyID, ownerTaskID, "", ""); eerr == nil && ev != nil && ev.EventData != nil {
		for _, k := range []string{
			"container_image_id", "container_image_url", "cloud_server_image_id",
			"region_id", "zone_id", "selected_instance", "hardware_config",
			"vpc_id", "vswitch_id", "security_group_id", "bandwidth", "bandwidth_charging_mode",
			"auto_create_vpc", "auto_create_vswitch", "auto_create_security_group",
			"cloud_platform_id", "client_public_ip",
		} {
			if v, ok := ev.EventData[k]; ok && v != nil {
				body[k] = v
			}
		}
	}
	if containerImageID != "" {
		body["container_image_id"] = containerImageID
	}
	if strField(body, "region_id") == "" {
		body["region_id"] = "cn-hangzhou"
	}
	body["runtime_source"] = "cloud_vm_terminal_migrate"
	enrichStartVmPayloadFromInstalledImage(companyID, body)

	regionID := strField(body, "region_id")
	instanceType := strField(body, "selected_instance")
	if instanceType == "" {
		if hw, ok := body["hardware_config"].(map[string]interface{}); ok {
			instanceType = strField(hw, "instance_type")
		}
	}
	desiredArch := inferInstanceArchitecture(instanceType)
	resolvedImageID, resolvedRegion, imgErr := resolveCloudServerImageID(
		companyID,
		strField(body, "cloud_server_image_id"),
		strField(body, "container_image_id"),
		cloudAuth.PlatformType,
		regionID,
		desiredArch,
	)
	if imgErr != "" {
		return "", fmt.Errorf("resolve cloud server image: %s", imgErr)
	}
	if resolvedImageID == "" {
		return "", fmt.Errorf("no cloud server image for migrate start-vm-auto")
	}
	if resolvedRegion != "" {
		body["region_id"] = resolvedRegion
	}

	if err := clearOwnerBindingKeepingMeta(owner); err != nil {
		return "", err
	}
	_, _ = db.Exec(
		`UPDATE cloud_server_configs SET terminal_released=0 WHERE company_id=? AND workspace_id=? AND task_id=?`,
		companyID, workspaceID, ownerTaskID,
	)

	ctx := context.Background()
	traceID := ownerTaskID
	result, statusCode, finErr := finalizeStartVmInGo(ctx, startVmFinalizeInput{
		TenantID:                   companyID,
		WorkspaceID:                workspaceID,
		UserID:                     "system-migrate",
		TraceID:                    traceID,
		Body:                       body,
		CloudAuth:                  cloudAuth,
		ResolvedCloudServerImageID: resolvedImageID,
		ResolvedRegionID:           strField(body, "region_id"),
		ContainerImageURL:          strField(body, "container_image_url"),
	})
	if finErr != nil {
		return "", fmt.Errorf("finalize start-vm-auto (status=%d): %w", statusCode, finErr)
	}
	// Go 原生异步：自动创建 VPC/SG/vSwitch + RunInstances。
	// 终端释放迁移场景允许后台执行，不阻塞调用方。
	eventID := result.EventID
	go func() {
		vmResult, vmErr := executeStartVmAutoNative(
			cloudAuth.SecretID, cloudAuth.SecretKey,
			strField(result.EventData, "region_id"),
			strField(result.EventData, "zone_id"),
			result.EventData,
		)
		if vmErr != nil {
			logInfo("event=container_migrate_start_vm_auto_failed tenant="+companyID+
				" owner_task="+ownerTaskID+" event_id="+eventID+" err="+vmErr.Error(), ownerTaskID)
			return
		}
		logInfo("event=container_migrate_start_vm_auto_ok tenant="+companyID+
			" owner_task="+ownerTaskID+" event_id="+eventID+
			" instance_id="+strField(vmResult, "instance_id"), ownerTaskID)
	}()
	logInfo("event=container_migrate_start_vm_auto tenant="+companyID+" workspace="+workspaceID+
		" owner_task="+ownerTaskID+" forbid_instance="+forbidInstanceID+
		" event_id="+eventID+" container_image_id="+strField(body, "container_image_id"), ownerTaskID)
	return "pending-start-" + eventID, nil
}

// publishContainerMigrateAwaitReady asks taskEvents to poll CSC until Running + server_url (heartbeat).
func publishContainerMigrateAwaitReady(companyID, workspaceID, taskID, instanceID, via, imageID, imageURL string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	data := map[string]interface{}{
		"company_id":         companyID,
		"tenant_id":          companyID,
		"workspace_id":       workspaceID,
		"task_id":            taskID,
		"instance_id":        instanceID,
		"via":                via,
		"container_image_id": imageID,
		"attempt":            1,
		"event_id":           fmt.Sprintf("migrate-await-%s-%d", taskID, time.Now().UnixNano()),
	}
	if imageURL != "" {
		data["container_image_url"] = imageURL
	}
	if err := publishDomainEvent(context.Background(), "CONTAINER_MIGRATE_AWAIT_READY", data, taskID); err != nil {
		logInfo("event=container_migrate_await_publish_failed tenant="+companyID+
			" task="+taskID+" err="+err.Error(), taskID)
		return
	}
	logInfo("event=container_migrate_await_published tenant="+companyID+" workspace="+workspaceID+
		" task="+taskID+" instance_id="+instanceID+" via="+via, taskID)
}

// startMigratedContainer asks container gateway to (re)start the image container on the owner's machine.
func startMigratedContainer(companyID, workspaceID, ownerTaskID, containerImageID, containerImageURL string) error {
	base := strings.TrimRight(strings.TrimSpace(cfg.ContainerGatewayURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8014"
	}
	// relay 为唯一容器启动路径（mock-run 已移除）。
	paths := []string{
		fmt.Sprintf("/api/tenant/%s/workspace/%s/task/%s/cloud/compute/relay-to-trae/start/",
			companyID, workspaceID, ownerTaskID),
	}
	payload := map[string]interface{}{
		"tenant_id":    companyID,
		"workspace_id": workspaceID,
		"task_id":      ownerTaskID,
	}
	if containerImageID != "" {
		payload["installed_image_id"] = containerImageID
		payload["container_image_id"] = containerImageID
	}
	if containerImageURL != "" {
		payload["image"] = containerImageURL
		payload["container_image_url"] = containerImageURL
	}
	raw, _ := json.Marshal(payload)
	var lastErr error
	for _, p := range paths {
		req, err := http.NewRequest(http.MethodPost, base+p, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
			req.Header.Set("X-Internal-Secret", sec)
		}
		if sec := strings.TrimSpace(cfg.ContainerGatewayInternalSecret); sec != "" {
			req.Header.Set("X-TaskContainerGateway-Internal-Secret", sec)
		}
		client := &http.Client{Timeout: 20 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 300 || resp.StatusCode == http.StatusAccepted {
			logInfo("event=container_migrate_start_triggered tenant="+companyID+
				" workspace="+workspaceID+" owner_task="+ownerTaskID+
				" path="+p+" status="+fmt.Sprint(resp.StatusCode), ownerTaskID)
			return nil
		}
		lastErr = fmt.Errorf("gateway start status=%d body=%s", resp.StatusCode, string(body))
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no gateway start path succeeded")
	}
	return lastErr
}
