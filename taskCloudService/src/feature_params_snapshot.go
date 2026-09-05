package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// append-only write helpers for cloud_task_feature_params_snapshot (via saas HTTP).

func providersSummaryJSON(providersJSON string) string {
	providers := parseProvidersJSON(providersJSON)
	out := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		apiKey := strings.TrimSpace(fmt.Sprintf("%v", p["api_key"]))
		if apiKey == "<nil>" {
			apiKey = ""
		}
		hash := ""
		if apiKey != "" {
			sum := sha256.Sum256([]byte(apiKey))
			hash = fmt.Sprintf("sha256:%x", sum[:4])
		}
		models := []any{}
		if raw, ok := p["supported_models"]; ok {
			switch v := raw.(type) {
			case []any:
				models = v
			}
		}
		out = append(out, map[string]any{
			"provider":         fmt.Sprintf("%v", p["provider"]),
			"base_url":         fmt.Sprintf("%v", p["base_url"]),
			"supported_models": models,
			"api_key_hash":     hash,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(b)
}
