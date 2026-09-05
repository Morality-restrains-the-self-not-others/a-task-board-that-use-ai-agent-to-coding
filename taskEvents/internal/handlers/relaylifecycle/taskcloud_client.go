package relaylifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"tracelog"
)

// TaskCloudInternalClient calls taskCloudService internal APIs.
type TaskCloudInternalClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

func NewTaskCloudInternalClient() *TaskCloudInternalClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8018"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET"))
	return &TaskCloudInternalClient{
		BaseURL:        base,
		InternalSecret: secret,
		HTTPClient:     &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *TaskCloudInternalClient) post(ctx context.Context, path string, payload map[string]interface{}) error {
	if c == nil {
		c = NewTaskCloudInternalClient()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimRight(c.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func eventScope(data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"tenant_id":    str(data, "tenant_id"),
		"workspace_id": str(data, "workspace_id"),
		"task_id":      str(data, "task_id"),
	}
}

func eventCorrelationCtx(ctx context.Context, data map[string]interface{}) context.Context {
	tid := firstNonEmpty(str(data, "trace_id"), tracelog.TraceIDFromContext(ctx))
	if tid == "" {
		return ctx
	}
	parent := normalizeSpanField(str(data, "span_id"))
	if parent == "" {
		parent = normalizeSpanField(str(data, "parent_span_id"))
	}
	if parent == "" {
		if c := tracelog.CorrelationFromContext(ctx); c.SpanID != "" {
			parent = c.SpanID
		}
	}
	return tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID:      tid,
		SpanID:       tracelog.NewSpanID(),
		ParentSpanID: parent,
	})
}

func normalizeSpanField(raw string) string {
	return tracelog.NormalizeSpanID(strings.TrimSpace(raw))
}
