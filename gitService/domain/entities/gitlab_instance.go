// Package entities defines domain entities for the gitService bounded context.
package entities

import "time"

// GitLabInstance represents a GitLab CE container deployment.
// It is the aggregate root for the infrastructure context.
type GitLabInstance struct {
	// ContainerName is the Docker container name (e.g. "gitlab").
	ContainerName string
	// Host is the external host address.
	Host string
	// HTTPPort is the GitLab web UI port.
	HTTPPort int
	// OIDCIssuer is the taskAuth OIDC provider URL used for OmniAuth.
	OIDCIssuer string
	// CreatedAt is the timestamp when this instance was first detected.
	CreatedAt time.Time
	// UpdatedAt is the last state change timestamp.
	UpdatedAt time.Time
}

// IsRunning checks if the GitLab container is operational.
func (g *GitLabInstance) IsRunning() bool {
	return g.ContainerName != "" && g.HTTPPort > 0
}

// OIDCDiscoveryURL returns the OIDC well-known discovery endpoint.
func (g *GitLabInstance) OIDCDiscoveryURL() string {
	return g.OIDCIssuer + "/.well-known/openid-configuration"
}
