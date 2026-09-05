package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

var authHTTP = &http.Client{Timeout: 10 * time.Second}

func authBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskAuthURL), "/")
	if base == "" {
		return "http://127.0.0.1:8003"
	}
	return base
}

// authGetUserEmail fetches email from taskAuth login methods (empty on miss/error).
func authGetUserEmail(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	ctx := context.Background()
	u := authBaseURL() + "/api/accounts/users/" + userID + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := authHTTP.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return ""
	}
	if email := strings.TrimSpace(strField(out, "email")); email != "" {
		return email
	}
	methods, _ := out["login_methods"].([]interface{})
	for _, item := range methods {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(strField(m, "method_type")) == "email" {
			if id := strings.TrimSpace(strField(m, "identifier")); id != "" {
				return id
			}
		}
	}
	return ""
}
