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

	"taskCredentialService/ports"
)

// CloudPolicyHTTPClient reads workspace machine policy from taskCloudService internal API.
type CloudPolicyHTTPClient struct {
	baseURL        string
	internalSecret string
	client         *http.Client
}

func NewCloudPolicyHTTPClient(cloudServiceBase, internalSecret string, timeoutSeconds int) *CloudPolicyHTTPClient {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 5
	}
	return &CloudPolicyHTTPClient{
		baseURL:        strings.TrimRight(strings.TrimSpace(cloudServiceBase), "/"),
		internalSecret: strings.TrimSpace(internalSecret),
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
	}
}

func (c *CloudPolicyHTTPClient) FetchWorkspaceMachinePolicy(companyID, workspaceID, taskID, commentID string) (ports.WorkspaceMachinePolicy, error) {
	out := ports.WorkspaceMachinePolicy{IdleRecycleMinutes: 30}
	if c == nil || c.baseURL == "" {
		return out, fmt.Errorf("task cloud service base not configured")
	}
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	if companyID == "" || workspaceID == "" {
		return out, fmt.Errorf("company_id and workspace_id required")
	}
	u, err := url.Parse(c.baseURL + "/api/internal/cloud/workspace-machine-policy/")
	if err != nil {
		return out, err
	}
	q := u.Query()
	q.Set("company_id", companyID)
	q.Set("workspace_id", workspaceID)
	if strings.TrimSpace(taskID) != "" {
		q.Set("task_id", strings.TrimSpace(taskID))
	}
	if strings.TrimSpace(commentID) != "" {
		q.Set("comment_id", strings.TrimSpace(commentID))
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return out, err
	}
	if c.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", c.internalSecret)
	}
	started := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[task-credential-service] cloud policy GET failed company=%s duration_ms=%d err=%v",
			companyID, time.Since(started).Milliseconds(), err)
		return out, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, err
	}
	log.Printf("[task-credential-service] cloud policy GET company=%s status=%d duration_ms=%d",
		companyID, resp.StatusCode, time.Since(started).Milliseconds())
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("cloud policy status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		IdleRecycleMinutes int                    `json:"idle_recycle_minutes"`
		MachineReleaseSTS  map[string]interface{} `json:"machine_release_sts"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return out, fmt.Errorf("decode cloud policy: %w", err)
	}
	out.IdleRecycleMinutes = parsed.IdleRecycleMinutes
	out.MachineReleaseSTS = parsed.MachineReleaseSTS
	return out, nil
}
