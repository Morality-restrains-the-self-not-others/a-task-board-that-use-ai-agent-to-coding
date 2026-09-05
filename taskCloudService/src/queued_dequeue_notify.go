package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// notifyTaskServiceDequeueQueued asks TTS to drop queue membership after manual/auto_run start.
func notifyTaskServiceDequeueQueued(tenantID, taskID, reason string) {
	base := strings.TrimSpace(cfg.TaskServiceURL)
	taskID = strings.TrimSpace(taskID)
	if base == "" || taskID == "" {
		return
	}
	u := fmt.Sprintf("%s/api/internal/tasks/%s/dequeue-queued-auto-run?reason=%s",
		strings.TrimRight(base, "/"), taskID, url.QueryEscape(strings.TrimSpace(reason)))
	req, err := http.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[taskCloudService] dequeue-queued notify failed task_id=%s: %v", taskID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[taskCloudService] dequeue-queued notify status=%d task_id=%s", resp.StatusCode, taskID)
	}
}
