package gatewayauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"confload"
	"tracelog"
)

const (
	HeaderGatewayVerified = "X-Gateway-Auth-Verified"
	HeaderGatewaySecret   = "X-TaskGateway-Internal-Secret"
	HeaderUserID          = "X-User-Id"
	HeaderAuthUserID      = "X-Auth-User-Id"
	HeaderAuthTenantID    = "X-Auth-Tenant-Id"
	HeaderCommentID       = "X-Comment-Id"
)

var defaultHTTPClient = &http.Client{Timeout: 10 * time.Second}

// UserFromGatewayHeaders returns X-User-Id when APISIX forward-auth headers are valid.
func UserFromGatewayHeaders(r *http.Request, gatewayInternalSecret string) string {
	secret := strings.TrimSpace(gatewayInternalSecret)
	if secret == "" {
		return ""
	}
	if r.Header.Get(HeaderGatewayVerified) != "1" {
		return ""
	}
	if strings.TrimSpace(r.Header.Get(HeaderGatewaySecret)) != secret {
		return ""
	}
	return strings.TrimSpace(r.Header.Get(HeaderUserID))
}

// ApplyGatewayUser sets X-Auth-User-Id from gateway headers when valid.
func ApplyGatewayUser(r *http.Request, gatewayInternalSecret string) bool {
	if userID := UserFromGatewayHeaders(r, gatewayInternalSecret); userID != "" {
		r.Header.Set(HeaderAuthUserID, userID)
		return true
	}
	return false
}

// LoadGatewayInternalSecret reads gatewayInternalSecret from task-gateway config.
func LoadGatewayInternalSecret(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("TASK_GATEWAY_INTERNAL_SECRET")); v != "" {
		return v
	}
	var gw struct {
		GatewayInternalSecret string `yaml:"gatewayInternalSecret"`
	}
	if err := confload.ReadAppConfig(repoRoot, "task-gateway", &gw); err != nil {
		return ""
	}
	return strings.TrimSpace(gw.GatewayInternalSecret)
}

// BearerOrTokenFromAuthHeader extracts the credential from Authorization.
func BearerOrTokenFromAuthHeader(authHeader string) string {
	auth := strings.TrimSpace(authHeader)
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if strings.HasPrefix(auth, "Token ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Token "))
	}
	return auth
}

// VerifyWithTaskAuth calls taskAuth /api/auth/verify with the request Authorization header.
func VerifyWithTaskAuth(r *http.Request, taskAuthURL string) (userID string, tenantID string, err error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", "", fmt.Errorf("missing Authorization header")
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(taskAuthURL, "/")+"/api/auth/verify", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", authHeader)
	tracelog.ApplyOutboundHeaders(req, r.Context())
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("auth service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("auth verify failed: %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(result.UserID) == "" {
		return "", "", fmt.Errorf("auth verify returned empty user_id")
	}
	return result.UserID, result.TenantID, nil
}
