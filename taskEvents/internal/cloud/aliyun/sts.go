package aliyun

import (
	"fmt"
	"os"

	openapiv1 "github.com/alibabacloud-go/darabonba-openapi/client"
	sts "github.com/alibabacloud-go/sts-20150401/client"
)

// GetIAMID calls STS GetCallerIdentity (aligned with cloud/providers/aliyun/base.py).
func GetIAMID(accessKey, secretKey, region string) (string, error) {
	if region == "" {
		region = "cn-hangzhou"
	}
	cfg := &openapiv1.Config{
		AccessKeyId:     &accessKey,
		AccessKeySecret: &secretKey,
		RegionId:        &region,
		Endpoint:        strPtr(fmt.Sprintf("sts.%s.aliyuncs.com", region)),
	}
	client, err := sts.NewClient(cfg)
	if err != nil {
		return "", err
	}
	resp, err := client.GetCallerIdentity()
	if err != nil {
		return "", err
	}
	if resp.Body == nil {
		return "", fmt.Errorf("STS empty response body")
	}
	if resp.Body.UserId != nil && *resp.Body.UserId != "" {
		return *resp.Body.UserId, nil
	}
	if resp.Body.PrincipalId != nil && *resp.Body.PrincipalId != "" {
		return *resp.Body.PrincipalId, nil
	}
	return "", fmt.Errorf("no iam id in STS response")
}

func strPtr(s string) *string { return &s }

// ECSModulePath returns monorepo ECS Go SDK path hint.
func ECSModulePath() string {
	if p := os.Getenv("SDK_ECS_GO"); p != "" {
		return p
	}
	return "../sdk/ecs-20140526"
}
