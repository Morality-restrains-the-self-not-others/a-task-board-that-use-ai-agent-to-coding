package main

import (
	"encoding/json"
	"fmt"
	"strings"

	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func aliyunDescribeInstanceAttribute(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, string, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	regionID = strings.TrimSpace(regionID)
	instanceID = strings.TrimSpace(instanceID)
	if accessKey == "" || secretKey == "" || regionID == "" || instanceID == "" {
		return nil, "", fmt.Errorf("missing aliyun describe parameters")
	}
	client, err := ecsclient.NewClient(&openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(regionID),
	})
	if err != nil {
		return nil, "", err
	}
	req := &ecsclient.DescribeInstanceAttributeRequest{
		InstanceId: dara.String(instanceID),
	}
	resp, err := client.DescribeInstanceAttribute(req)
	if err != nil {
		return nil, "", err
	}
	requestID := ""
	if resp != nil && resp.Body != nil && resp.Body.RequestId != nil {
		requestID = strings.TrimSpace(*resp.Body.RequestId)
	}
	out := map[string]interface{}{}
	if resp != nil && resp.Body != nil {
		body := resp.Body
		if body.Status != nil {
			out["Status"] = strings.TrimSpace(*body.Status)
		}
		if body.InstanceId != nil {
			out["InstanceId"] = strings.TrimSpace(*body.InstanceId)
		}
		if body.RegionId != nil {
			out["RegionId"] = strings.TrimSpace(*body.RegionId)
		}
		if body.InstanceType != nil {
			out["InstanceType"] = strings.TrimSpace(*body.InstanceType)
		}
		if body.PublicIpAddress != nil && body.PublicIpAddress.IpAddress != nil {
			ips := []string{}
			for _, ip := range body.PublicIpAddress.IpAddress {
				if ip != nil && strings.TrimSpace(*ip) != "" {
					ips = append(ips, strings.TrimSpace(*ip))
				}
			}
			out["PublicIpAddress"] = map[string]interface{}{"IpAddress": ips}
		}
	}
	return out, requestID, nil
}

// describeInstanceForRuntimeFn looks up a single ECS instance for server-runtime-status.
// found=false means the cloud API returned no instance (treated as released).
type describeInstanceForRuntimeFn func(accessKey, secretKey, regionID, instanceID string) (attr map[string]interface{}, found bool, requestID string, err error)

// describeInstanceForRuntime is overridable in tests.
var describeInstanceForRuntime describeInstanceForRuntimeFn = aliyunDescribeInstanceForRuntime

// aliyunDescribeInstanceForRuntime uses DescribeInstances so AutoReleaseTime and
// other UI fields are available (DescribeInstanceAttribute omits AutoReleaseTime).
func aliyunDescribeInstanceForRuntime(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	regionID = strings.TrimSpace(regionID)
	instanceID = strings.TrimSpace(instanceID)
	if accessKey == "" || secretKey == "" || regionID == "" || instanceID == "" {
		return nil, false, "", fmt.Errorf("missing aliyun describe parameters")
	}
	client, err := ecsclient.NewClient(&openapiutil.Config{
		AccessKeyId:     dara.String(accessKey),
		AccessKeySecret: dara.String(secretKey),
		RegionId:        dara.String(regionID),
	})
	if err != nil {
		return nil, false, "", err
	}
	idsJSON, err := json.Marshal([]string{instanceID})
	if err != nil {
		return nil, false, "", err
	}
	req := &ecsclient.DescribeInstancesRequest{
		RegionId:    dara.String(regionID),
		InstanceIds: dara.String(string(idsJSON)),
		PageNumber:  dara.Int32(1),
		PageSize:    dara.Int32(1),
	}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, false, "", err
	}
	requestID := ""
	if resp != nil && resp.Body != nil && resp.Body.RequestId != nil {
		requestID = strings.TrimSpace(*resp.Body.RequestId)
	}
	if resp == nil || resp.Body == nil || resp.Body.Instances == nil || len(resp.Body.Instances.Instance) == 0 {
		return nil, false, requestID, nil
	}
	inst := resp.Body.Instances.Instance[0]
	if inst == nil {
		return nil, false, requestID, nil
	}
	return mapDescribeInstancesInstance(inst), true, requestID, nil
}

func mapDescribeInstancesInstance(inst *ecsclient.DescribeInstancesResponseBodyInstancesInstance) map[string]interface{} {
	out := map[string]interface{}{}
	if inst == nil {
		return out
	}
	if inst.Status != nil {
		out["Status"] = strings.TrimSpace(*inst.Status)
	}
	if inst.InstanceId != nil {
		out["InstanceId"] = strings.TrimSpace(*inst.InstanceId)
	}
	if inst.RegionId != nil {
		out["RegionId"] = strings.TrimSpace(*inst.RegionId)
	}
	if inst.ZoneId != nil {
		out["ZoneId"] = strings.TrimSpace(*inst.ZoneId)
	}
	if inst.InstanceName != nil {
		out["InstanceName"] = strings.TrimSpace(*inst.InstanceName)
	}
	if inst.InstanceType != nil {
		out["InstanceType"] = strings.TrimSpace(*inst.InstanceType)
	}
	if inst.ImageId != nil {
		out["ImageId"] = strings.TrimSpace(*inst.ImageId)
	}
	if inst.CreationTime != nil {
		out["CreationTime"] = strings.TrimSpace(*inst.CreationTime)
	}
	if inst.AutoReleaseTime != nil {
		// Empty string means auto-release is not configured.
		if t := strings.TrimSpace(*inst.AutoReleaseTime); t != "" {
			out["AutoReleaseTime"] = t
		}
	}
	if inst.Cpu != nil {
		out["Cpu"] = *inst.Cpu
	}
	if inst.Memory != nil {
		out["Memory"] = *inst.Memory
	}
	if inst.InternetChargeType != nil {
		if t := strings.TrimSpace(*inst.InternetChargeType); t != "" {
			out["InternetChargeType"] = t
		}
	}
	if inst.InternetMaxBandwidthOut != nil {
		out["InternetMaxBandwidthOut"] = *inst.InternetMaxBandwidthOut
	}
	if inst.InternetMaxBandwidthIn != nil {
		out["InternetMaxBandwidthIn"] = *inst.InternetMaxBandwidthIn
	}
	if eip := mapDescribeInstancesEipAddress(inst.EipAddress); len(eip) > 0 {
		out["EipAddress"] = eip
	}
	if inst.PublicIpAddress != nil && inst.PublicIpAddress.IpAddress != nil {
		ips := make([]string, 0, len(inst.PublicIpAddress.IpAddress))
		for _, ip := range inst.PublicIpAddress.IpAddress {
			if ip != nil && strings.TrimSpace(*ip) != "" {
				ips = append(ips, strings.TrimSpace(*ip))
			}
		}
		out["PublicIpAddress"] = map[string]interface{}{"IpAddress": ips}
	}
	if inst.SecurityGroupIds != nil && inst.SecurityGroupIds.SecurityGroupId != nil {
		sgs := make([]string, 0, len(inst.SecurityGroupIds.SecurityGroupId))
		for _, sg := range inst.SecurityGroupIds.SecurityGroupId {
			if sg != nil && strings.TrimSpace(*sg) != "" {
				sgs = append(sgs, strings.TrimSpace(*sg))
			}
		}
		out["SecurityGroupIds"] = map[string]interface{}{"SecurityGroupId": sgs}
	}
	if inst.VpcAttributes != nil {
		vpc := map[string]interface{}{}
		if inst.VpcAttributes.VpcId != nil {
			vpc["VpcId"] = strings.TrimSpace(*inst.VpcAttributes.VpcId)
		}
		if inst.VpcAttributes.VSwitchId != nil {
			vpc["VSwitchId"] = strings.TrimSpace(*inst.VpcAttributes.VSwitchId)
		}
		if inst.VpcAttributes.PrivateIpAddress != nil && inst.VpcAttributes.PrivateIpAddress.IpAddress != nil {
			ips := make([]string, 0, len(inst.VpcAttributes.PrivateIpAddress.IpAddress))
			for _, ip := range inst.VpcAttributes.PrivateIpAddress.IpAddress {
				if ip != nil && strings.TrimSpace(*ip) != "" {
					ips = append(ips, strings.TrimSpace(*ip))
				}
			}
			vpc["PrivateIpAddress"] = map[string]interface{}{"IpAddress": ips}
		}
		out["VpcAttributes"] = vpc
	}
	return out
}

func mapDescribeInstancesEipAddress(eip *ecsclient.DescribeInstancesResponseBodyInstancesInstanceEipAddress) map[string]interface{} {
	if eip == nil {
		return nil
	}
	out := map[string]interface{}{}
	if eip.IpAddress != nil {
		if ip := strings.TrimSpace(*eip.IpAddress); ip != "" {
			out["IpAddress"] = ip
		}
	}
	if eip.InternetChargeType != nil {
		if t := strings.TrimSpace(*eip.InternetChargeType); t != "" {
			out["InternetChargeType"] = t
		}
	}
	if eip.Bandwidth != nil {
		out["Bandwidth"] = *eip.Bandwidth
	}
	return out
}

// publicIPFromDescribeAttr 从 mapDescribeInstancesInstance 产物提取公网 IP。
func publicIPFromDescribeAttr(attr map[string]interface{}) string {
	if attr == nil {
		return ""
	}
	if m, ok := attr["PublicIpAddress"].(map[string]interface{}); ok {
		if ips, ok := m["IpAddress"].([]string); ok {
			for _, ip := range ips {
				if v := strings.TrimSpace(ip); v != "" {
					return v
				}
			}
		}
		if raw, ok := m["IpAddress"].([]interface{}); ok {
			for _, item := range raw {
				if v := strings.TrimSpace(fmt.Sprint(item)); v != "" && v != "<nil>" {
					return v
				}
			}
		}
	}
	if m, ok := attr["EipAddress"].(map[string]interface{}); ok {
		if v := strings.TrimSpace(fmt.Sprint(m["IpAddress"])); v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func aliyunErrorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, code := range []string{"InvalidInstanceId.NotFound", "InvalidInstanceId"} {
		if strings.Contains(msg, code) {
			return code
		}
	}
	return ""
}
