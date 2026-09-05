package taskcommentimagementioned

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"tracelog"
)

// TaskClient loads linked projects for a task.
type TaskClient interface {
	FetchTaskLinkedProjects(ctx context.Context, taskID, tenantID, userID string) ([]map[string]interface{}, error)
}

// ProjectClient loads project details including server_run_template.
type ProjectClient interface {
	FetchProject(ctx context.Context, tenantID, projectID, userID string) (map[string]interface{}, error)
}

// CloudClient posts start-vm / start-vm-auto.
type CloudClient interface {
	StartVM(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error
}

// AICommentClient updates container agent comment run_status.
type AICommentClient interface {
	SetRunStatusByParent(ctx context.Context, parentCommentID, runStatus string) error
}

// ErrAICommentNotFound 表示按 parent 回写运行状态时 AI 评论不存在或已清理
// （taskAIComment by-parent status 返回 HTTP 404）。此时 start-vm 已成功，
// 状态回写应降级为 warn 并写明可操作原因，避免误以为启服失败（OPT-20260811-056）。
var ErrAICommentNotFound = errors.New("aiComment by-parent status 404: parent comment not found or cleaned up")

type httpTaskClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

type httpProjectClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type httpCloudClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

type httpAICommentClient struct {
	BaseURL        string
	InternalSecret string
	HTTPClient     *http.Client
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func envBase(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}

func envSecret(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func NewDefaultTaskClient() TaskClient {
	return &httpTaskClient{
		BaseURL:        envBase("TASK_TASK_SERVICE_BASE_URL", "http://127.0.0.1:8017"),
		InternalSecret: envSecret("SHARED_INTERNAL_SECRET", "TASK_TASK_INTERNAL_SECRET"),
		HTTPClient:     defaultHTTPClient(),
	}
}

func NewDefaultProjectClient() ProjectClient {
	return &httpProjectClient{
		BaseURL:    envBase("TASK_PROJECT_SERVICE_BASE_URL", "http://127.0.0.1:8016"),
		HTTPClient: defaultHTTPClient(),
	}
}

func NewDefaultCloudClient() CloudClient {
	return &httpCloudClient{
		BaseURL:        envBase("TASK_CLOUD_SERVICE_BASE_URL", "http://127.0.0.1:8018"),
		InternalSecret: envSecret("SHARED_INTERNAL_SECRET", "TASK_CLOUD_INTERNAL_SECRET"),
		HTTPClient:     defaultHTTPClient(),
	}
}

func NewDefaultAICommentClient() AICommentClient {
	return &httpAICommentClient{
		BaseURL:        envBase("TASK_AI_COMMENT_URL", "http://127.0.0.1:8019"),
		InternalSecret: envSecret("TASK_AI_COMMENT_INTERNAL_SECRET", "SHARED_INTERNAL_SECRET"),
		HTTPClient:     defaultHTTPClient(),
	}
}

func (c *httpTaskClient) FetchTaskLinkedProjects(ctx context.Context, taskID, tenantID, userID string) ([]map[string]interface{}, error) {
	// Prefer public GET with internal-style auth headers (X-Auth-User-Id).
	url := fmt.Sprintf("%s/api/tasks/%s", strings.TrimRight(c.BaseURL, "/"), taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if userID != "" {
		req.Header.Set("X-Auth-User-Id", userID)
		req.Header.Set("X-User-Id", userID)
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		// Fallback: container-snapshot (service-to-service).
		return c.fetchLinkedFromSnapshot(ctx, taskID)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return asObjectSlice(out["projects"]), nil
}

func (c *httpTaskClient) fetchLinkedFromSnapshot(ctx context.Context, taskID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/internal/tasks/%s/container-snapshot", strings.TrimRight(c.BaseURL, "/"), taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("task snapshot HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return asObjectSlice(out["projects"]), nil
}

func (c *httpProjectClient) FetchProject(ctx context.Context, tenantID, projectID, userID string) (map[string]interface{}, error) {
	// Convention path (legacy /api/tenant/{tid}/projects/{pid}/ retired — OPT-20260809-006).
	url := fmt.Sprintf("%s/api/projects/tenant_id/%s/%s/", strings.TrimRight(c.BaseURL, "/"), tenantID, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if userID != "" {
		req.Header.Set("X-Auth-User-Id", userID)
		req.Header.Set("X-User-Id", userID)
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("project GET HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *httpCloudClient) StartVM(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error {
	apiPath = strings.Trim(apiPath, "/")
	// Convention path (legacy /api/tenant/{tid}/workspace/{wid}/cloud/compute/ retired).
	url := fmt.Sprintf(
		"%s/api/cloud/compute/%s/tenant_id/%s/workspace_id/%s/",
		strings.TrimRight(c.BaseURL, "/"), apiPath, tenantID, workspaceID,
	)
	if body == nil {
		body = map[string]interface{}{}
	}
	// Align with taskTaskService startVM: image invoker drives idle-reuse gate.
	if userID != "" {
		if _, ok := body["image_invoker_user_id"]; !ok {
			body["image_invoker_user_id"] = userID
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Auth-User-Id", userID)
	}
	if tenantID != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	if c.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud %s HTTP %d: %s", apiPath, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (c *httpAICommentClient) SetRunStatusByParent(ctx context.Context, parentCommentID, runStatus string) error {
	parentCommentID = strings.TrimSpace(parentCommentID)
	runStatus = strings.TrimSpace(runStatus)
	if parentCommentID == "" || runStatus == "" {
		return fmt.Errorf("parent_comment_id and run_status required")
	}
	url := fmt.Sprintf(
		"%s/api/internal/task-ai-comment/container-agent-comments/by-parent/%s/status",
		strings.TrimRight(c.BaseURL, "/"), parentCommentID,
	)
	raw, err := json.Marshal(map[string]string{"run_status": runStatus})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.InternalSecret != "" {
		req.Header.Set("X-TaskAIComment-Internal-Secret", c.InternalSecret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := c.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w (parent=%s): %s", ErrAICommentNotFound, parentCommentID, strings.TrimSpace(string(respBody)))
		}
		return fmt.Errorf("aiComment status HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func asObjectSlice(v interface{}) []map[string]interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		if typed, ok := v.([]map[string]interface{}); ok {
			return typed
		}
		return nil
	}
	out := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}
