package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type stsReleaseMintInput struct {
	AccessKey     string
	SecretKey     string
	Region        string
	RoleARN       string
	SessionPolicy string
	DurationSec   int64
	InstanceID    string
}

// stsReleaseMinter is injectable. Production default is mintAliyunReleaseSTS
// (AssumeRole + session policy). Tests stub to avoid live Aliyun.
// Official API: https://help.aliyun.com/zh/ram/developer-reference/api-sts-2015-04-01-assumerole
var stsReleaseMinter = mintAliyunReleaseSTS

func buildSTSReleaseSessionPolicy(instanceID string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instance_id required")
	}
	if strings.ContainsAny(instanceID, " \t\n\"'\\") {
		return "", fmt.Errorf("instance_id contains illegal characters")
	}
	doc := map[string]any{
		"Version": "1",
		"Statement": []map[string]any{
			{
				"Effect":   "Allow",
				"Action":   []string{"ecs:DeleteInstance", "ecs:DescribeInstances"},
				"Resource": []string{"acs:ecs:*:*:instance/" + instanceID},
			},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// stsReleaseDurationSeconds is TTL = min(3600, (idle_recycle_minutes+15)*60),
// clamped to Aliyun AssumeRole minimum 900s. minutes<=0 means do not mint.
func stsReleaseDurationSeconds(idleMinutes int) int64 {
	if idleMinutes <= 0 {
		return 0
	}
	sec := int64(idleMinutes+15) * 60
	if sec < 900 {
		sec = 900
	}
	if sec > 3600 {
		sec = 3600
	}
	return sec
}

func loadCPAStsReleaseRoleARN(authorizationID string) (string, error) {
	authorizationID = strings.TrimSpace(authorizationID)
	if authorizationID == "" || db == nil {
		return "", nil
	}
	var arn string
	err := db.QueryRow(
		`SELECT COALESCE(sts_release_role_arn,'') FROM cloud_platform_authorizations WHERE id=?`,
		authorizationID,
	).Scan(&arn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(arn), nil
}

func maybeMachineReleaseSTS(companyID, workspaceID, taskID, commentID string) map[string]any {
	if stsReleaseMinter == nil {
		return nil
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	cfg, err := loadCloudServerConfigForComment(companyID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		return nil
	}
	if strings.TrimSpace(strings.ToLower(cfg.Platform)) != "aliyun" {
		return nil
	}
	instanceID := strings.TrimSpace(cfg.InstanceID)
	if instanceID == "" {
		return nil
	}
	policy, err := loadWorkspaceMachinePolicy(companyID, workspaceID)
	if err != nil {
		return nil
	}
	duration := stsReleaseDurationSeconds(policy.IdleRecycleMinutes)
	if duration <= 0 {
		return nil
	}
	roleARN, err := loadCPAStsReleaseRoleARN(cfg.AuthorizationID)
	if err != nil || roleARN == "" {
		return nil
	}
	auth, err := loadCloudAuth(companyID, cfg.AuthorizationID)
	if err != nil || auth == nil {
		return nil
	}
	if strings.TrimSpace(auth.SecretID) == "" || strings.TrimSpace(auth.SecretKey) == "" {
		return nil
	}
	sessionPolicy, err := buildSTSReleaseSessionPolicy(instanceID)
	if err != nil {
		logInfo("event=sts_release_policy status=error instance="+instanceID+" err="+err.Error(), taskID)
		return nil
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "cn-hangzhou"
	}
	sts, err := stsReleaseMinter(stsReleaseMintInput{
		AccessKey:     auth.SecretID,
		SecretKey:     auth.SecretKey,
		Region:        region,
		RoleARN:       roleARN,
		SessionPolicy: sessionPolicy,
		DurationSec:   duration,
		InstanceID:    instanceID,
	})
	if err != nil || sts == nil {
		if err != nil {
			logInfo("event=sts_release_mint status=error err="+err.Error(), taskID)
		}
		return nil
	}
	return sts
}
