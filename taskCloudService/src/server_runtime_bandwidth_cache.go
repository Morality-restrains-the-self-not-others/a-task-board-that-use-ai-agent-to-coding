package main

import "strings"

// persistInstanceBandwidth caches the last-seen Aliyun internet billing fields on the
// comment CSC so the auth_missing fallback can still render the bandwidth rows.
// Mock instances are intentionally skipped (no real billing fields).
func persistInstanceBandwidth(instanceID, chargeType string, bwOut int) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || strings.HasPrefix(instanceID, "mock-") {
		return nil
	}
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET internet_charge_type=?, internet_max_bandwidth_out=? WHERE instance_id=?`,
		strings.TrimSpace(chargeType), bwOut, instanceID,
	)
	return err
}

// loadInstanceBandwidth reads the cached internet billing fields for the instance.
// Missing rows degrade to zero values so the fallback omits the bandwidth keys.
func loadInstanceBandwidth(instanceID string) (chargeType string, bwOut int) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || strings.HasPrefix(instanceID, "mock-") {
		return "", 0
	}
	err := db.QueryRow(
		`SELECT COALESCE(internet_charge_type,''), COALESCE(internet_max_bandwidth_out,0)
		   FROM cloud_server_configs WHERE instance_id=? LIMIT 1`,
		instanceID,
	).Scan(&chargeType, &bwOut)
	if err != nil {
		return "", 0
	}
	return strings.TrimSpace(chargeType), bwOut
}

// bandwidthChargeTypeFromAttr extracts InternetChargeType from a DescribeInstances attr map.
func bandwidthChargeTypeFromAttr(attr map[string]interface{}) string {
	if attr == nil {
		return ""
	}
	if v, ok := attr["InternetChargeType"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// bandwidthOutFromAttr extracts InternetMaxBandwidthOut from a DescribeInstances attr map.
// The Aliyun SDK exposes the field as int32; tolerate the usual numeric widenings.
func bandwidthOutFromAttr(attr map[string]interface{}) int {
	if attr == nil {
		return 0
	}
	switch v := attr["InternetMaxBandwidthOut"].(type) {
	case int32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}
