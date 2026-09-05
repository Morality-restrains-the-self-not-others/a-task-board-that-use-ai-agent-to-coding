package taskstatuschanged

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// BusyForeignBinding is another task still busy on the same instance.
type BusyForeignBinding struct {
	TaskID    string
	ServerURL string
}

// InstanceMigrator migrates foreign containers and marks terminal hard-release.
type InstanceMigrator interface {
	ListBusyForeign(ctx context.Context, companyID int64, workspaceID, instanceID, excludeTaskID string) ([]BusyForeignBinding, error)
	MigrateOff(ctx context.Context, tenantID, workspaceID, ownerTaskID, fromInstanceID string) error
	ClearIdleSiblings(ctx context.Context, tenantID, workspaceID, instanceID, excludeTaskID string) error
	SetTerminalReleasedFlag(ctx context.Context, tenantID, workspaceID, taskID string) error
	MarkTerminalReleased(ctx context.Context, tenantID, workspaceID, taskID string) error
}

type noopMigrator struct{}

func (noopMigrator) ListBusyForeign(context.Context, int64, string, string, string) ([]BusyForeignBinding, error) {
	return nil, nil
}
func (noopMigrator) MigrateOff(context.Context, string, string, string, string) error { return nil }
func (noopMigrator) ClearIdleSiblings(context.Context, string, string, string, string) error {
	return nil
}
func (noopMigrator) SetTerminalReleasedFlag(context.Context, string, string, string) error {
	return nil
}
func (noopMigrator) MarkTerminalReleased(context.Context, string, string, string) error {
	return nil
}

// HTTPMigrator calls taskCloudService internal compute APIs.
type HTTPMigrator struct {
	BaseURL        string
	InternalSecret string
	Client         *http.Client
}

// NewHTTPMigrator builds a migrator against TASK_CLOUD_SERVICE_BASE_URL (default :8018).
func NewHTTPMigrator() *HTTPMigrator {
	base := strings.TrimSpace(os.Getenv("TASK_CLOUD_SERVICE_BASE_URL"))
	if base == "" {
		base = "http://127.0.0.1:8018"
	}
	secret := strings.TrimSpace(os.Getenv("TASK_CLOUD_INTERNAL_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("SHARED_INTERNAL_SECRET"))
	}
	return &HTTPMigrator{
		BaseURL:        strings.TrimRight(base, "/"),
		InternalSecret: secret,
		Client:         &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *HTTPMigrator) client() *http.Client {
	if m.Client != nil {
		return m.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (m *HTTPMigrator) setSecret(req *http.Request) {
	if m.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", m.InternalSecret)
	}
}

func (m *HTTPMigrator) doJSON(ctx context.Context, method, path string, payload map[string]string) error {
	var body io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		body = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, m.BaseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	m.setSecret(req)
	resp, err := m.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	return nil
}

func (m *HTTPMigrator) ListBusyForeign(ctx context.Context, companyID int64, workspaceID, instanceID, excludeTaskID string) ([]BusyForeignBinding, error) {
	q := url.Values{}
	q.Set("company_id", fmt.Sprint(companyID))
	q.Set("workspace_id", workspaceID)
	q.Set("instance_id", instanceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		m.BaseURL+"/api/internal/cloud/compute/instance-bindings/?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	m.setSecret(req)
	resp, err := m.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("instance-bindings status=%d body=%s", resp.StatusCode, string(raw))
	}
	var body struct {
		Bindings []struct {
			TaskID            string `json:"task_id"`
			ServerURL         string `json:"server_url"`
			LastRuntimeStatus string `json:"last_runtime_status"`
		} `json:"bindings"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	var out []BusyForeignBinding
	for _, b := range body.Bindings {
		if strings.TrimSpace(b.TaskID) == "" || b.TaskID == excludeTaskID {
			continue
		}
		serverURL := strings.TrimSpace(b.ServerURL)
		started := strings.EqualFold(strings.TrimSpace(b.LastRuntimeStatus), "Running")
		if serverURL == "" && !started {
			continue
		}
		// Busy = has container URL; started-without-URL is idle (cleared separately).
		if serverURL == "" {
			continue
		}
		out = append(out, BusyForeignBinding{TaskID: b.TaskID, ServerURL: serverURL})
	}
	return out, nil
}

func (m *HTTPMigrator) MigrateOff(ctx context.Context, tenantID, workspaceID, ownerTaskID, fromInstanceID string) error {
	return m.doJSON(ctx, http.MethodPost, "/api/internal/cloud/compute/migrate-container-off-instance/", map[string]string{
		"tenant_id":        tenantID,
		"workspace_id":     workspaceID,
		"owner_task_id":    ownerTaskID,
		"from_instance_id": fromInstanceID,
	})
}

func (m *HTTPMigrator) ClearIdleSiblings(ctx context.Context, tenantID, workspaceID, instanceID, excludeTaskID string) error {
	return m.doJSON(ctx, http.MethodPost, "/api/internal/cloud/compute/clear-idle-bindings-on-instance/", map[string]string{
		"tenant_id":       tenantID,
		"workspace_id":    workspaceID,
		"instance_id":     instanceID,
		"exclude_task_id": excludeTaskID,
	})
}

func (m *HTTPMigrator) SetTerminalReleasedFlag(ctx context.Context, tenantID, workspaceID, taskID string) error {
	return m.doJSON(ctx, http.MethodPost, "/api/internal/cloud/compute/set-terminal-released-flag/", map[string]string{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
	})
}

func (m *HTTPMigrator) MarkTerminalReleased(ctx context.Context, tenantID, workspaceID, taskID string) error {
	return m.doJSON(ctx, http.MethodPost, "/api/internal/cloud/compute/mark-terminal-released/", map[string]string{
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      taskID,
	})
}
