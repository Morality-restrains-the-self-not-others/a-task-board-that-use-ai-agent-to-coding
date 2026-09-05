package aliyun

import (
	"fmt"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
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
	cfg.Endpoint = dara.String(aliyunECSEndpointHost(regionID))
	cfg.NoProxy = dara.String(aliyunECSNoProxyHosts(regionID))
	empty := ""
	cfg.HttpProxy = &empty
	cfg.HttpsProxy = &empty
	cfg.Socks5Proxy = &empty
}

func newECSClient(accessKey, secretKey, region string) (*ecs.Client, error) {
	cfg := &openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(region),
	}
	applyAliyunECSNetwork(cfg, region)
	return ecs.NewClient(cfg)
}
