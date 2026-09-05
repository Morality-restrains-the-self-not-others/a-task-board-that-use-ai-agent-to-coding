package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var taskHTTP = &http.Client{Timeout: 15 * time.Second}

func taskRequest(method, path, tenantID, userID string, body []byte) (int, json.RawMessage, error) {
	url := strings.TrimRight(cfg.TaskServiceURL, "/") + path
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, strings.NewReader(string(body)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	if userID != "" {
		req.Header.Set("X-Auth-User-Id", userID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := taskHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, json.RawMessage(raw), nil
}

func getTaskInWorkspace(tenantID, workspaceID, taskID, userID string) (map[string]interface{}, int, error) {
	path := fmt.Sprintf("/api/tenant/%s/workspace/%s/todos/%s/", tenantID, workspaceID, taskID)
	status, raw, err := taskRequest(http.MethodGet, path, tenantID, userID, nil)
	if err != nil {
		return nil, 0, err
	}
	if status == http.StatusNotFound {
		return nil, status, fmt.Errorf("task not found")
	}
	if status == http.StatusForbidden {
		return nil, status, fmt.Errorf("forbidden")
	}
	if status != http.StatusOK {
		return nil, status, fmt.Errorf("task service returned %d", status)
	}
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, status, nil
}

func verifyTaskAccess(tenantID, workspaceID, taskID, userID string) error {
	_, status, err := getTaskInWorkspace(tenantID, workspaceID, taskID, userID)
	if err != nil {
		if status == http.StatusForbidden {
			return fmt.Errorf("forbidden")
		}
		return err
	}
	return nil
}
