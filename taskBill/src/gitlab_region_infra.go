package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

const (
	gitlabRegionInfraReady       = "ready"
	gitlabRegionInfraPendingNode = "pending_node"
)

// errGitlabRegionInfraPending 节点未部署，禁止调用 GitLab Admin API。
var errGitlabRegionInfraPending = errors.New("该区域 GitLab 节点尚未部署，请先创建服务节点并标记为已就绪")

// NormalizeGitlabRegionInfraStatus maps empty/unknown to ready (存量腾讯云)。
func NormalizeGitlabRegionInfraStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case gitlabRegionInfraPendingNode:
		return gitlabRegionInfraPendingNode
	default:
		return gitlabRegionInfraReady
	}
}

// ParseGitlabRegionInfraStatus accepts only ready|pending_node.
func ParseGitlabRegionInfraStatus(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return gitlabRegionInfraReady, nil
	}
	if s == gitlabRegionInfraReady || s == gitlabRegionInfraPendingNode {
		return s, nil
	}
	return "", errors.New("infra_status 须为 ready 或 pending_node")
}

func gitlabRegionIsPendingNode(r *GitlabRegion) bool {
	if r == nil {
		return false
	}
	return NormalizeGitlabRegionInfraStatus(r.InfraStatus) == gitlabRegionInfraPendingNode
}

func publishGitlabManualNodeFulfillmentIfNeeded(ctx context.Context, tenantID, orderID int64, items []ResourceOrderItem) {
	for _, item := range items {
		if item.ResourceType != ResourceTypeGitlabDisk && item.ResourceType != ResourceTypeGitlabTraffic {
			continue
		}
		slug := strings.TrimSpace(item.Region)
		if slug == "" {
			continue
		}
		reg, err := getGitlabRegionBySlug(slug)
		if err != nil || !gitlabRegionIsPendingNode(reg) {
			continue
		}
		slog.InfoContext(ctx, "gitlab_manual_node_fulfillment_queued",
			"level", "info",
			"tenant_id", formatID(tenantID),
			"order_id", formatID(orderID),
			"region", slug,
			"cloud_provider", reg.CloudProvider,
		)
		_ = gitlabRegionEventPublisher(ctx, "GitlabManualNodeFulfillmentQueued", map[string]interface{}{
			"tenant_id":      formatID(tenantID),
			"order_id":       formatID(orderID),
			"region_slug":    slug,
			"cloud_provider": reg.CloudProvider,
		}, formatID(orderID))
		return
	}
}
