package main

import (
	"fmt"
	"net"
	"strings"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

// ── CreateVpc ──────────────────────────────────────────────────────────────

func aliyunCreateVpc(accessKey, secretKey, regionID, vpcName, cidrBlock string) (string, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return "", "", err
	}
	if cidrBlock == "" {
		cidrBlock = "192.168.0.0/16" // 与 Django auto_create_resources 一致
	}
	resp, err := client.CreateVpc(&ecsclient.CreateVpcRequest{
		RegionId:  dara.String(regionID),
		VpcName:   dara.String(vpcName),
		CidrBlock: dara.String(cidrBlock),
	})
	if err != nil {
		return "", "", err
	}
	if resp.Body == nil {
		return "", "", fmt.Errorf("CreateVpc empty response")
	}
	return derefString(resp.Body.VpcId), cidrBlock, nil
}

// ── CreateVSwitch ───────────────────────────────────────────────────────────

func aliyunCreateVSwitch(accessKey, secretKey, regionID, zoneID, vpcID, switchName, cidrBlock string) (string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return "", err
	}
	if cidrBlock == "" {
		cidrBlock = pickNonConflictingVSwitchCIDR(vpcID)
	}
	resp, err := client.CreateVSwitch(&ecsclient.CreateVSwitchRequest{
		RegionId:    dara.String(regionID),
		ZoneId:      dara.String(zoneID),
		VpcId:       dara.String(vpcID),
		VSwitchName: dara.String(switchName),
		CidrBlock:   dara.String(cidrBlock),
	})
	if err != nil {
		return "", err
	}
	if resp.Body == nil {
		return "", fmt.Errorf("CreateVSwitch empty response")
	}
	return derefString(resp.Body.VSwitchId), nil
}

func pickNonConflictingVSwitchCIDR(vpcID string) string {
	_ = vpcID
	return "192.168.1.0/24" // 与 Django auto_create_resources 一致
}

// ── CreateSecurityGroup ─────────────────────────────────────────────────────

func aliyunCreateSecurityGroup(accessKey, secretKey, regionID, vpcID, sgName string) (string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return "", err
	}
	req := &ecsclient.CreateSecurityGroupRequest{
		RegionId:          dara.String(regionID),
		SecurityGroupName: dara.String(sgName),
		VpcId:             dara.String(vpcID),
	}
	resp, err := client.CreateSecurityGroup(req)
	if err != nil {
		return "", err
	}
	if resp.Body == nil {
		return "", fmt.Errorf("CreateSecurityGroup empty response")
	}
	return derefString(resp.Body.SecurityGroupId), nil
}

// ── 查询已有资源（去重）────────────────────────────────────────────────────

func aliyunDescribeVpcs(accessKey, secretKey, regionID string) ([]string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, err
	}
	resp, err := client.DescribeVpcs(&ecsclient.DescribeVpcsRequest{
		RegionId: dara.String(regionID),
	})
	if err != nil {
		return nil, err
	}
	var ids []string
	if resp.Body != nil && resp.Body.Vpcs != nil {
		for _, v := range resp.Body.Vpcs.Vpc {
			if v != nil && v.VpcId != nil {
				ids = append(ids, derefString(v.VpcId))
			}
		}
	}
	return ids, nil
}

func aliyunDescribeVSwitches(accessKey, secretKey, regionID, vpcID, zoneID string) ([]string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, err
	}
	req := &ecsclient.DescribeVSwitchesRequest{
		RegionId: dara.String(regionID),
	}
	if vpcID != "" {
		req.VpcId = dara.String(vpcID)
	}
	if zoneID != "" {
		req.ZoneId = dara.String(zoneID)
	}
	resp, err := client.DescribeVSwitches(req)
	if err != nil {
		return nil, err
	}
	var ids []string
	if resp.Body != nil && resp.Body.VSwitches != nil {
		for _, v := range resp.Body.VSwitches.VSwitch {
			if v != nil && v.VSwitchId != nil {
				ids = append(ids, derefString(v.VSwitchId))
			}
		}
	}
	return ids, nil
}

func aliyunDescribeSecurityGroups(accessKey, secretKey, regionID, vpcID string) ([]string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, err
	}
	req := &ecsclient.DescribeSecurityGroupsRequest{
		RegionId: dara.String(regionID),
	}
	if vpcID != "" {
		req.VpcId = dara.String(vpcID)
	}
	resp, err := client.DescribeSecurityGroups(req)
	if err != nil {
		return nil, err
	}
	var ids []string
	if resp.Body != nil && resp.Body.SecurityGroups != nil {
		for _, sg := range resp.Body.SecurityGroups.SecurityGroup {
			if sg != nil && sg.SecurityGroupId != nil {
				ids = append(ids, derefString(sg.SecurityGroupId))
			}
		}
	}
	return ids, nil
}

// ── Rich describe for network listing APIs ──────────────────────────────────

type networkResource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CidrBlock string `json:"cidr_block,omitempty"`
	VpcID     string `json:"vpc_id,omitempty"`
	ZoneID    string `json:"zone_id,omitempty"`
}

func aliyunDescribeVpcsRich(accessKey, secretKey, regionID string) ([]networkResource, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.DescribeVpcs(&ecsclient.DescribeVpcsRequest{
		RegionId: dara.String(regionID),
	})
	if err != nil {
		return nil, "", err
	}
	var result []networkResource
	rid := ""
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.Vpcs != nil {
			for _, v := range resp.Body.Vpcs.Vpc {
				if v == nil {
					continue
				}
				result = append(result, networkResource{
					ID:        derefString(v.VpcId),
					Name:      defaultString(derefString(v.VpcName), ""),
					CidrBlock: derefString(v.CidrBlock),
				})
			}
		}
	}
	return result, rid, nil
}

func aliyunDescribeVSwitchesRich(accessKey, secretKey, regionID, vpcID, zoneID string) ([]networkResource, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	req := &ecsclient.DescribeVSwitchesRequest{
		RegionId: dara.String(regionID),
	}
	if vpcID != "" {
		req.VpcId = dara.String(vpcID)
	}
	if zoneID != "" {
		req.ZoneId = dara.String(zoneID)
	}
	resp, err := client.DescribeVSwitches(req)
	if err != nil {
		return nil, "", err
	}
	var result []networkResource
	rid := ""
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.VSwitches != nil {
			for _, v := range resp.Body.VSwitches.VSwitch {
				if v == nil {
					continue
				}
				result = append(result, networkResource{
					ID:        derefString(v.VSwitchId),
					Name:      defaultString(derefString(v.VSwitchName), ""),
					CidrBlock: derefString(v.CidrBlock),
					VpcID:     derefString(v.VpcId),
					ZoneID:    derefString(v.ZoneId),
				})
			}
		}
	}
	return result, rid, nil
}

func aliyunDescribeSecurityGroupsRich(accessKey, secretKey, regionID, vpcID string) ([]networkResource, string, error) {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return nil, "", err
	}
	req := &ecsclient.DescribeSecurityGroupsRequest{
		RegionId: dara.String(regionID),
	}
	if vpcID != "" {
		req.VpcId = dara.String(vpcID)
	}
	resp, err := client.DescribeSecurityGroups(req)
	if err != nil {
		return nil, "", err
	}
	var result []networkResource
	rid := ""
	if resp.Body != nil {
		rid = derefString(resp.Body.RequestId)
		if resp.Body.SecurityGroups != nil {
			for _, sg := range resp.Body.SecurityGroups.SecurityGroup {
				if sg == nil {
					continue
				}
				result = append(result, networkResource{
					ID:    derefString(sg.SecurityGroupId),
					Name:  defaultString(derefString(sg.SecurityGroupName), ""),
					VpcID: derefString(sg.VpcId),
				})
			}
		}
	}
	return result, rid, nil
}

func getOrCreateVpc(accessKey, secretKey, regionID, taskID string) (string, error) {
	existing, err := aliyunDescribeVpcs(accessKey, secretKey, regionID)
	if err != nil {
		return "", fmt.Errorf("describe VPCs: %w", err)
	}
	if len(existing) > 0 {
		return existing[0], nil
	}
	vpcName := fmt.Sprintf("task-%s-vpc", taskID)
	vpcID, _, err := aliyunCreateVpc(accessKey, secretKey, regionID, vpcName, "")
	return vpcID, err
}

func getOrCreateVSwitch(accessKey, secretKey, regionID, zoneID, vpcID, taskID string) (string, error) {
	existing, err := aliyunDescribeVSwitches(accessKey, secretKey, regionID, vpcID, zoneID)
	if err != nil {
		return "", fmt.Errorf("describe vSwitches: %w", err)
	}
	if len(existing) > 0 {
		return existing[0], nil
	}
	switchName := fmt.Sprintf("task-%s-vsw", taskID)
	return aliyunCreateVSwitch(accessKey, secretKey, regionID, zoneID, vpcID, switchName, "")
}

func getOrCreateSecurityGroup(accessKey, secretKey, regionID, vpcID, taskID string, eventData map[string]interface{}) (string, error) {
	existing, err := aliyunDescribeSecurityGroups(accessKey, secretKey, regionID, vpcID)
	if err != nil {
		return "", fmt.Errorf("describe SGs: %w", err)
	}
	if len(existing) > 0 {
		return existing[0], nil
	}
	sgName := fmt.Sprintf("task-%s-sg", taskID)
	sgID, err := aliyunCreateSecurityGroup(accessKey, secretKey, regionID, vpcID, sgName)
	if err != nil {
		return "", err
	}
	// 授权入站规则
	if err := authorizeAutoSecurityGroupIngress(accessKey, secretKey, regionID, sgID, eventData); err != nil {
		logInfo("auto-sg ingress authorization failed: "+err.Error(), taskID)
	}
	return sgID, nil
}

// ── 自动资源创建编排 ───────────────────────────────────────────────────────

type autoCreateResult struct {
	VpcID           string
	VSwitchID       string
	SecurityGroupID string
}

func autoCreateResources(accessKey, secretKey, regionID, zoneID, taskID string, eventData map[string]interface{}) (*autoCreateResult, error) {
	res := &autoCreateResult{}

	if boolField(eventData, "auto_create_vpc") {
		vpcID, err := getOrCreateVpc(accessKey, secretKey, regionID, taskID)
		if err != nil {
			return nil, fmt.Errorf("auto-create VPC: %w", err)
		}
		res.VpcID = vpcID
		eventData["vpc_id"] = vpcID
	} else {
		res.VpcID = strField(eventData, "vpc_id")
	}

	if boolField(eventData, "auto_create_security_group") {
		vpcID := strField(eventData, "vpc_id")
		sgID, err := getOrCreateSecurityGroup(accessKey, secretKey, regionID, vpcID, taskID, eventData)
		if err != nil {
			return nil, fmt.Errorf("auto-create SG: %w", err)
		}
		res.SecurityGroupID = sgID
		eventData["security_group_id"] = sgID
	} else {
		res.SecurityGroupID = strField(eventData, "security_group_id")
	}

	if boolField(eventData, "auto_create_vswitch") {
		vpcID := strField(eventData, "vpc_id")
		vswID, err := getOrCreateVSwitch(accessKey, secretKey, regionID, zoneID, vpcID, taskID)
		if err != nil {
			return nil, fmt.Errorf("auto-create vSwitch: %w", err)
		}
		res.VSwitchID = vswID
		eventData["vswitch_id"] = vswID
	} else {
		res.VSwitchID = strField(eventData, "vswitch_id")
	}

	return res, nil
}

// authorizeAutoSecurityGroupIngress 为自动创建的安全组添加入站规则。
// 复用现有 aliyun_authorize_ingress.go 中的 authorizeSecurityGroupIngress 模式。
func authorizeAutoSecurityGroupIngress(accessKey, secretKey, regionID, securityGroupID string, eventData map[string]interface{}) error {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return err
	}

	// 收集需要放行的 CIDR
	cidrs := []string{"0.0.0.0/0"} // 默认全放行（与 Django auto_create_resources 一致）

	// 额外入站 CIDR（来自环境变量或请求）
	if extras := resolveExtraIngressCIDRs(eventData); len(extras) > 0 {
		cidrs = append(cidrs, extras...)
	}

	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		// IPv4 only for ECS security group ingress
		if ipNet.IP.To4() == nil {
			continue
		}

		// SSH (22)
		_ = authorizePortRange(client, securityGroupID, "22/22", cidr, "tcp", "task2app-ssh")
		// HTTP (80)
		_ = authorizePortRange(client, securityGroupID, "80/80", cidr, "tcp", "task2app-http")
		// HTTPS (443)
		_ = authorizePortRange(client, securityGroupID, "443/443", cidr, "tcp", "task2app-https")
		// App port range (8080-9000)
		_ = authorizePortRange(client, securityGroupID, "8080/9000", cidr, "tcp", "task2app-app")
	}
	return nil
}

func authorizePortRange(client *ecsclient.Client, securityGroupID, portRange, cidr, protocol, description string) error {
	_, err := client.AuthorizeSecurityGroup(&ecsclient.AuthorizeSecurityGroupRequest{
		RegionId:        client.RegionId,
		SecurityGroupId: dara.String(securityGroupID),
		IpProtocol:      dara.String(protocol),
		PortRange:       dara.String(portRange),
		SourceCidrIp:    dara.String(cidr),
		Description:     dara.String(description),
	})
	return err
}

// ── ModifyVpcAttribute ──────────────────────────────────────────────────────

func aliyunModifyVpcAttribute(accessKey, secretKey, regionID, vpcID, vpcName string) error {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return err
	}
	_, err = client.ModifyVpcAttribute(&ecsclient.ModifyVpcAttributeRequest{
		VpcId:   dara.String(vpcID),
		VpcName: dara.String(vpcName),
	})
	return err
}

func aliyunModifyVSwitchAttribute(accessKey, secretKey, regionID, vswitchID, vswitchName string) error {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return err
	}
	_, err = client.ModifyVSwitchAttribute(&ecsclient.ModifyVSwitchAttributeRequest{
		VSwitchId:   dara.String(vswitchID),
		VSwitchName: dara.String(vswitchName),
	})
	return err
}

func aliyunModifySecurityGroupAttribute(accessKey, secretKey, regionID, securityGroupID, securityGroupName string) error {
	client, err := newECSClient(accessKey, secretKey, regionID)
	if err != nil {
		return err
	}
	_, err = client.ModifySecurityGroupAttribute(&ecsclient.ModifySecurityGroupAttributeRequest{
		SecurityGroupId:   dara.String(securityGroupID),
		SecurityGroupName: dara.String(securityGroupName),
	})
	return err
}
