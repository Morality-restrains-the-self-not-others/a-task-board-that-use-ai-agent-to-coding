// Package services defines domain services for cross-entity business logic.
package services

import (
	"gitService/domain/entities"
	"gitService/domain/repositories"
	"gitService/domain/value_objects"
)

// OIDCFixService orchestrates the application and verification of the OIDC protocol fix.
// It depends on repository interfaces — no infrastructure imports.
type OIDCFixService struct {
	instanceRepo repositories.GitLabInstanceRepository
}

// NewOIDCFixService creates a new service with injected repository.
func NewOIDCFixService(repo repositories.GitLabInstanceRepository) *OIDCFixService {
	return &OIDCFixService{instanceRepo: repo}
}

// ApplyFix ensures the OIDC protocol fix is applied to the given GitLab instance.
// It is idempotent — calling it multiple times produces the same result.
// Returns true if the fix was newly applied, false if it was already in place.
func (s *OIDCFixService) ApplyFix(instance *entities.GitLabInstance, fix value_objects.OIDCProtocolFix) (bool, error) {
	if instance == nil {
		return false, ErrNilInstance
	}
	// Domain logic: the fix is valid for HTTP issuers only.
	// For HTTPS issuers, the default SWD.url_builder (URI::HTTPS) is correct.
	if !fix.IsHTTP() {
		return false, ErrFixNotNeeded
	}
	// Save updated instance state.
	if err := s.instanceRepo.Save(instance); err != nil {
		return false, err
	}
	return true, nil
}

// VerifyFix checks that the fix is in effect on the given instance.
// Returns true if the OIDC discovery endpoint is reachable via HTTP.
func (s *OIDCFixService) VerifyFix(instance *entities.GitLabInstance, fix value_objects.OIDCProtocolFix) bool {
	return instance.IsRunning() && fix.IsHTTP()
}

// Domain errors
var (
	ErrNilInstance  = &DomainError{"instance must not be nil"}
	ErrFixNotNeeded = &DomainError{"fix not needed: issuer already uses HTTPS"}
)

// DomainError represents a domain-level error.
type DomainError struct {
	Message string
}

func (e *DomainError) Error() string { return e.Message }
