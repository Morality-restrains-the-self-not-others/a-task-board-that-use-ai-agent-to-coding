package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const DefaultTenantGitLabScope = "read_repository write_repository api read_user"

// TenantGitLabOAuthConnection is the aggregate root for a company-scoped self-hosted GitLab OAuth App.
// Invariant: at most one connection per CompanyID.
type TenantGitLabOAuthConnection struct {
	ID              string
	CompanyID       string
	BaseURL         string
	ClientID        string
	ClientSecretEnc string
	Remark          string
	RedirectURI     string
	Scope           string
	Active          bool
	Intranet        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func TenantServiceProvider(companyID string) string {
	return "tenant-" + strings.TrimSpace(companyID)
}

func TenantProviderKey(companyID string) string {
	return ProviderKey("gitlab", TenantServiceProvider(companyID))
}

// ProviderKeyForCompany is an alias used by handlers/tests.
func ProviderKeyForCompany(companyID string) string {
	return TenantProviderKey(companyID)
}

// ParseTenantCompanyID extracts company id from service_provider "tenant-{id}".
// Shared callback SP "tenant-gitlab" is not a company id.
func ParseTenantCompanyID(serviceProvider string) (string, bool) {
	sp := strings.TrimSpace(serviceProvider)
	lower := strings.ToLower(sp)
	if lower == "" || lower == "tenant-gitlab" {
		return "", false
	}
	const prefix = "tenant-"
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	id := strings.TrimSpace(sp[len(prefix):])
	if id == "" {
		return "", false
	}
	return id, true
}

// NormalizeBaseURL strips trailing slash and validates http(s) URL with host.
func NormalizeBaseURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("base_url required")
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("base_url must be absolute http(s) URL")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("base_url scheme must be http or https")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	out := strings.TrimRight(u.String(), "/")
	return out, nil
}

// DefaultRedirectURI builds the OAuth callback URL for a tenant GitLab App.
// Each tenant gets a distinct redirect URI embedding tenant-{companyID} so
// GitLab Application whitelists and authorize/exchange stay tenant-scoped.
// Empty companyID keeps the legacy shared callback for backward compatibility.
func DefaultRedirectURI(publicBase, companyID string) string {
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if base == "" {
		base = "http://127.0.0.1:8002"
	}
	id := strings.TrimSpace(companyID)
	if id == "" {
		return base + "/api/accounts/tenant-gitlab/oauth/callback/"
	}
	return base + "/api/accounts/" + TenantServiceProvider(id) + "/oauth/callback/"
}

// CanonicalTenantRedirectURI rewrites a stored (possibly legacy shared)
// redirect URI to the tenant-scoped callback path while preserving scheme/host.
func CanonicalTenantRedirectURI(stored, companyID string) string {
	id := strings.TrimSpace(companyID)
	pathSuffix := "/api/accounts/tenant-gitlab/oauth/callback/"
	if id != "" {
		pathSuffix = "/api/accounts/" + TenantServiceProvider(id) + "/oauth/callback/"
	}
	u, err := url.Parse(strings.TrimSpace(stored))
	if err == nil && u.Scheme != "" && u.Host != "" {
		return strings.TrimRight(u.Scheme+"://"+u.Host, "/") + pathSuffix
	}
	return DefaultRedirectURI("", id)
}
