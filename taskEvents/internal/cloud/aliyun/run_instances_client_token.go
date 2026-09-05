package aliyun

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const aliyunClientTokenMaxLen = 64

var hardwareClientTokenKeys = []string{
	"cpu_cores",
	"memory_gb",
	"storage_gb",
	"region_id",
	"zone_id",
	"instance_type",
	"system_disk_category",
	"internet_max_bandwidth_out",
	"instance_charge_type",
	"spot_strategy",
	"spot_duration",
	"auto_release_minutes",
	"auto_release_hours",
}

// ComputeClientTokenFromTaskIDAndHardware mirrors Python compute_client_token_from_task_id_and_hardware.
func ComputeClientTokenFromTaskIDAndHardware(taskID string, hardwareSnapshot map[string]interface{}) string {
	hw := make(map[string]interface{}, len(hardwareClientTokenKeys))
	for _, k := range hardwareClientTokenKeys {
		hw[k] = hardwareSnapshot[k]
	}
	canonical := canonicalJSON(map[string]interface{}{
		"task_id":  taskID,
		"hardware": hw,
	})
	digest := sha256.Sum256([]byte(canonical))
	prefix := taskID
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}
	if prefix == "" {
		prefix = "tk"
	}
	raw := fmt.Sprintf("%s-%x", prefix, digest[:12])
	return truncateClientToken(raw)
}

// ComputeRunInstancesClientTokenAfterMismatch derives a new ClientToken after IdempotentParameterMismatch.
func ComputeRunInstancesClientTokenAfterMismatch(previousToken string, requestParams map[string]interface{}, attemptIndex int) string {
	canonical := canonicalJSON(requestParams)
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", previousToken, canonical, attemptIndex)))
	prefix := previousToken
	if len(prefix) > 40 {
		prefix = prefix[:40]
	}
	if prefix == "" {
		prefix = "tk"
	}
	raw := fmt.Sprintf("%s-%x", prefix, digest[:10])
	return truncateClientToken(raw)
}

// IsIdempotentParameterMismatchAliyunError reports 403 IdempotentParameterMismatch.
func IsIdempotentParameterMismatchAliyunError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "IdempotentParameterMismatch") {
		return false
	}
	return regexp.MustCompile(`\b403\b`).MatchString(msg)
}

// IsIdempotentProcessingAliyunError reports 403 IdempotentProcessing (same token still in flight).
func IsIdempotentProcessingAliyunError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "IdempotentProcessing") {
		return false
	}
	return regexp.MustCompile(`\b403\b`).MatchString(msg)
}

func truncateClientToken(raw string) string {
	if len(raw) <= aliyunClientTokenMaxLen {
		return raw
	}
	return raw[:aliyunClientTokenMaxLen]
}

func canonicalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	var m interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return string(b)
	}
	normalized := normalizeJSONValue(m)
	out, err := json.Marshal(normalized)
	if err != nil {
		return string(b)
	}
	return string(out)
}

func normalizeJSONValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]interface{}, len(t))
		for _, k := range keys {
			out[k] = normalizeJSONValue(t[k])
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = normalizeJSONValue(item)
		}
		return out
	default:
		return fmt.Sprint(v)
	}
}
