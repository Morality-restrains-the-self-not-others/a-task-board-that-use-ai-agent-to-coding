package main

import (
	"fmt"
	"strings"

	openapiv1 "github.com/alibabacloud-go/darabonba-openapi/client"
	stsclient "github.com/alibabacloud-go/sts-20150401/client"
	util "github.com/alibabacloud-go/tea-utils/service"
)

func verifyAccessKeyCallerIdentity(platformType, secretID, secretKey string) map[string]interface{} {
	if platformType != "aliyun" {
		return map[string]interface{}{
			"success": false,
			"detail":  fmt.Sprintf("云平台「%s」暂不支持 Access Key 凭据校验", platformType),
		}
	}
	if useInMemoryCloud() {
		return map[string]interface{}{
			"success": true,
			"caller_identity": map[string]string{
				"account_id": "mock-account",
				"user_id":    "mock-user",
				"arn":        "acs:ram::mock:user/mock",
				"request_id": "mock-request-id",
			},
		}
	}
	return fetchAliyunCallerIdentity(secretID, secretKey, "cn-hangzhou")
}

func fetchAliyunCallerIdentity(accessKey, secretKey, regionID string) map[string]interface{} {
	if regionID == "" {
		regionID = "cn-hangzhou"
	}
	cfg := &openapiv1.Config{
		AccessKeyId:     &accessKey,
		AccessKeySecret: &secretKey,
		RegionId:        &regionID,
	}
	applyAliyunSTSNetwork(cfg, regionID)
	client, err := stsclient.NewClient(cfg)
	if err != nil {
		return map[string]interface{}{"success": false, "detail": err.Error()}
	}
	resp, err := client.GetCallerIdentity()
	if err != nil {
		return map[string]interface{}{"success": false, "detail": err.Error()}
	}
	if resp.Body == nil {
		return map[string]interface{}{"success": false, "detail": "STS 返回空响应体"}
	}
	identity := map[string]string{}
	if resp.Body.AccountId != nil && *resp.Body.AccountId != "" {
		identity["account_id"] = *resp.Body.AccountId
	}
	if resp.Body.Arn != nil && *resp.Body.Arn != "" {
		identity["arn"] = *resp.Body.Arn
	}
	if resp.Body.IdentityType != nil && *resp.Body.IdentityType != "" {
		identity["identity_type"] = *resp.Body.IdentityType
	}
	if resp.Body.PrincipalId != nil && *resp.Body.PrincipalId != "" {
		identity["principal_id"] = *resp.Body.PrincipalId
	}
	if resp.Body.UserId != nil && *resp.Body.UserId != "" {
		identity["user_id"] = *resp.Body.UserId
	}
	if resp.Body.RoleId != nil && *resp.Body.RoleId != "" {
		identity["role_id"] = *resp.Body.RoleId
	}
	if resp.Body.RequestId != nil && *resp.Body.RequestId != "" {
		identity["request_id"] = *resp.Body.RequestId
	}
	return map[string]interface{}{"success": true, "caller_identity": identity}
}

func strPtr(s string) *string { return &s }

func int64Ptr(v int64) *int64 { return &v }

var stsAssumeRoleAPI = liveAssumeRoleReleaseSTS

func mintAliyunReleaseSTS(in stsReleaseMintInput) (map[string]any, error) {
	if strings.TrimSpace(in.RoleARN) == "" {
		return nil, fmt.Errorf("role arn required")
	}
	if useInMemoryCloud() {
		return map[string]any{
			"access_key_id":     "STS.mock",
			"access_key_secret": "mock",
			"security_token":    "mock",
			"expiration":        "2099-01-01T00:00:00Z",
		}, nil
	}
	return stsAssumeRoleAPI(in)
}

func stsReleaseSessionName(instanceID string) string {
	raw := "idleRel-" + strings.TrimSpace(instanceID)
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '@' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 64 {
		out = out[:64]
	}
	if len(out) < 2 {
		return "idleRel"
	}
	return out
}

func aliyunSTSDirectRuntime() *util.RuntimeOptions {
	empty := ""
	return &util.RuntimeOptions{
		HttpProxy:   &empty,
		HttpsProxy:  &empty,
		Socks5Proxy: &empty,
		NoProxy:     strPtr(".aliyuncs.com,sts.aliyuncs.com"),
	}
}

// liveAssumeRoleReleaseSTS calls STS AssumeRole (never GetSessionToken).
// https://help.aliyun.com/zh/ram/developer-reference/api-sts-2015-04-01-assumerole
func liveAssumeRoleReleaseSTS(in stsReleaseMintInput) (map[string]any, error) {
	regionID := strings.TrimSpace(in.Region)
	if regionID == "" {
		regionID = "cn-hangzhou"
	}
	cfg := &openapiv1.Config{
		AccessKeyId:     &in.AccessKey,
		AccessKeySecret: &in.SecretKey,
		RegionId:        &regionID,
	}
	applyAliyunSTSNetwork(cfg, regionID)
	client, err := stsclient.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	req := &stsclient.AssumeRoleRequest{
		DurationSeconds: int64Ptr(in.DurationSec),
		Policy:          strPtr(in.SessionPolicy),
		RoleArn:         strPtr(in.RoleARN),
		RoleSessionName: strPtr(stsReleaseSessionName(in.InstanceID)),
	}
	resp, err := client.AssumeRoleWithOptions(req, aliyunSTSDirectRuntime())
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Credentials == nil {
		return nil, fmt.Errorf("STS AssumeRole returned empty credentials")
	}
	cred := resp.Body.Credentials
	out := map[string]any{}
	if cred.AccessKeyId != nil {
		out["access_key_id"] = *cred.AccessKeyId
	}
	if cred.AccessKeySecret != nil {
		out["access_key_secret"] = *cred.AccessKeySecret
	}
	if cred.SecurityToken != nil {
		out["security_token"] = *cred.SecurityToken
	}
	if cred.Expiration != nil {
		out["expiration"] = *cred.Expiration
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("STS AssumeRole credentials missing fields")
	}
	return out, nil
}
