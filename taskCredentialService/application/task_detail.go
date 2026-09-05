package application

import (
	"fmt"
	"log"
	"strings"

	"taskCredentialService/domain"
	"taskCredentialService/ports"
)

const defaultIdleRecycleMinutes = 5

// TaskDetailService fetches task metadata for container consumption.
type TaskDetailService struct {
	businessRepo  ports.BusinessDataRepository
	nestedFetcher NestedGitReposFetcher
	policyFetcher ports.WorkspaceMachinePolicyFetcher
}

func NewTaskDetailService(businessRepo ports.BusinessDataRepository) *TaskDetailService {
	return &TaskDetailService{businessRepo: businessRepo}
}

// WithNestedFetcher enables merging latest nested git repos into project_repos.
func (s *TaskDetailService) WithNestedFetcher(f NestedGitReposFetcher) *TaskDetailService {
	s.nestedFetcher = f
	return s
}

// WithPolicyFetcher wires Cloud-owned workspace idle recycle minutes (no DB hop).
func (s *TaskDetailService) WithPolicyFetcher(f ports.WorkspaceMachinePolicyFetcher) *TaskDetailService {
	s.policyFetcher = f
	return s
}

// FetchTaskDetail returns task snapshot + project_repos for a validated container request.
// 契约见 docs/skills/saas-container/saas-machine-container.md（须含 project_repos，供 onlineServiceJS collectRepoUrls）。
func (s *TaskDetailService) FetchTaskDetail(taskID, commentID string) (*domain.ContainerTaskDetail, error) {
	snap, err := s.businessRepo.FetchTaskSnapshot(taskID, commentID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	repos, err := s.businessRepo.FetchTaskRepos(taskID, commentID)
	if err != nil {
		return nil, fmt.Errorf("fetch task repos: %w", err)
	}
	if repos == nil {
		repos = []domain.TaskRepoSnapshot{}
	}
	idents, err := s.businessRepo.FetchRepoIdentities(taskID, commentID)
	if err != nil {
		return nil, fmt.Errorf("fetch repo identities: %w", err)
	}
	authorID := lookupCommentAuthorUserID(s.businessRepo, commentID)
	authorIdents := lookupUserGitIdentities(s.businessRepo, authorID)
	repos, idents, discoveryUserID := EnrichReposForCommentAuthor(
		authorID, authorIdents, idents, repos, snap.CompanyID, s.nestedFetcher,
	)
	log.Printf("[task-credential-service] task-detail nested enrich task=%s comment=%s author=%d discovery_user=%d identities=%d projects=%d repos=%d",
		taskID, commentID, authorID, discoveryUserID, len(idents), len(repos), len(collectRepoURLs(repos)))
	repoGitIdentities := make([]domain.RepoGitIdentityDTO, 0, len(idents))
	for _, ident := range idents {
		name := strings.TrimSpace(ident.UserName)
		email := strings.TrimSpace(ident.UserEmail)
		if name == "" || email == "" {
			continue
		}
		repoGitIdentities = append(repoGitIdentities, domain.RepoGitIdentityDTO{
			RepoURL:    ident.RepoURL,
			UserName:   name,
			UserEmail:  email,
			IdentityID: ident.GitIdentityID,
		})
	}
	detail := &domain.ContainerTaskDetail{
		CompanyID:         snap.CompanyID,
		WorkspaceID:       snap.WorkspaceID,
		TaskID:            snap.ID,
		Task:              snap,
		ProjectRepos:      repos,
		RepoGitIdentities: repoGitIdentities,
	}
	s.AttachIdlePolicy(detail, commentID)
	return detail, nil
}

// AttachIdlePolicy fills idle_recycle_minutes from Cloud (default 5). Path IDs win.
func (s *TaskDetailService) AttachIdlePolicy(detail *domain.ContainerTaskDetail, commentID string) {
	if detail == nil {
		return
	}
	minutes := defaultIdleRecycleMinutes
	var sts map[string]interface{}
	if s.policyFetcher != nil && strings.TrimSpace(detail.CompanyID) != "" && strings.TrimSpace(detail.WorkspaceID) != "" {
		p, err := s.policyFetcher.FetchWorkspaceMachinePolicy(detail.CompanyID, detail.WorkspaceID, detail.TaskID, commentID)
		if err != nil {
			log.Printf("[task-credential-service] workspace policy fetch failed company=%s workspace=%s err=%v; default minutes=%d",
				detail.CompanyID, detail.WorkspaceID, err, defaultIdleRecycleMinutes)
		} else {
			minutes = p.IdleRecycleMinutes
			sts = p.MachineReleaseSTS
		}
	}
	detail.IdleRecycleMinutes = minutes
	detail.InstructionIdle = domain.InstructionIdlePolicy{
		Enabled: minutes > 0,
		Minutes: minutes,
	}
	if len(sts) > 0 {
		detail.MachineReleaseSTS = sts
	} else {
		detail.MachineReleaseSTS = nil
	}
}
