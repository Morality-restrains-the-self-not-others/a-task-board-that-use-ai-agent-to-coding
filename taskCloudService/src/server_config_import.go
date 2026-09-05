package main

import (
	"fmt"
	"strings"
	"time"
)

func stripTaskLevelRuntimeFields(c *CloudServerConfig) {
	if c == nil || strings.TrimSpace(c.CommentID) != "" {
		return
	}
	c.InstanceID = ""
	c.LastRuntimeStatus = ""
	c.PublicIP = ""
	c.ServerURL = ""
	c.BusinessAPIEndpoint = ""
	c.ContainerVscodeURL = ""
	c.LaunchRequestID = ""
}

func importCloudServerConfigs(rows []CloudServerConfig) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	count := 0
	for _, c := range rows {
		if c.ID == "" || c.CompanyID == "" || c.TaskID == "" {
			return count, fmt.Errorf("row %d: id, company_id, task_id required", count)
		}
		stripTaskLevelRuntimeFields(&c)
		createdAt := c.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		updatedAt := c.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
		_, err := tx.Exec(
			`INSERT INTO cloud_server_configs(
				id, company_id, workspace_id, task_id, comment_id, platform, instance_id,
				security_group_id, vswitch_id, region, zone_id, authorization_id,
				public_ip, server_url, business_api_endpoint, container_vscode_url,
				error_reason, launch_request_id, client_token, last_runtime_status,
				image_invoker_user_id, created_at, updated_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON DUPLICATE KEY UPDATE
				company_id=VALUES(company_id),
				workspace_id=VALUES(workspace_id),
				task_id=VALUES(task_id),
				comment_id=VALUES(comment_id),
				platform=VALUES(platform),`+cloudServerRuntimeUpsertPreserve+`
				security_group_id=VALUES(security_group_id),
				vswitch_id=VALUES(vswitch_id),
				region=VALUES(region),
				zone_id=VALUES(zone_id),
				authorization_id=VALUES(authorization_id),
				business_api_endpoint=VALUES(business_api_endpoint),
				container_vscode_url=VALUES(container_vscode_url),
				error_reason=VALUES(error_reason),
				launch_request_id=VALUES(launch_request_id),
				client_token=VALUES(client_token),
				image_invoker_user_id=CASE
					WHEN TRIM(COALESCE(VALUES(image_invoker_user_id),'')) != ''
					THEN VALUES(image_invoker_user_id)
					ELSE cloud_server_configs.image_invoker_user_id
				END,
				updated_at=VALUES(updated_at)`,
			c.ID, c.CompanyID, c.WorkspaceID, c.TaskID, c.CommentID, c.Platform, c.InstanceID,
			c.SecurityGroupID, c.VswitchID, c.Region, c.ZoneID, c.AuthorizationID,
			c.PublicIP, c.ServerURL, c.BusinessAPIEndpoint, c.ContainerVscodeURL,
			c.ErrorReason, c.LaunchRequestID, c.ClientToken, c.LastRuntimeStatus,
			c.ImageInvokerUserID, createdAt, updatedAt,
		)
		if err != nil {
			return count, err
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}
