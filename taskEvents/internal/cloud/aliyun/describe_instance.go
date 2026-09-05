package aliyun

import (
	"context"
	"fmt"
	"strings"
	"time"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

const (
	publicIPPollAttempts = 15
	publicIPPollInterval = 2 * time.Second
)

// WaitInstancePublicIP polls DescribeInstances until a public IP appears or attempts exhausted.
func WaitInstancePublicIP(ctx context.Context, accessKey, secretKey, region, instanceID string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	if region == "" || instanceID == "" {
		return "", fmt.Errorf("region and instance_id required")
	}
	client, err := newECSClient(accessKey, secretKey, region)
	if err != nil {
		return "", err
	}
	var lastErr error
	for i := 0; i < publicIPPollAttempts; i++ {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}
		ip, err := describeInstancePublicIP(client, region, instanceID)
		if err != nil {
			lastErr = err
		} else if ip != "" {
			return ip, nil
		}
		time.Sleep(publicIPPollInterval)
	}
	if lastErr != nil {
		return "", fmt.Errorf("public ip not ready for %s: %w", instanceID, lastErr)
	}
	return "", fmt.Errorf("public ip not ready for %s after %d attempts", instanceID, publicIPPollAttempts)
}

func describeInstancePublicIP(client *ecs.Client, region, instanceID string) (string, error) {
	req := &ecs.DescribeInstancesRequest{
		RegionId:    strPtr(region),
		InstanceIds: strPtr(fmt.Sprintf(`["%s"]`, instanceID)),
	}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil || resp.Body.Instances == nil {
		return "", nil
	}
	for _, inst := range resp.Body.Instances.Instance {
		if inst == nil {
			continue
		}
		if ip := firstPublicIPFromInstance(inst); ip != "" {
			return ip, nil
		}
	}
	return "", nil
}

func firstPublicIPFromInstance(inst *ecs.DescribeInstancesResponseBodyInstancesInstance) string {
	if inst == nil {
		return ""
	}
	if inst.PublicIpAddress != nil {
		for _, ip := range inst.PublicIpAddress.IpAddress {
			if ip != nil {
				if s := strings.TrimSpace(*ip); s != "" {
					return s
				}
			}
		}
	}
	if inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		if s := strings.TrimSpace(*inst.EipAddress.IpAddress); s != "" {
			return s
		}
	}
	return ""
}
