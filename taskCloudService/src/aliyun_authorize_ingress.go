package main

import (
	"fmt"
	"net"
	"strings"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func isNonPublicClientIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// RFC1918 + CGNAT
		if ip4[0] == 10 {
			return true
		}
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
		if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

// normalizePublicHostCIDR converts a host IP or /32|/128 into Aliyun ingress CIDR.
// Rejects loopback / private / non-host prefixes (must not pollute auto SG whitelist).
func normalizePublicHostCIDR(raw string) string {
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
		if isNonPublicClientIP(ip) {
			return ""
		}
		if bits == 32 && ones == 32 {
			return ip.To4().String() + "/32"
		}
		if bits == 128 && ones == 128 {
			return ip.String() + "/128"
		}
		return ""
	}
	ip := net.ParseIP(raw)
	if ip == nil || isNonPublicClientIP(ip) {
		return ""
	}
	if ip.To4() != nil {
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}

func isIPv6HostCIDR(cidr string) bool {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return false
	}
	if strings.Contains(cidr, "/") {
		ip, _, err := net.ParseCIDR(cidr)
		return err == nil && ip != nil && ip.To4() == nil
	}
	ip := net.ParseIP(cidr)
	return ip != nil && ip.To4() == nil
}

func isAliyunDuplicatePermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidpermission.duplicate") ||
		strings.Contains(msg, "permission.duplicate") ||
		strings.Contains(msg, "already exists")
}

func isAliyunInvalidSourceCIDRError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalidparam.sourcecidrip") ||
		strings.Contains(msg, "invalidparam.ipv6sourcecidrip") ||
		strings.Contains(msg, "sourcecidrip is not valid") ||
		strings.Contains(msg, "ipv6sourcecidrip is not valid")
}

// aliyunAuthorizeClientIngress adds ALL/-1/-1 accept for clientCIDR on the SG (idempotent).
func aliyunAuthorizeClientIngress(accessKey, secretKey, region, sgID, clientCIDR string) (requestID string, already bool, err error) {
	region = strings.TrimSpace(region)
	sgID = strings.TrimSpace(sgID)
	cidr := normalizePublicHostCIDR(clientCIDR)
	if region == "" || sgID == "" || cidr == "" {
		return "", false, fmt.Errorf("region, security_group_id and client ip required")
	}
	if cidr == "0.0.0.0/0" || cidr == "::/0" {
		return "", false, fmt.Errorf("refusing full-open ingress")
	}
	client, err := newECSClient(accessKey, secretKey, region)
	if err != nil {
		return "", false, err
	}
	req := &ecsclient.AuthorizeSecurityGroupRequest{
		RegionId:        dara.String(region),
		SecurityGroupId: dara.String(sgID),
		IpProtocol:      dara.String("all"),
		PortRange:       dara.String("-1/-1"),
	}
	if isIPv6HostCIDR(cidr) {
		req.Ipv6SourceCidrIp = dara.String(cidr)
	} else {
		req.SourceCidrIp = dara.String(cidr)
	}
	resp, err := client.AuthorizeSecurityGroup(req)
	if err != nil {
		if isAliyunDuplicatePermissionError(err) {
			return "", true, nil
		}
		if isIPv6HostCIDR(cidr) && isAliyunInvalidSourceCIDRError(err) {
			return "", false, fmt.Errorf("ipv6 ingress unsupported: %w", err)
		}
		return "", false, err
	}
	if resp != nil && resp.Body != nil && resp.Body.RequestId != nil {
		requestID = strings.TrimSpace(*resp.Body.RequestId)
	}
	return requestID, false, nil
}
