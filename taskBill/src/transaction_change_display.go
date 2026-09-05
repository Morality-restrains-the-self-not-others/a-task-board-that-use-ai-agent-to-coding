package main

import (
	"fmt"
	"strings"
)

type resourceChange struct {
	ResourceType string
	Quantity     int64
	Region       string
	Label        string
	Display      string
}

func resourceTypeLabelZH(resourceType string) string {
	switch resourceType {
	case ResourceTypeTaskPost:
		return "任务帖"
	case ResourceTypeGitlabDisk:
		return "GitLab 磁盘"
	case ResourceTypeGitlabTraffic:
		return "GitLab 流量"
	default:
		return resourceType
	}
}

func resourceTypeUnitZH(resourceType string) string {
	switch resourceType {
	case ResourceTypeTaskPost:
		return "帖"
	case ResourceTypeGitlabDisk, ResourceTypeGitlabTraffic:
		return "GB"
	default:
		return ""
	}
}

func formatResourceChangeLine(resourceType string, quantity int64, region string) string {
	label := resourceTypeLabelZH(resourceType)
	unit := resourceTypeUnitZH(resourceType)
	line := fmt.Sprintf("%s +%d %s", label, quantity, unit)
	if region != "" && (resourceType == ResourceTypeGitlabDisk || resourceType == ResourceTypeGitlabTraffic) {
		line += "（" + region + "）"
	}
	return strings.TrimSpace(line)
}

func formatAdminGrantDescription(grants []ResourceGrantInput) string {
	if len(grants) == 0 {
		return "管理员后台赠送资源"
	}
	parts := make([]string, 0, len(grants))
	reasons := make([]string, 0, len(grants))
	seenReason := map[string]struct{}{}
	for _, g := range grants {
		parts = append(parts, formatResourceChangeLine(g.ResourceType, g.Quantity, strings.TrimSpace(g.Region)))
		reason := strings.TrimSpace(g.Reason)
		if reason == "" {
			continue
		}
		if _, ok := seenReason[reason]; ok {
			continue
		}
		seenReason[reason] = struct{}{}
		reasons = append(reasons, reason)
	}
	desc := "管理员后台赠送：" + strings.Join(parts, "；")
	if len(reasons) == 0 {
		return desc
	}
	joined := strings.Join(reasons, "；")
	if strings.Contains(desc, joined) {
		return desc
	}
	return desc + "（" + joined + "）"
}

func adminGrantTransactionDescription(grants []ResourceGrantInput, membershipTier, membershipReason string) string {
	membershipTier = strings.TrimSpace(membershipTier)
	membershipReason = strings.TrimSpace(membershipReason)
	if membershipTier != "" && len(grants) == 0 {
		if membershipReason != "" {
			return membershipReason
		}
		return "管理员设置会员等级为 " + membershipTier
	}
	desc := formatAdminGrantDescription(grants)
	if membershipTier != "" {
		desc += "；会员等级=" + membershipTier
	}
	return desc
}

func pointsSourceTypeDisplay(src string) string {
	switch src {
	case "user_recharge_paypal":
		return "PayPal 支付"
	case "user_recharge_wechat":
		return "微信支付"
	case "user_recharge_admin":
		return "管理员直充"
	case "user_recharge":
		return "用户支付（历史）"
	case "admin_grant":
		return "后台赠送"
	case "promotion":
		return "活动赠送"
	case "adjustment":
		return "人工调账"
	case "quota_consumption":
		return "配额消耗"
	case "consumption":
		return "消耗"
	case "resource_purchase":
		return "资源购买"
	case "resource_grant_expiry":
		return "赠送到期"
	default:
		return src
	}
}

func createdAtSecond(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) >= 19 {
		return s[:19]
	}
	return s
}

func joinResourceChangeDisplays(changes []resourceChange) string {
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		if c.Display == "" {
			continue
		}
		parts = append(parts, c.Display)
	}
	return strings.Join(parts, "；")
}

func resourceChangeJSON(c resourceChange) map[string]interface{} {
	out := map[string]interface{}{
		"resource_type": c.ResourceType,
		"quantity":      c.Quantity,
		"label":         c.Label,
		"display":       c.Display,
	}
	if c.Region != "" {
		out["region"] = c.Region
	}
	return out
}
