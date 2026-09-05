package ports

import (
	"taskCredentialService/domain"
	"testing"
)

type signatureProbeRepo struct{}

func (signatureProbeRepo) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	return nil, nil
}
func (signatureProbeRepo) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	return nil, nil
}
func (signatureProbeRepo) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	return nil, nil
}
func (signatureProbeRepo) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	return 0, nil
}
func (signatureProbeRepo) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	return nil, nil
}

func TestBusinessDataRepositoryFetchRepoIdentitiesRequiresCommentID(t *testing.T) {
	var repo BusinessDataRepository = signatureProbeRepo{}
	got, err := repo.FetchRepoIdentities("task-1", "cmt-1")
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if got != nil {
		t.Fatalf("probe must return nil, got %+v", got)
	}
}
