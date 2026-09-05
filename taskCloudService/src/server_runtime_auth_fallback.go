package main

import (
	"strings"
)

// resolveCloudAuthForRuntimeDescribe loads CSC authorization_id first; if missing/deleted,
// falls back to the tenant's first auth for the same platform so live Describe can still run.
func resolveCloudAuthForRuntimeDescribe(tenantID string, cfg *CloudServerConfig) (*cloudAuthRecord, error) {
	if cfg == nil {
		return nil, nil
	}
	authID := strings.TrimSpace(cfg.AuthorizationID)
	auth, err := loadCloudAuth(tenantID, authID)
	if err != nil {
		return nil, err
	}
	if auth != nil {
		return auth, nil
	}
	plat := strings.TrimSpace(cfg.Platform)
	if plat == "" || strings.EqualFold(plat, "mock") {
		if isMockMachineInstanceID(cfg.InstanceID) {
			return nil, nil
		}
		// 真实 ECS 卡在遗留 mock 平台标签：仍用租户授权做 Describe，而不是 auth_missing。
		return loadFirstCloudAuthForCompanyAnyPlatform(tenantID)
	}
	return loadFirstCloudAuthForCompany(tenantID, plat)
}

// buildAuthMissingRuntimeFallback returns a success payload from local CSC cache when
// cloud credentials are unavailable. Avoids status=error → FE 「未知」与「已启动」矛盾。
func buildAuthMissingRuntimeFallback(cfg *CloudServerConfig, instanceID string) map[string]interface{} {
	cached := ""
	publicIP := ""
	platform := ""
	region := ""
	if cfg != nil {
		cached = strings.TrimSpace(cfg.LastRuntimeStatus)
		publicIP = strings.TrimSpace(cfg.PublicIP)
		platform = strings.TrimSpace(cfg.Platform)
		region = strings.TrimSpace(cfg.Region)
		if cached == "" {
			if strings.TrimSpace(cfg.ServerURL) != "" ||
				publicIP != "" ||
				strings.TrimSpace(cfg.BusinessAPIEndpoint) != "" {
				cached = machineRuntimeRunning
			}
		}
	}
	resp := map[string]interface{}{
		"status":       "success",
		"instance_id":  instanceID,
		"platform":     platform,
		"region":       region,
		"auth_missing": true,
		"message":      "未找到云平台授权信息，已使用本地缓存的运行状态；请在工作区设置中配置并激活云平台授权以刷新实例详情",
	}
	if cached != "" {
		resp["runtime_status"] = cached
	} else {
		resp["runtime_status"] = nil
	}
	if publicIP != "" || cached != "" {
		body := map[string]interface{}{
			"InstanceId": instanceID,
		}
		if cached != "" {
			body["Status"] = cached
		}
		if publicIP != "" {
			body["PublicIpAddress"] = map[string]interface{}{
				"IpAddress": []string{publicIP},
			}
		}
		// 授权缺失时仍展示上次 Describe 缓存的带宽行（OPT-20260817-002）。
		if chargeType, bwOut := loadInstanceBandwidth(instanceID); chargeType != "" || bwOut > 0 {
			if chargeType != "" {
				body["InternetChargeType"] = chargeType
			}
			if bwOut > 0 {
				body["InternetMaxBandwidthOut"] = bwOut
			}
		}
		resp["instance_attribute"] = map[string]interface{}{"body": body}
	}
	return resp
}
