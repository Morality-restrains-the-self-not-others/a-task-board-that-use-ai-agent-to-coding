package main

import (
	"fmt"
	"strings"
)

// optionalServerRunTemplate extracts a complete @镜像 start override from the comment POST body.
// Incomplete objects (missing region or cloud_platform_id) are ignored so the consumer
// falls back to the project server_run_template. The field is event-only and is not stored.
func optionalServerRunTemplate(body map[string]interface{}) map[string]interface{} {
	if body == nil {
		return nil
	}
	raw, ok := body["server_run_template"]
	if !ok || raw == nil {
		return nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil
	}
	region := mapStringField(m, "region")
	platform := mapStringField(m, "cloud_platform_id")
	if region == "" || platform == "" {
		return nil
	}
	return m
}

func mapStringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
