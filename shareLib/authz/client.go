package authz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CheckPermRequest is the PDP payload: does userID hold perm in companyID?
type CheckPermRequest struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id,omitempty"`
	PermCode  string `json:"perm_code"`
	// ResourceType/ResourceID enable group-resource fallback checks.
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
}

// CheckPermResponse is the PDP verdict.
type CheckPermResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Client is a minimal PDP HTTP fallback client (used only when gateway
// headers are absent, e.g. internal service-to-service calls and tests).
type Client struct {
	// BaseURL is the taskAuth PDP endpoint base, e.g. "http://taskAuth:8003".
	BaseURL string
	// InternalSecret is sent as X-Internal-Secret.
	InternalSecret string
	HTTP           *http.Client
}

// NewClient builds a PDP fallback client.
func NewClient(baseURL, internalSecret string) *Client {
	return &Client{
		BaseURL:        baseURL,
		InternalSecret: internalSecret,
		HTTP:           &http.Client{Timeout: 3 * time.Second},
	}
}

// CheckPermission calls POST /api/internal/authz/check on the PDP.
func (c *Client) CheckPermission(req CheckPermRequest) (bool, error) {
	if c == nil || c.BaseURL == "" {
		return false, fmt.Errorf("authz client not configured")
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/internal/authz/check", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.InternalSecret != "" {
		httpReq.Header.Set("X-Internal-Secret", c.InternalSecret)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("pdp unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return false, fmt.Errorf("pdp returned %d: %s", resp.StatusCode, raw)
	}
	var verdict CheckPermResponse
	if err := json.NewDecoder(resp.Body).Decode(&verdict); err != nil {
		return false, err
	}
	return verdict.Allowed, nil
}
