package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"tracelog"
)

// publishAIInstructStreamSSE is a var so tests can stub HTTP fan-out.
var publishAIInstructStreamSSE = publishAIInstructStreamSSEImpl

func publishAIInstructStreamSSEImpl(ctx context.Context, taskID, instructID, phase, message string, httpStatus *int, apiEndpoint, traceID string) {
	if strings.TrimSpace(cfg.TaskSseURL) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusData := map[string]interface{}{
		"status":      "ai_instruct_stream",
		"instruct_id": instructID,
		"phase":       phase,
		"message":     message,
		"progress":    0,
		"event_name":  "server_status_update",
	}
	if httpStatus != nil {
		statusData["http_status"] = *httpStatus
	}
	if apiEndpoint != "" {
		statusData["api_endpoint"] = apiEndpoint
	}
	body, _ := json.Marshal(map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.TaskSseURL, "/")+"/internal/publish", bytes.NewReader(body))
	if err != nil {
		log.Printf("[taskAIComment] sse publish build req: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.TaskSseSecret != "" {
		req.Header.Set("X-Task-Sse-Secret", cfg.TaskSseSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		log.Printf("[taskAIComment] sse publish error task=%s instruct=%s: %v", taskID, instructID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("[taskAIComment] sse publish status=%d task=%s instruct=%s body=%s", resp.StatusCode, taskID, instructID, strings.TrimSpace(string(raw)))
	}
}

func publishContainerAgentStreamSSE(ctx context.Context, taskID, agentCommentID, phase, message, traceID string) {
	if strings.TrimSpace(cfg.TaskSseURL) == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusData := map[string]interface{}{
		"status":           "container_agent_stream",
		"agent_comment_id": agentCommentID,
		"phase":            phase,
		"message":          message,
		"progress":         0,
		"event_name":       "server_status_update",
	}
	body, _ := json.Marshal(map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.TaskSseURL, "/")+"/internal/publish", bytes.NewReader(body))
	if err != nil {
		log.Printf("[taskAIComment] event=container_agent_sse_build_err task=%s agent_comment=%s: %v", taskID, agentCommentID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.TaskSseSecret != "" {
		req.Header.Set("X-Task-Sse-Secret", cfg.TaskSseSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		log.Printf("[taskAIComment] event=container_agent_sse_err task=%s agent_comment=%s phase=%s: %v", taskID, agentCommentID, phase, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("[taskAIComment] event=container_agent_sse_status task=%s agent_comment=%s status=%d body=%s",
			taskID, agentCommentID, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}
