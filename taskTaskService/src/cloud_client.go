package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskTaskService/src/domain"
	"tracelog"
)

var cloudHTTP = &http.Client{Timeout: 30 * time.Second}

// startVMFn is replaceable in tests.
var startVMFn = startVM

// stopVMFn is replaceable in tests.
var stopVMFn = stopVM

// notifyContainerClosingSoonFn is replaceable in tests.
var notifyContainerClosingSoonFn = notifyContainerClosingSoon

// shutdownContainerLifecycleFn is replaceable in tests.
var shutdownContainerLifecycleFn = shutdownContainerLifecycle

// startVM posts start-vm / start-vm-auto to taskCloudService.
func startVM(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error {
	base := strings.TrimSpace(cfg.CloudServiceURL)
	if base == "" {
		return fmt.Errorf("cloud service url not configured")
	}
	apiPath = strings.Trim(apiPath, "/")
	if apiPath == "" {
		return fmt.Errorf("start-vm api path required")
	}
	u := fmt.Sprintf(
		"%s/api/cloud/compute/%s/tenant_id/%s/workspace_id/%s/",
		strings.TrimRight(base, "/"), apiPath, tenantID, workspaceID,
	)
	if body == nil {
		body = map[string]interface{}{}
	}
	// 镜像调用人员（评论人员）写入 CSC，供审计与回收筛选；不再用于跨任务闲置复用。
	if userID != "" {
		if _, ok := body["image_invoker_user_id"]; !ok {
			body["image_invoker_user_id"] = userID
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Auth-User-Id", userID)
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud %s HTTP %d: %s", apiPath, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// stopVM posts compute/stop-vm to taskCloudService (workspace-scoped).
func stopVM(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
	if body == nil {
		body = map[string]interface{}{}
	}
	if strings.TrimSpace(strField(body, "task_id")) == "" {
		body["task_id"] = taskID
	}
	return cloudComputePost(ctx, tenantID, workspaceID, "", "stop-vm", "", body)
}

func notifyContainerClosingSoon(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
	return cloudComputePost(ctx, tenantID, workspaceID, taskID, "container-task-lifecycle-closing-soon", "", body)
}

func shutdownContainerLifecycle(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
	return cloudComputePost(ctx, tenantID, workspaceID, taskID, "container-task-lifecycle-shutdown", "", body)
}

// cloudComputePost posts to workspace-scoped or task-scoped cloud compute paths.
// When taskID is non-empty, uses /task/{taskID}/cloud/compute/{action}/.
func cloudComputePost(ctx context.Context, tenantID, workspaceID, taskID, action, userID string, body map[string]interface{}) error {
	base := strings.TrimSpace(cfg.CloudServiceURL)
	if base == "" {
		return fmt.Errorf("cloud service url not configured")
	}
	action = strings.Trim(action, "/")
	if action == "" {
		return fmt.Errorf("cloud compute action required")
	}
	var u string
	if strings.TrimSpace(taskID) != "" {
		u = fmt.Sprintf(
			"%s/api/cloud/compute/%s/tenant_id/%s/workspace_id/%s/task_id/%s/",
			strings.TrimRight(base, "/"), action, tenantID, workspaceID, taskID,
		)
	} else {
		u = fmt.Sprintf(
			"%s/api/cloud/compute/%s/tenant_id/%s/workspace_id/%s/",
			strings.TrimRight(base, "/"), action, tenantID, workspaceID,
		)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Auth-User-Id", userID)
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud %s HTTP %d: %s", action, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// ErrInstalledImageNotFound is returned when Cloud lookup yields 404.
var ErrInstalledImageNotFound = errors.New("installed image not found")

// firstNonEmpty returns the first non-empty string in vs.
func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// InstalledImageLookup is the tenant-scoped installed image snapshot from Cloud.
type InstalledImageLookup struct {
	ID              string
	Name            string
	Version         string
	ExternalImageID string
	ImageSkills     domain.ImageSkillList
}

// lookupInstalledImageFn is replaceable in tests.
var lookupInstalledImageFn = lookupInstalledImage

// lookupInstalledImage fetches a tenant-scoped installed image snapshot from Cloud.
// imageNames 可选：调用方持有镜像名（技能快照 image_name / @提及 name）时传入，
// Cloud 端在 id= 命中失败后按租户内唯一名回退，解决陈旧 installed_image_id 仍
// 能解析到现网 ID 的问题（OPT-20260827-035）。
func lookupInstalledImage(tenantID, imageID string, imageNames ...string) (*InstalledImageLookup, error) {
	base := strings.TrimSpace(cfg.CloudServiceURL)
	if base == "" {
		return nil, fmt.Errorf("cloud service url not configured")
	}
	tenantID = strings.TrimSpace(tenantID)
	imageID = strings.TrimSpace(imageID)
	if tenantID == "" || imageID == "" {
		return nil, ErrInstalledImageNotFound
	}
	q := url.Values{}
	q.Set("tenant_id", tenantID)
	q.Set("id", imageID)
	if imageName := strings.TrimSpace(firstNonEmpty(imageNames...)); imageName != "" {
		q.Set("name", imageName)
	}
	u := strings.TrimRight(base, "/") + "/api/internal/tenant-installed-images/lookup?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrInstalledImageNotFound
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloud lookup status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("cloud lookup decode: %w", err)
		}
	}
	id := strings.TrimSpace(fmt.Sprintf("%v", out["id"]))
	if id == "" || id == "<nil>" {
		id = imageID
	}
	name := strings.TrimSpace(fmt.Sprintf("%v", out["name"]))
	if name == "<nil>" {
		name = ""
	}
	version := strings.TrimSpace(fmt.Sprintf("%v", out["version"]))
	if version == "<nil>" {
		version = ""
	}
	extID := strings.TrimSpace(fmt.Sprintf("%v", out["external_image_id"]))
	if extID == "<nil>" {
		extID = ""
	}
	return &InstalledImageLookup{
		ID: id, Name: name, Version: version, ExternalImageID: extID,
		ImageSkills: domain.DecodeImageSkillsJSON(out["image_skills"]),
	}, nil
}
