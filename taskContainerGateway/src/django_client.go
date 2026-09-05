package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type scope struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	CommentID   string
}

type containerTarget struct {
	BaseURL     string
	AccessToken string
}

type validateSessionResult struct {
	UserID     string
	AuthMethod string
	ScopeOK    bool
}

func traceIDFromContext(ctx context.Context) string {
	return tracelog.TraceIDFromContext(ctx)
}

// No Django internal API config or client remains in taskContainerGateway.
// Hot-path / job-stream / git-push / open-runtime-session / relay call
// taskCloudService (and taskAuth) internal APIs instead.

func sanitizeUpstreamURLForLog(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	// Drop query strings that may carry tokens.
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	return u
}

func forwardToOnlineService(
	ctx context.Context,
	method string,
	upstreamURL string,
	accessToken string,
	body []byte,
) (int, []byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, upstreamURL, reader)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Access-Token", accessToken)
	tracelog.ApplyOutboundHeaders(req, ctx)
	transport := &http.Transport{Proxy: nil}
	timeout := time.Duration(cfg.ForwardConnectSec)*time.Second + time.Duration(cfg.ForwardReadSec)*time.Second
	client := &http.Client{Transport: transport, Timeout: timeout}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	logURL := sanitizeUpstreamURLForLog(upstreamURL)
	if err != nil {
		tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
			"upstream_url":    logURL,
			"upstream_status": 502,
			"duration_ms":     duration,
			"detail":          fmt.Sprintf("upstream error: %v", err),
		})
		payload, _ := json.Marshal(map[string]string{
			"detail": fmt.Sprintf("upstream error: %v", err),
		})
		return http.StatusBadGateway, payload, nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
			"upstream_url":    logURL,
			"upstream_status": 502,
			"duration_ms":     duration,
			"detail":          "failed to read upstream response",
		})
		return http.StatusBadGateway, []byte(`{"detail":"failed to read upstream response"}`), nil
	}
	if len(respBody) == 0 {
		respBody = []byte("{}")
	}
	tracelog.LogForwardStage(ctx, "upstream_forward", map[string]any{
		"upstream_url":    logURL,
		"upstream_status": resp.StatusCode,
		"duration_ms":     duration,
	})
	return resp.StatusCode, respBody, nil
}
