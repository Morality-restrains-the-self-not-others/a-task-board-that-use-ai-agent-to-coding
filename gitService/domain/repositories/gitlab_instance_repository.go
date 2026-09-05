// Package repositories defines abstract storage contracts (ABC).
package repositories

import "gitService/domain/entities"

// GitLabInstanceRepository is the abstract persistence contract for the GitLabInstance aggregate root.
// Concrete implementations live in the infrastructure layer.
type GitLabInstanceRepository interface {
	// FindByContainerName retrieves the instance by Docker container name.
	FindByContainerName(name string) (*entities.GitLabInstance, error)

	// Save persists the instance state.
	Save(instance *entities.GitLabInstance) error
}
