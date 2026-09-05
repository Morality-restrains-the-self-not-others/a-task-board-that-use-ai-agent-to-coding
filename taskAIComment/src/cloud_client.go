package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

var cloudHTTP = &http.Client{Timeout: 15 * time.Second}

func cloudServiceBase() string {
	if v := strings.TrimSpace(cfg.TaskCloudServiceURL); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8018"
}

func validatePostBeforeCreate(ctx context.Context, payload map[string]interface{}, testHeaders map[string]string) (int, map[string]interface{}, error) {
	return validatePostViaCloudService(ctx, payload, testHeaders)
}

func validatePostViaCloudService(ctx context.Context, payload map[string]interface{}, testHeaders map[string]string) (int, map[string]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	body, _ := json.Marshal(payload)
	url := cloudServiceBase() + "/api/internal/cloud-server-config/validate-ai-comment-post/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 500, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	for k, v := range testHeaders {
		req.Header.Set(k, v)
	}
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return 502, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out, nil
}
