package containermigrateawaitready

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
)

// HTTPContainerStarter re-triggers relay container start via Container Gateway.
type HTTPContainerStarter struct {
	BaseURL        string
	InternalSecret string
	GatewaySecret  string
	Client         *http.Client
}

func NewDefaultContainerStarter() *HTTPContainerStarter {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_CONTAINER_GATEWAY_URL")), "/")
	if base == "" {
		base = "http://127.0.0.1:8014"
	}
	return &HTTPContainerStarter{
		BaseURL:        base,
		InternalSecret: strings.TrimSpace(os.Getenv("INTERNAL_SECRET")),
		GatewaySecret:  strings.TrimSpace(os.Getenv("TASK_CONTAINER_GATEWAY_INTERNAL_SECRET")),
		Client:         &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *HTTPContainerStarter) Start(ctx context.Context, tenantID, workspaceID, taskID, imageID, imageURL string) error {
	if s == nil {
		return fmt.Errorf("nil starter")
	}
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8014"
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	paths := []string{
		fmt.Sprintf("/api/tenant/%s/workspace/%s/task/%s/cloud/compute/relay-to-trae/start/",
			tenantID, workspaceID, taskID),
	}
	payload := map[string]interface{}{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
	}
	if imageID != "" {
		payload["installed_image_id"] = imageID
		payload["container_image_id"] = imageID
	}
	if imageURL != "" {
		payload["image"] = imageURL
		payload["container_image_url"] = imageURL
	}
	raw, _ := json.Marshal(payload)
	var lastErr error
	for _, p := range paths {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+p, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if s.InternalSecret != "" {
			req.Header.Set("X-Internal-Secret", s.InternalSecret)
		}
		if s.GatewaySecret != "" {
			req.Header.Set("X-TaskContainerGateway-Internal-Secret", s.GatewaySecret)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 300 || resp.StatusCode == http.StatusAccepted {
			return nil
		}
		lastErr = fmt.Errorf("gateway start status=%d body=%s", resp.StatusCode, string(body))
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no gateway start path succeeded")
	}
	return lastErr
}
