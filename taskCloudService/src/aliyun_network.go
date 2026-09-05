package main

import (
	"fmt"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiv1 "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/dara"
)

func aliyunECSEndpointHost(regionID string) string {
	regionID = strings.TrimSpace(regionID)
	if regionID == "" {
		regionID = "cn-hangzhou"
	}
	return fmt.Sprintf("ecs.%s.aliyuncs.com", regionID)
}

func aliyunECSNoProxyHosts(regionID string) string {
	regionID = strings.TrimSpace(regionID)
	hosts := []string{
		"ecs-cn-hangzhou.aliyuncs.com",
		"ecs.aliyuncs.com",
		".aliyuncs.com",
	}
	if regionID != "" {
		hosts = append(hosts,
			fmt.Sprintf("ecs.%s.aliyuncs.com", regionID),
			fmt.Sprintf("ecs-%s.aliyuncs.com", regionID),
		)
	}
	return strings.Join(hosts, ",")
}

func applyAliyunECSNetwork(cfg *openapiutil.Config, regionID string) {
	if cfg == nil {
		return
	}
	endpoint := aliyunECSEndpointHost(regionID)
	cfg.Endpoint = dara.String(endpoint)
	noProxy := aliyunECSNoProxyHosts(regionID)
	cfg.NoProxy = dara.String(noProxy)
	empty := ""
	cfg.HttpProxy = &empty
	cfg.HttpsProxy = &empty
	cfg.Socks5Proxy = &empty
}

func applyAliyunSTSNetwork(cfg *openapiv1.Config, regionID string) {
	if cfg == nil {
		return
	}
	regionID = strings.TrimSpace(regionID)
	if regionID == "" {
		regionID = "cn-hangzhou"
	}
	endpoint := fmt.Sprintf("sts.%s.aliyuncs.com", regionID)
	cfg.Endpoint = strPtr(endpoint)
	noProxy := strings.Join([]string{
		endpoint,
		fmt.Sprintf("sts-%s.aliyuncs.com", regionID),
		".aliyuncs.com",
	}, ",")
	cfg.NoProxy = strPtr(noProxy)
	empty := ""
	cfg.HttpProxy = &empty
	cfg.HttpsProxy = &empty
	cfg.Socks5Proxy = &empty
}
