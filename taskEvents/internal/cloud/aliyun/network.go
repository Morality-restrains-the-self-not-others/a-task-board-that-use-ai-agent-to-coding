package aliyun

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
)

const (
	defaultVPCName     = "task2app-vpc"
	defaultVPCCIDR     = "192.168.0.0/16"
	defaultVSwitchName = "task2app-vswitch"
	defaultVSwitchCIDR = "192.168.1.0/24"
	// 自动创建安全组：入网白名单（用户公网 IP + 服务器公网 IP + 可选额外 CIDR），非全开。
	defaultSGName        = "task2app-sg-入网白名单"
	legacySGNameFullOpen = "task2app-sg-端口全开有风险"
	legacySGName         = "task2app-sg"
	resourceSettleDelay  = 5 * time.Second
	fullOpenIngressCIDR  = "0.0.0.0/0"
	// TASK2APP_SG_EXTRA_INGRESS_CIDRS: comma-separated CIDRs allowed in addition to
	// client/server public IPs (e.g. SaaS egress for container probes).
	extraIngressCIDRsEnv = "TASK2APP_SG_EXTRA_INGRESS_CIDRS"
)

// sgIngressRule is one AuthorizeSecurityGroup ingress permission.
type sgIngressRule struct {
	proto, port, cidr string
}

// normalizePublicIPCIDR converts a host IP or single-host CIDR into /32 or /128.
func normalizePublicIPCIDR(raw string) string {
	return normalizeIngressSourceCIDR(raw, true)
}

// normalizeIngressSourceCIDR normalizes a source CIDR for SG ingress.
// When hostOnly is true, only single-host (/32 or /128) is accepted.
// When false, any valid CIDR except full-open is accepted (for platform extras).
func normalizeIngressSourceCIDR(raw string, hostOnly bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "/") {
		ip, network, err := net.ParseCIDR(raw)
		if err != nil || ip == nil || network == nil {
			return ""
		}
		ones, bits := network.Mask.Size()
		normalized := network.String()
		if normalized == fullOpenIngressCIDR || normalized == "::/0" {
			return ""
		}
		if hostOnly {
			if bits == 32 && ones == 32 {
				return ip.To4().String() + "/32"
			}
			if bits == 128 && ones == 128 {
				return ip.String() + "/128"
			}
			return ""
		}
		return normalized
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return ""
	}
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}

// parseExtraIngressCIDRs splits a comma/semicolon/whitespace separated list.
func parseExtraIngressCIDRs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	var out []string
	seen := map[string]struct{}{}
	for _, f := range fields {
		cidr := normalizeIngressSourceCIDR(f, false)
		if cidr == "" {
			continue
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		out = append(out, cidr)
	}
	return out
}

func extraIngressCIDRsFromEnv() []string {
	return parseExtraIngressCIDRs(os.Getenv(extraIngressCIDRsEnv))
}

func mergeIngressCIDRs(lists ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, list := range lists {
		for _, raw := range list {
			cidr := normalizeIngressSourceCIDR(raw, false)
			if cidr == "" {
				continue
			}
			if _, ok := seen[cidr]; ok {
				continue
			}
			seen[cidr] = struct{}{}
			out = append(out, cidr)
		}
	}
	return out
}

// whitelistAutoSGIngressRules builds ingress for auto-created security groups.
// Host IPs are forced to /32|/128; extras may be ranges. Never includes 0.0.0.0/0.
func whitelistAutoSGIngressRules(allowedCIDRs ...string) []sgIngressRule {
	seen := map[string]struct{}{}
	var out []sgIngressRule
	for _, raw := range allowedCIDRs {
		cidr := normalizeIngressSourceCIDR(raw, true)
		if cidr == "" {
			cidr = normalizeIngressSourceCIDR(raw, false)
		}
		if cidr == "" || cidr == fullOpenIngressCIDR || cidr == "::/0" {
			continue
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		out = append(out, sgIngressRule{"all", "-1/-1", cidr})
	}
	return out
}

// defaultAutoSGIngressRules builds Phase A rules: client IP + env/event extras + SaaS egress.
func defaultAutoSGIngressRules(clientPublicIP string, extraCIDRs ...string) []sgIngressRule {
	merged := mergeIngressCIDRs([]string{clientPublicIP}, extraCIDRs, platformExtraIngressCIDRs())
	return whitelistAutoSGIngressRules(merged...)
}

// AutoNetworkInput describes optional auto-create flags for CLOUD_SERVER_START_AUTO.
type AutoNetworkInput struct {
	RegionID          string
	ZoneID            string
	AutoVPC           bool
	AutoVSwitch       bool
	AutoSG            bool
	ExistingVPCID     string
	ExistingVSwitchID string
	ExistingSGID      string
	ClientPublicIP    string
	ExtraIngressCIDRs []string
}

// AutoNetworkResult holds resolved network resource IDs.
type AutoNetworkResult struct {
	VPCID           string
	VSwitchID       string
	SecurityGroupID string
}

// ProvisionAutoNetworkResources creates VPC/VSwitch/SecurityGroup when requested.
func ProvisionAutoNetworkResources(ctx context.Context, accessKey, secretKey string, in AutoNetworkInput) (AutoNetworkResult, error) {
	_ = ctx
	if in.RegionID == "" {
		return AutoNetworkResult{}, fmt.Errorf("region_id required")
	}
	client, err := newECSClient(accessKey, secretKey, in.RegionID)
	if err != nil {
		return AutoNetworkResult{}, err
	}

	out := AutoNetworkResult{
		VPCID:           in.ExistingVPCID,
		VSwitchID:       in.ExistingVSwitchID,
		SecurityGroupID: in.ExistingSGID,
	}

	if in.AutoVPC {
		vpcID, err := getOrCreateVPC(client, in.RegionID, defaultVPCName, defaultVPCCIDR)
		if err != nil {
			return AutoNetworkResult{}, fmt.Errorf("create vpc: %w", err)
		}
		out.VPCID = vpcID
		time.Sleep(resourceSettleDelay)
	} else if out.VPCID == "" && (in.AutoVSwitch || in.AutoSG) {
		if vpcID, err := describeFirstVPCInRegion(client, in.RegionID); err == nil && vpcID != "" {
			out.VPCID = vpcID
		}
	}
	if out.VPCID == "" {
		return AutoNetworkResult{}, fmt.Errorf("vpc_id required")
	}

	if in.AutoVSwitch {
		if in.ZoneID == "" {
			return AutoNetworkResult{}, fmt.Errorf("zone_id required for auto_create_vswitch")
		}
		vswitchID, err := getOrCreateVSwitch(client, in.RegionID, in.ZoneID, out.VPCID, defaultVSwitchName, defaultVSwitchCIDR)
		if err != nil {
			return AutoNetworkResult{}, fmt.Errorf("create vswitch: %w", err)
		}
		out.VSwitchID = vswitchID
		time.Sleep(resourceSettleDelay)
	}

	if in.AutoSG {
		sgID, err := getOrCreateSecurityGroup(client, in.RegionID, out.VPCID, defaultSGName, in.ClientPublicIP, in.ExtraIngressCIDRs)
		if err != nil {
			return AutoNetworkResult{}, fmt.Errorf("create security group: %w", err)
		}
		out.SecurityGroupID = sgID
	}

	return out, nil
}

func createVPC(client *ecs.Client, region, name, cidr string) (string, error) {
	req := &ecs.CreateVpcRequest{
		RegionId:    strPtr(region),
		CidrBlock:   strPtr(cidr),
		VpcName:     strPtr(name),
		Description: strPtr(name + " - Task2App VPC"),
	}
	resp, err := client.CreateVpc(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil && resp.Body.VpcId != nil && *resp.Body.VpcId != "" {
		return *resp.Body.VpcId, nil
	}
	return "", fmt.Errorf("CreateVpc returned empty vpc id")
}

func createVSwitch(client *ecs.Client, region, zoneID, vpcID, name, cidr string) (string, error) {
	req := &ecs.CreateVSwitchRequest{
		RegionId:    strPtr(region),
		ZoneId:      strPtr(zoneID),
		VpcId:       strPtr(vpcID),
		CidrBlock:   strPtr(cidr),
		VSwitchName: strPtr(name),
		Description: strPtr(name + " - Task2App Virtual Switch"),
	}
	resp, err := client.CreateVSwitch(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil && resp.Body.VSwitchId != nil && *resp.Body.VSwitchId != "" {
		return *resp.Body.VSwitchId, nil
	}
	return "", fmt.Errorf("CreateVSwitch returned empty vswitch id")
}

func createSecurityGroupWithRules(client *ecs.Client, region, vpcID, clientPublicIP string, extraCIDRs []string) (string, error) {
	req := &ecs.CreateSecurityGroupRequest{
		RegionId:          strPtr(region),
		VpcId:             strPtr(vpcID),
		SecurityGroupName: strPtr(defaultSGName),
		Description:       strPtr("Task2App Security Group (ingress whitelist: client + server public IP + extras)"),
	}
	resp, err := client.CreateSecurityGroup(req)
	if err != nil {
		return "", err
	}
	if resp.Body == nil || resp.Body.SecurityGroupId == nil || *resp.Body.SecurityGroupId == "" {
		return "", fmt.Errorf("CreateSecurityGroup returned empty security group id")
	}
	sgID := *resp.Body.SecurityGroupId

	if err := applyIngressRules(client, region, sgID, defaultAutoSGIngressRules(clientPublicIP, extraCIDRs...)); err != nil {
		return "", err
	}
	if err := authorizeEgress(client, region, sgID, "all", "-1/-1", "0.0.0.0/0"); err != nil {
		return "", err
	}
	return sgID, nil
}

func applyIngressRules(client *ecs.Client, region, sgID string, rules []sgIngressRule) error {
	for _, rule := range rules {
		if err := authorizeIngress(client, region, sgID, rule.proto, rule.port, rule.cidr); err != nil {
			if isDuplicatePermissionError(err) {
				continue
			}
			// IPv6 rules require VPC/instance IPv6 support; never block auto-SG on that.
			if isIPv6CIDR(rule.cidr) && isInvalidSourceCidrError(err) {
				log.Printf("[aliyun] skip unsupported IPv6 ingress cidr=%s sg=%s: %v", rule.cidr, sgID, err)
				continue
			}
			return fmt.Errorf("authorize ingress cidr=%s: %w", rule.cidr, err)
		}
	}
	return nil
}

// isIPv6CIDR reports whether cidr is an IPv6 address or prefix.
func isIPv6CIDR(cidr string) bool {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return false
	}
	if strings.Contains(cidr, "/") {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil || ip == nil {
			return false
		}
		return ip.To4() == nil
	}
	ip := net.ParseIP(cidr)
	return ip != nil && ip.To4() == nil
}

// ingressSourceFields maps a CIDR to Aliyun Authorize/Revoke source fields.
// IPv4 → SourceCidrIp; IPv6 → Ipv6SourceCidrIp (never put IPv6 in SourceCidrIp).
func ingressSourceFields(cidr string) (sourceCidrIp, ipv6SourceCidrIp *string) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil, nil
	}
	if isIPv6CIDR(cidr) {
		return nil, strPtr(cidr)
	}
	return strPtr(cidr), nil
}

func isInvalidSourceCidrError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidparam.sourcecidrip") ||
		strings.Contains(msg, "invalidparam.ipv6sourcecidrip") ||
		strings.Contains(msg, "sourcecidrip is not valid") ||
		strings.Contains(msg, "ipv6sourcecidrip is not valid")
}

// ensureWhitelistIngress applies client IP + extras allowlist and revokes legacy full-open ingress.
func ensureWhitelistIngress(client *ecs.Client, region, sgID, clientPublicIP string, extraCIDRs []string) error {
	if err := revokeFullOpenIngress(client, region, sgID); err != nil {
		return err
	}
	return applyIngressRules(client, region, sgID, defaultAutoSGIngressRules(clientPublicIP, extraCIDRs...))
}

// AuthorizeServerPublicIPIngress adds the VM public IP to an auto-created whitelist SG (Phase B).
func AuthorizeServerPublicIPIngress(accessKey, secretKey, region, sgID, serverPublicIP string, extraCIDRs ...string) error {
	cidr := normalizePublicIPCIDR(serverPublicIP)
	if region == "" || sgID == "" || cidr == "" {
		return fmt.Errorf("region, security_group_id and server public ip required")
	}
	client, err := newECSClient(accessKey, secretKey, region)
	if err != nil {
		return err
	}
	if err := revokeFullOpenIngress(client, region, sgID); err != nil {
		return err
	}
	merged := mergeIngressCIDRs([]string{cidr}, extraCIDRs, platformExtraIngressCIDRs())
	return applyIngressRules(client, region, sgID, whitelistAutoSGIngressRules(merged...))
}

func revokeFullOpenIngress(client *ecs.Client, region, sgID string) error {
	return revokeIngress(client, region, sgID, "all", "-1/-1", fullOpenIngressCIDR)
}

func revokeIngress(client *ecs.Client, region, sgID, proto, portRange, cidr string) error {
	v4, v6 := ingressSourceFields(cidr)
	req := &ecs.RevokeSecurityGroupRequest{
		RegionId:         strPtr(region),
		SecurityGroupId:  strPtr(sgID),
		IpProtocol:       strPtr(proto),
		PortRange:        strPtr(portRange),
		SourceCidrIp:     v4,
		Ipv6SourceCidrIp: v6,
	}
	_, err := client.RevokeSecurityGroup(req)
	if err != nil && isMissingPermissionError(err) {
		return nil
	}
	return err
}

func isDuplicatePermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidpermission.duplicate") ||
		strings.Contains(msg, "permission.duplicate") ||
		strings.Contains(msg, "already exists")
}

func isMissingPermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidpermission.notfound") ||
		strings.Contains(msg, "permission.notfound") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

func authorizeIngress(client *ecs.Client, region, sgID, proto, portRange, cidr string) error {
	v4, v6 := ingressSourceFields(cidr)
	req := &ecs.AuthorizeSecurityGroupRequest{
		RegionId:         strPtr(region),
		SecurityGroupId:  strPtr(sgID),
		IpProtocol:       strPtr(proto),
		PortRange:        strPtr(portRange),
		SourceCidrIp:     v4,
		Ipv6SourceCidrIp: v6,
	}
	_, err := client.AuthorizeSecurityGroup(req)
	return err
}

func authorizeEgress(client *ecs.Client, region, sgID, proto, portRange, cidr string) error {
	req := &ecs.AuthorizeSecurityGroupEgressRequest{
		RegionId:        strPtr(region),
		SecurityGroupId: strPtr(sgID),
		IpProtocol:      strPtr(proto),
		PortRange:       strPtr(portRange),
		DestCidrIp:      strPtr(cidr),
	}
	_, err := client.AuthorizeSecurityGroupEgress(req)
	return err
}
