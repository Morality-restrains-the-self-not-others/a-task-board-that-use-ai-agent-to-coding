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

var djangoHTTP = &http.Client{Timeout: 30 * time.Second}

// verifyCompanyExists checks company existence via taskTenantService.
// Migrated from Django /api/internal/companies/{id}/exists/ (OPT-20260729-056).
func verifyCompanyExists(companyID string) error {
	return verifyCompanyViaTenantService(companyID)
}

func projectServicePost(path string, payload map[string]interface{}) (map[string]interface{}, int, error) {
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(cfg.ProjectServiceURL, "/") + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 500, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := djangoHTTP.Do(req)
	if err != nil {
		return nil, 502, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, resp.StatusCode, nil
}
