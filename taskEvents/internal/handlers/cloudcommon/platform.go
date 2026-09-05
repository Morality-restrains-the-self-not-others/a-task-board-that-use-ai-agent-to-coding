package cloudcommon

import "strings"

// IsLocalSkipCloudPlatform reports platforms that have no vendor DeleteInstance/RunInstances.
// mock / relay-local are first-class local runtimes; they must not be dead-lettered.
func IsLocalSkipCloudPlatform(platform string) bool {
	p := strings.ToLower(strings.TrimSpace(platform))
	return p == "mock" || p == "relay-local" || strings.HasPrefix(p, "relay")
}

// LooksLikeAliyunInstanceID reports ECS instance ids (i-...).
func LooksLikeAliyunInstanceID(instanceID string) bool {
	return strings.HasPrefix(strings.TrimSpace(instanceID), "i-")
}

// ShouldSkipCloudAPI is true when the stop/start must not call Aliyun.
// mock- prefixed instances are always local. A local platform label on a
// non-ECS id is also local. A stale mock label on a real i-* instance is NOT skipped.
func ShouldSkipCloudAPI(platform, instanceID string) bool {
	id := strings.TrimSpace(instanceID)
	if strings.HasPrefix(id, "mock-") {
		return true
	}
	return IsLocalSkipCloudPlatform(platform) && !LooksLikeAliyunInstanceID(id)
}

// NormalizeStopPlatform remaps a stale local label on a real ECS id to aliyun
// so CLOUD_SERVER_STOPPED still releases the vendor instance.
func NormalizeStopPlatform(platform, instanceID string) string {
	p := strings.TrimSpace(platform)
	if p == "" {
		return "aliyun"
	}
	if IsLocalSkipCloudPlatform(p) && LooksLikeAliyunInstanceID(instanceID) {
		return "aliyun"
	}
	return p
}
