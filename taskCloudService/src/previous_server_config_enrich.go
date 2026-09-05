package main

import (
	"strings"
)

func enrichPreviousServerConfig(tenantID string, cfg map[string]interface{}, h *CloudServerConfigHistory) map[string]interface{} {
	if cfg == nil || h == nil {
		return cfg
	}
	authID := strings.TrimSpace(h.AuthorizationID)
	instType := strings.TrimSpace(h.InstanceTypeID)
	if authID == "" || instType == "" || strings.HasPrefix(strings.ToLower(authID), "mock") {
		return cfg
	}
	auth, err := loadCloudAuth(tenantID, authID)
	if err != nil || auth == nil || strings.TrimSpace(auth.PlatformType) != "aliyun" {
		return cfg
	}
	inStock := aliyunInstanceTypeInStock(auth.SecretID, auth.SecretKey, h.Region, instType, h.ZoneID)
	cfg["availability"] = map[string]interface{}{
		"available":        inStock,
		"instance_type_id": instType,
		"region_id":        h.Region,
		"zone_id":          h.ZoneID,
	}
	bandwidth := 5
	if price, _, err := aliyunDescribePrice(
		auth.SecretID, auth.SecretKey, h.Region, instType, "cloud_essd", h.StorageGB, bandwidth, "SpotAsPriceGo",
	); err == nil && len(price) > 0 {
		cfg["price"] = price
	}
	return cfg
}
