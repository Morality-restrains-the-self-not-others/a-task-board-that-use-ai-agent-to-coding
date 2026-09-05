package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AICommentClient calls taskAIComment internal APIs (soft-fail at call sites).
type AICommentClient struct {
	BaseURL        string
	InternalSecret string
	HTTP           *http.Client
}

func NewAICommentClient(baseURL, internalSecret string) *AICommentClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil
	}
	return &AICommentClient{
		BaseURL:        strings.TrimRight(baseURL, "/"),
		InternalSecret: strings.TrimSpace(internalSecret),
		HTTP:           &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchActiveContextPack returns at_mention_run + comment_thread from the latest active agent comment.
// Returns (nil, nil) when no active comment / 404.
func (c *AICommentClient) FetchActiveContextPack(taskID string) (map[string]interface{}, error) {
	if c == nil || c.BaseURL == "" {
		return nil, nil
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("task_id", taskID)
	u := c.BaseURL + "/api/internal/task-ai-comment/container-agent-comments/active-by-task?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if c.InternalSecret != "" {
		req.Header.Set("X-TaskAIComment-Internal-Secret", c.InternalSecret)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("aiComment active-by-task status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var body map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("aiComment decode: %w", err)
		}
	}
	pack, _ := body["context_pack"].(map[string]interface{})
	if pack == nil {
		return nil, nil
	}
	out := map[string]interface{}{}
	if v, ok := pack["at_mention_run"]; ok {
		out["at_mention_run"] = v
	}
	if v, ok := pack["comment_thread"]; ok {
		out["comment_thread"] = v
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// SoftFetchActiveContextPack logs and returns nil on failure (never errors to caller).
func SoftFetchActiveContextPack(client *AICommentClient, taskID string) map[string]interface{} {
	if client == nil {
		return nil
	}
	pack, err := client.FetchActiveContextPack(taskID)
	if err != nil {
		log.Printf("[task-credential-service] event=at_mention_context_pack_fetch_failed task_id=%s err=%v", taskID, err)
		return nil
	}
	return pack
}
