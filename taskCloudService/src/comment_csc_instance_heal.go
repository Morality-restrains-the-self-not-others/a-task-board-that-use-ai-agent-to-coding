package main

import (
	"fmt"
	"strings"
)

func commentCSCInstanceOwnedByOther(companyID, workspaceID, taskID, commentID, instanceID string) bool {
	if db == nil || trim(instanceID) == "" {
		return false
	}
	q := `SELECT comment_id FROM cloud_server_configs
		WHERE instance_id=? AND TRIM(COALESCE(comment_id,''))!='' AND comment_id!=?`
	args := []interface{}{trim(instanceID), trim(commentID)}
	if trim(companyID) != "" {
		q += ` AND company_id=?`
		args = append(args, trim(companyID))
	}
	if trim(workspaceID) != "" {
		q += ` AND workspace_id=?`
		args = append(args, trim(workspaceID))
	}
	if trim(taskID) != "" {
		q += ` AND task_id=?`
		args = append(args, trim(taskID))
	}
	q += ` LIMIT 1`
	var owner string
	err := db.QueryRow(q, args...).Scan(&owner)
	return err == nil && trim(owner) != ""
}

// healCommentCSCInstanceFromCloud 评论行 instance 为空时按 InstanceName 找回 ECS 并回填。
// 不抢其它评论已占用的 instance；Released/Stopped 行不自愈。
func healCommentCSCInstanceFromCloud(cfg *CloudServerConfig, tenantID, workspaceID, taskID string) *CloudServerConfig {
	if cfg == nil || trim(cfg.InstanceID) != "" || trim(cfg.CommentID) == "" {
		return cfg
	}
	switch strings.ToLower(normalizeMachineRuntimeStatus(cfg.LastRuntimeStatus)) {
	case "released", "terminated", "stopped", "stopping":
		return cfg
	}
	auth, err := resolveCloudAuthForRuntimeDescribe(tenantID, cfg)
	if err != nil || auth == nil {
		return cfg
	}
	region := trim(cfg.Region)
	if region == "" {
		return cfg
	}
	for _, name := range ecsInstanceNames(taskID, cfg.CommentID) {
		insts, _, descErr := describeInstancesByName(auth.SecretID, auth.SecretKey, region, name)
		if descErr != nil {
			logWarn(fmt.Sprintf("event=comment_csc_instance_heal_describe_failed task_id=%s comment_id=%s name=%s err=%s",
				taskID, cfg.CommentID, name, descErr.Error()), taskID)
			continue
		}
		for _, inst := range insts {
			id := trim(fmt.Sprintf("%v", inst["InstanceId"]))
			if id == "" || id == "<nil>" || strings.HasPrefix(id, "mock-") {
				continue
			}
			if commentCSCInstanceOwnedByOther(tenantID, workspaceID, taskID, cfg.CommentID, id) {
				logInfo(fmt.Sprintf("event=comment_csc_instance_heal_skip_owned task_id=%s comment_id=%s instance_id=%s",
					taskID, cfg.CommentID, id), taskID)
				continue
			}
			if bindErr := persistStartVmInstanceBinding(map[string]interface{}{
				"task_id": taskID, "comment_id": cfg.CommentID, "csc_id": cfg.ID,
			}, id, ""); bindErr != nil {
				logWarn(fmt.Sprintf("event=comment_csc_instance_heal_persist_failed task_id=%s comment_id=%s instance_id=%s err=%s",
					taskID, cfg.CommentID, id, bindErr.Error()), taskID)
				return cfg
			}
			cfg.InstanceID = id
			if trim(cfg.LastRuntimeStatus) == "" {
				cfg.LastRuntimeStatus = machineRuntimeStarting
			}
			logInfo(fmt.Sprintf("event=comment_csc_runtime_instance_bound task_id=%s comment_id=%s csc_id=%s instance_id=%s",
				taskID, cfg.CommentID, cfg.ID, id), taskID)
			return cfg
		}
	}
	return cfg
}

func promoteCommentCSCRunningOnBootEvidence(cfg *CloudServerConfig) {
	if cfg == nil || trim(cfg.InstanceID) == "" || trim(cfg.CommentID) == "" {
		return
	}
	switch strings.ToLower(normalizeMachineRuntimeStatus(cfg.LastRuntimeStatus)) {
	case "running", "released", "terminated", "stopped", "stopping":
		return
	}
	if db == nil {
		return
	}
	if _, err := db.Exec(
		`UPDATE cloud_server_configs SET last_runtime_status=?, updated_at=CURRENT_TIMESTAMP
		 WHERE company_id=? AND task_id=? AND comment_id=? AND instance_id=?`,
		machineRuntimeRunning, cfg.CompanyID, cfg.TaskID, cfg.CommentID, cfg.InstanceID,
	); err != nil {
		logWarn(fmt.Sprintf("event=comment_csc_boot_promote_failed task_id=%s comment_id=%s err=%s",
			cfg.TaskID, cfg.CommentID, err.Error()), cfg.TaskID)
		return
	}
	cfg.LastRuntimeStatus = machineRuntimeRunning
	logInfo(fmt.Sprintf("event=comment_csc_boot_promoted_running task_id=%s comment_id=%s instance_id=%s",
		cfg.TaskID, cfg.CommentID, cfg.InstanceID), cfg.TaskID)
}
