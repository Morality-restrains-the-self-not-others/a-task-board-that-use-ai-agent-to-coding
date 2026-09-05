package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	TenantGitLabOidcClientPrefix = "gitlab-tenant-"
	PlatformGitLabOidcPrefix     = "gitlab-git-service"
	TenantGitLabOidcPurpose      = "tenant_gitlab_sso"
	TenantGitLabOidcManagedBy    = "tenant"
	TenantGitLabOidcRegionKey    = "settings.gitlab.main"
	TenantGitLabOidcCallbackPath = "/users/auth/openid_connect/callback"
)

var (
	ErrNotTenantGitLabOidcClient = errors.New("not a tenant gitlab oidc client")
	ErrInvalidRedirectScheme     = errors.New("redirect_uri scheme must be http or https")
	ErrInvalidRedirectPath       = errors.New("redirect_uri path must be /users/auth/openid_connect/callback")
	ErrEmptyCompanyID            = errors.New("company_id required")
	ErrEmptyBaseURL              = errors.New("base_url required")
	ErrBaseURLMismatch           = errors.New("base_url must match saved gitlab connection")
	ErrAuthorizeMembershipDenied = errors.New("access_denied")
	ErrOidcPathTenantMismatch    = errors.New("oidc path tenant does not match client")
)

// TenantGitLabOidcClientID returns the stable client_id for a tenant.
func TenantGitLabOidcClientID(companyID string) (string, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return "", ErrEmptyCompanyID
	}
	return TenantGitLabOidcClientPrefix + cid, nil
}

// OwnerCompanyIDFromClientID parses gitlab-tenant-{company_id}.
func OwnerCompanyIDFromClientID(clientID string) (string, error) {
	id := strings.TrimSpace(clientID)
	if !strings.HasPrefix(id, TenantGitLabOidcClientPrefix) {
		return "", ErrNotTenantGitLabOidcClient
	}
	rest := strings.TrimPrefix(id, TenantGitLabOidcClientPrefix)
	if rest == "" || strings.HasPrefix(id, PlatformGitLabOidcPrefix) {
		return "", ErrNotTenantGitLabOidcClient
	}
	return rest, nil
}

// IsTenantGitLabOidcClient reports whether this OIDC client is a tenant SSO RP.
func IsTenantGitLabOidcClient(clientID, managedBy string) bool {
	if strings.EqualFold(strings.TrimSpace(managedBy), TenantGitLabOidcManagedBy) {
		return true
	}
	_, err := OwnerCompanyIDFromClientID(clientID)
	return err == nil
}

// RegionGateApplies is true only for platform gitService clients.
func RegionGateApplies(clientID string) bool {
	return strings.HasPrefix(strings.TrimSpace(clientID), PlatformGitLabOidcPrefix)
}

// NormalizeBaseURL trims trailing slashes for comparison.
func NormalizeBaseURL(raw string) (string, error) {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return "", ErrEmptyBaseURL
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base_url")
	}
	return s, nil
}

// RedirectURIFromBaseURL builds GitLab OmniAuth OIDC callback URL.
func RedirectURIFromBaseURL(baseURL string) (string, error) {
	base, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	return base + TenantGitLabOidcCallbackPath, nil
}

// ValidateProductionRedirectURI requires http or https plus the OmniAuth callback path.
// HTTP is allowed so Path A base_url can match GitLab external_url when the instance has no TLS.
func ValidateProductionRedirectURI(redirectURI string) error {
	u, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid redirect_uri")
	}
	if u.Path != TenantGitLabOidcCallbackPath {
		return ErrInvalidRedirectPath
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return ErrInvalidRedirectScheme
	}
	return nil
}

// RedirectURIIsHTTP reports whether the OmniAuth callback uses http (plaintext).
func RedirectURIIsHTTP(redirectURI string) bool {
	u, err := url.Parse(strings.TrimSpace(redirectURI))
	return err == nil && strings.EqualFold(u.Scheme, "http")
}

// AuthorizeMembershipInput is the snapshot for the tenant SSO gate.
type AuthorizeMembershipInput struct {
	UserID          string
	OwnerCompanyID  string
	IsMember        bool
	MembershipError error
}

// DecideAuthorizeMembership fail-closes unless the user is an in-tenant member.
// Superusers do not bypass; lookup errors deny.
func DecideAuthorizeMembership(in AuthorizeMembershipInput) error {
	if strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.OwnerCompanyID) == "" {
		return ErrAuthorizeMembershipDenied
	}
	if in.MembershipError != nil {
		return ErrAuthorizeMembershipDenied
	}
	if !in.IsMember {
		return ErrAuthorizeMembershipDenied
	}
	return nil
}

// TenantOidcProtocolBase is the tenant-scoped OIDC prefix {issuer}/api/oidc/{tenantID}.
func TenantOidcProtocolBase(issuer, tenantID string) string {
	iss := strings.TrimRight(strings.TrimSpace(issuer), "/")
	tid := strings.TrimSpace(tenantID)
	return iss + "/api/oidc/" + tid
}

// CheckOidcPathTenant is a no-op when pathTenantID is empty (global /api/oidc/*).
// Otherwise ownerCompanyID must equal the path tenant (future shard key).
func CheckOidcPathTenant(pathTenantID, ownerCompanyID string) error {
	pt := strings.TrimSpace(pathTenantID)
	if pt == "" {
		return nil
	}
	if strings.TrimSpace(ownerCompanyID) != pt {
		return ErrOidcPathTenantMismatch
	}
	return nil
}

// OmniAuthSecretPlaceholder is the gitlab.rb client_options secret value shown in
// the customer snippet. It is a placeholder only — the real one-time client_secret
// is injected client-side (never stored or returned by the API).
const OmniAuthSecretPlaceholder = "'<PASTE_CLIENT_SECRET>'"

// OmniAuthSnippet is the customer gitlab.rb fragment (secret placeholder only).
// Protocol endpoints include tenantID so GitLab talks to /api/oidc/{tid}/* (ADR-0044).
// discovery is false: global well-known would otherwise override these URLs.
func OmniAuthSnippet(issuer, clientID, redirectURI, tenantID string) string {
	iss := strings.TrimRight(strings.TrimSpace(issuer), "/")
	base := TenantOidcProtocolBase(iss, tenantID)
	return fmt.Sprintf(`gitlab_rails['omniauth_allow_single_sign_on'] = ['openid_connect']
gitlab_rails['omniauth_block_auto_created_users'] = false
gitlab_rails['omniauth_auto_link_user'] = ['openid_connect']
gitlab_rails['omniauth_providers'] = [
  {
    name: 'openid_connect',
    label: 'Daydaymoney SSO',
    args: {
      name: 'openid_connect',
      scope: ['openid', 'profile', 'email'],
      response_type: 'code',
      issuer: %q,
      discovery: false,
      client_auth_method: 'basic',
      uid_field: 'sub',
      client_options: {
        identifier: %q,
        secret: %s,
        redirect_uri: %q,
        authorization_endpoint: %q,
        token_endpoint: %q,
        userinfo_endpoint: %q,
        jwks_uri: %q
      }
    }
  }
]
`, iss, clientID, OmniAuthSecretPlaceholder, redirectURI,
		base+"/authorize",
		base+"/token",
		base+"/userinfo",
		base+"/jwks")
}
