package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"snowflake"
	"strings"

	"tracelog"
)

func credentialServiceBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(cfg.CredentialServiceURL), "/")
	if base == "" {
		return "http://127.0.0.1:8015"
	}
	return base
}

func upsertBootstrapCloudServerConfig(
	companyID, workspaceID, taskID, authorizationID, platformType, regionID, zoneID string,
) error {
	existing, err := loadCloudServerConfig(companyID, workspaceID, taskID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		existing = nil
	}
	row := CloudServerConfig{
		CompanyID:       companyID,
		WorkspaceID:     workspaceID,
		TaskID:          taskID,
		AuthorizationID: authorizationID,
		Platform:        platformType,
		Region:          regionID,
		ZoneID:          zoneID,
	}
	if existing != nil {
		row.ID = existing.ID
		row.InstanceID = existing.InstanceID
		row.SecurityGroupID = existing.SecurityGroupID
		row.VswitchID = existing.VswitchID
		row.PublicIP = existing.PublicIP
		row.ServerURL = existing.ServerURL
		row.BusinessAPIEndpoint = existing.BusinessAPIEndpoint
		row.ContainerVscodeURL = existing.ContainerVscodeURL
		row.ErrorReason = existing.ErrorReason
		row.LaunchRequestID = existing.LaunchRequestID
		row.ClientToken = existing.ClientToken
		row.LastRuntimeStatus = existing.LastRuntimeStatus
		row.ImageInvokerUserID = existing.ImageInvokerUserID
		row.CreatedAt = existing.CreatedAt
	} else {
		row.ID = snowflake.GenerateIDString()
		if row.Platform == "" {
			row.Platform = "aliyun"
		}
	}
	return upsertCloudServerConfig(row)
}

func bootstrapStartVmTokens(
	ctx context.Context,
	companyID, workspaceID, taskID, commentID, authorizationID, platformType, regionID, zoneID string,
) (accessToken string, err error) {
	companyID = trim(companyID)
	workspaceID = trim(workspaceID)
	taskID = trim(taskID)
	commentID = trim(commentID)
	authorizationID = trim(authorizationID)
	if companyID == "" || workspaceID == "" || taskID == "" {
		return "", fmt.Errorf("company_id, workspace_id and task_id required")
	}
	if commentID == "" {
		return "", fmt.Errorf("comment_id required")
	}
	if authorizationID == "" {
		return "", fmt.Errorf("authorization_id required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if err := upsertBootstrapCloudServerConfig(companyID, workspaceID, taskID, authorizationID, platformType, regionID, zoneID); err != nil {
		log.Printf("[taskCloudService] bootstrap upsert cloud_server_config failed: company_id=%s task_id=%s err=%v",
			companyID, taskID, err)
		return "", fmt.Errorf("upsert cloud_server_config: %w", err)
	}
	// 任务模板落真实 platform/region/auth 后，必须克隆评论级 CSC，否则 RunInstances
	// 成功也无法 persist instance_id（零行），runtime-status 会误报「未找到服务器配置记录」。
	if _, err := ensureCommentCloudServerConfig(companyID, workspaceID, taskID, commentID); err != nil {
		log.Printf("[taskCloudService] bootstrap ensure comment csc failed: company_id=%s task_id=%s comment_id=%s err=%v",
			companyID, taskID, commentID, err)
		return "", fmt.Errorf("ensure comment cloud_server_config: %w", err)
	}

	url := fmt.Sprintf(
		"%s/v1/token/init/tenant/%s/workspace/%s/task/%s/comment/%s",
		credentialServiceBaseURL(),
		companyID,
		workspaceID,
		taskID,
		commentID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		log.Printf("[taskCloudService] bootstrap token init request build failed: task_id=%s err=%v", taskID, err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)

	resp, err := credentialHTTP.Do(req)
	if err != nil {
		log.Printf("[taskCloudService] bootstrap token init unreachable: task_id=%s url=%s err=%v", taskID, url, err)
		return "", fmt.Errorf("token init unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskCloudService] bootstrap token init failed: task_id=%s status=%d body=%s",
			taskID, resp.StatusCode, string(raw))
		return "", fmt.Errorf("token init failed: status=%d", resp.StatusCode)
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		log.Printf("[taskCloudService] bootstrap token init invalid JSON: task_id=%s err=%v", taskID, err)
		return "", fmt.Errorf("token init invalid response: %w", err)
	}
	accessToken = strings.TrimSpace(parsed.AccessToken)
	if accessToken == "" {
		log.Printf("[taskCloudService] bootstrap token init empty access_token: task_id=%s", taskID)
		return "", fmt.Errorf("token init returned empty access_token")
	}

	log.Printf("[taskCloudService] bootstrap token init OK: company_id=%s task_id=%s comment_id=%s", companyID, taskID, commentID)
	return accessToken, nil
}
