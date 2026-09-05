package main

import (
	"fmt"
	"strings"
)

// parseCreateTaskAgentModels reads optional body.agent_models.
// When auto_run is false the field is ignored.
// When auto_run is true and the field is present it must be exactly one
// {provider, model} object with both strings non-empty.
func parseCreateTaskAgentModels(body map[string]interface{}, autoRun bool) ([]map[string]interface{}, error) {
	if body == nil || !autoRun {
		return nil, nil
	}
	raw, ok := body["agent_models"]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("agent_models must be an array")
	}
	if len(arr) != 1 {
		return nil, fmt.Errorf("agent_models must contain exactly one item when auto_run")
	}
	item, ok := arr[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("agent_models item must be an object")
	}
	provider := optionalJSONString(item, "provider")
	model := optionalJSONString(item, "model")
	if provider == "" || model == "" {
		return nil, fmt.Errorf("agent_models item requires provider and model")
	}
	return []map[string]interface{}{
		{"provider": provider, "model": model},
	}, nil
}

func optionalJSONString(item map[string]interface{}, key string) string {
	if item == nil {
		return ""
	}
	raw, ok := item[key]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return strings.TrimSpace(s)
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if s == "<nil>" {
		return ""
	}
	return s
}
