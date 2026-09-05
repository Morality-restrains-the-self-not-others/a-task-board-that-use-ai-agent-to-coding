package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"taskCredentialService/domain"
)

// HTTPBusinessRepository implements ports.BusinessDataRepository via owner-service HTTP APIs.
// task-task-service: container-snapshot; git-identities enrichment was retired with saas-backend (OPT-049).
type HTTPBusinessRepository struct {
	taskServiceBase string
	internalSecret  string
	client          *http.Client

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	snap *containerSnapshotDTO
	at   time.Time
}

type containerSnapshotDTO struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	WorkspaceID      string `json:"workspace_id"`
	TenantID         string `json:"tenant_id"`
	AutoRun          bool   `json:"auto_run"`
	OwnerID          string `json:"owner_id"`
	InstalledImageID string `json:"installed_image_id"`
	TargetBranch     string `json:"target_branch"`
	BranchStrategy   *struct {
		WorkBranchName        string `json:"work_branch_name"`
		MergeTargetBranchName string `json:"merge_target_branch_name"`
		TargetBranchName      string `json:"target_branch_name"`
	} `json:"branch_strategy"`
	Projects []struct {
		ProjectID            string   `json:"project_id"`
		ProjectName          string   `json:"project_name"`
		GitRepos             []string `json:"git_repos"`
		AutoCloneNestedRepos *bool    `json:"auto_clone_nested_repos"`
		GitRepoEntries       []struct {
			URL        string `json:"url"`
			CloneAlias string `json:"clone_alias"`
		} `json:"git_repo_entries"`
		RepoBranches []struct {
			GitRepo      string `json:"git_repo"`
			BaseBranch   string `json:"base_branch"`
			TargetBranch string `json:"target_branch"`
		} `json:"repo_branches"`
	} `json:"projects"`
	RepoIdentities []struct {
		RepoURL           string `json:"repo_url"`
		GitIdentityID     string `json:"git_identity_id"`
		UserID            string `json:"user_id"`
		UserName          string `json:"user_name"`
		UserEmail         string `json:"user_email"`
		OauthGitsite      string `json:"oauth_gitsite"`
		OauthRemoteUserID string `json:"oauth_remote_user_id"`
		OauthGrantedAt    string `json:"oauth_granted_at"`
	} `json:"repo_identities"`
}

// NewHTTPBusinessRepository wires HTTP client for task-service container snapshots.
func NewHTTPBusinessRepository(taskServiceBase, internalSecret string, timeoutSeconds int) *HTTPBusinessRepository {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	return &HTTPBusinessRepository{
		taskServiceBase: strings.TrimRight(strings.TrimSpace(taskServiceBase), "/"),
		internalSecret:  strings.TrimSpace(internalSecret),
		client: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
		cache: map[string]cacheEntry{},
	}
}

func (r *HTTPBusinessRepository) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	snap, err := r.loadSnapshot(taskID, commentID)
	if err != nil {
		return nil, err
	}
	out := &domain.TaskSnapshot{
		ID:               snap.ID,
		Title:            snap.Title,
		Description:      snap.Description,
		TargetBranch:     snap.TargetBranch,
		AutoRun:          snap.AutoRun,
		InstalledImageID: strings.TrimSpace(snap.InstalledImageID),
		CompanyID:        snap.TenantID,
		WorkspaceID:      snap.WorkspaceID,
	}
	if snap.BranchStrategy != nil {
		bs := &domain.BranchStrategySnapshot{
			WorkBranchName:        strings.TrimSpace(snap.BranchStrategy.WorkBranchName),
			MergeTargetBranchName: strings.TrimSpace(snap.BranchStrategy.MergeTargetBranchName),
			TargetBranchName:      strings.TrimSpace(snap.BranchStrategy.TargetBranchName),
		}
		out.BranchStrategy = bs
		if out.TargetBranch == "" {
			if bs.WorkBranchName != "" {
				out.TargetBranch = bs.WorkBranchName
			} else if bs.TargetBranchName != "" {
				out.TargetBranch = bs.TargetBranchName
			}
		}
	}
	return out, nil
}

func (r *HTTPBusinessRepository) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	snap, err := r.loadSnapshot(taskID, commentID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaskRepoSnapshot, 0, len(snap.Projects))
	for _, p := range snap.Projects {
		autoClone := true
		if p.AutoCloneNestedRepos != nil {
			autoClone = *p.AutoCloneNestedRepos
		}
		repo := domain.TaskRepoSnapshot{
			ProjectID:            p.ProjectID,
			ProjectName:          p.ProjectName,
			RepoURLs:             append([]string{}, p.GitRepos...),
			AutoCloneNestedRepos: autoClone,
		}
		for _, e := range p.GitRepoEntries {
			url := strings.TrimSpace(e.URL)
			if url == "" {
				continue
			}
			repo.RepoEntries = append(repo.RepoEntries, domain.RepoCloneEntry{
				URL:        url,
				CloneAlias: strings.TrimSpace(e.CloneAlias),
			})
		}
		if len(repo.RepoEntries) == 0 {
			for _, u := range repo.RepoURLs {
				repo.RepoEntries = append(repo.RepoEntries, domain.RepoCloneEntry{URL: u})
			}
		}
		for _, b := range p.RepoBranches {
			repo.RepoBranches = append(repo.RepoBranches, domain.RepoBranchEntry{
				GitRepo:      b.GitRepo,
				BaseBranch:   b.BaseBranch,
				TargetBranch: b.TargetBranch,
			})
		}
		out = append(out, repo)
	}
	return out, nil
}

// FetchRepoIdentities maps container-snapshot repo_identities (comment-scoped when commentID is set).
func (r *HTTPBusinessRepository) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	snap, err := r.loadSnapshot(taskID, commentID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return nil, nil
	}
	fromComment := strings.TrimSpace(commentID) != "" && commentID != "-"
	out := make([]domain.GitIdentitySnapshot, 0, len(snap.RepoIdentities))
	for _, row := range snap.RepoIdentities {
		url := strings.TrimSpace(row.RepoURL)
		gid := strings.TrimSpace(row.GitIdentityID)
		oauthSite := strings.TrimSpace(row.OauthGitsite)
		if url == "" && gid == "" && oauthSite == "" {
			continue
		}
		s := domain.GitIdentitySnapshot{
			RepoURL:           url,
			GitIdentityID:     gid,
			UserName:          strings.TrimSpace(row.UserName),
			UserEmail:         strings.TrimSpace(row.UserEmail),
			OauthGitsite:      oauthSite,
			OauthRemoteUserID: strings.TrimSpace(row.OauthRemoteUserID),
			OauthGrantedAt:    strings.TrimSpace(row.OauthGrantedAt),
			FromComment:       fromComment && (gid != "" || oauthSite != ""),
		}
		if uid := strings.TrimSpace(row.UserID); uid != "" {
			if parsed, convErr := strconv.ParseInt(uid, 10, 64); convErr == nil {
				s.UserID = parsed
			}
		}
		out = append(out, s)
	}
	log.Printf("[task-credential-service] http repo identities fetched: task=%s comment=%s count=%d from_comment=%v",
		taskID, commentID, len(out), fromComment)
	return out, nil
}

// FetchCommentCreatedByUserID resolves the comment author via the owner-service
// internal comment-created-by API (OPT-20260820-021). Previously this returned 0
// on the HTTP fallback, which made nested-repo discovery fall back to task-level
// or anonymous git users when the direct MySQL path was unavailable.
func (r *HTTPBusinessRepository) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	commentID = strings.TrimSpace(commentID)
	if r == nil || commentID == "" {
		return 0, nil
	}
	u := fmt.Sprintf("%s/api/internal/comments/%s/created-by", r.taskServiceBase, url.PathEscape(commentID))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, fmt.Errorf("build comment-created-by request: %w", err)
	}
	if r.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", r.internalSecret)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("comment-created-by request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[task-credential-service] comment author not found comment=%s", commentID)
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("comment-created-by returned %d", resp.StatusCode)
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode comment-created-by: %w", err)
	}
	uid := parseUserID(body.UserID)
	log.Printf("[task-credential-service] comment author fetched comment=%s user=%d", commentID, uid)
	return uid, nil
}

// FetchUserGitIdentities is unavailable on the HTTP fallback (OPT-049).
func (r *HTTPBusinessRepository) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	return nil, nil
}

func (r *HTTPBusinessRepository) loadSnapshot(taskID, commentID string) (*containerSnapshotDTO, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id required")
	}
	commentID = strings.TrimSpace(commentID)
	cacheKey := taskID
	if commentID != "" && commentID != "-" {
		cacheKey = taskID + "|" + commentID
	}
	r.mu.Lock()
	if cached, ok := r.cache[cacheKey]; ok && time.Since(cached.at) < 3*time.Second {
		r.mu.Unlock()
		return cached.snap, nil
	}
	r.mu.Unlock()

	start := time.Now()
	base := fmt.Sprintf("%s/api/internal/tasks/%s/container-snapshot", r.taskServiceBase, taskID)
	if commentID != "" && commentID != "-" {
		base = base + "?comment_id=" + url.QueryEscape(commentID)
	}
	req, err := http.NewRequest(http.MethodGet, base, nil)
	if err != nil {
		return nil, fmt.Errorf("build container-snapshot request: %w", err)
	}
	if r.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", r.internalSecret)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("container-snapshot request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("container-snapshot returned %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var snap containerSnapshotDTO
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, fmt.Errorf("decode container-snapshot: %w", err)
	}
	if strings.TrimSpace(snap.ID) == "" {
		snap.ID = taskID
	}
	r.mu.Lock()
	r.cache[cacheKey] = cacheEntry{snap: &snap, at: time.Now()}
	r.mu.Unlock()
	log.Printf("[task-credential-service] container-snapshot fetched: task=%s comment=%s projects=%d identities=%d duration=%dms",
		taskID, commentID, len(snap.Projects), len(snap.RepoIdentities), time.Since(start).Milliseconds())
	return &snap, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// parseUserID parses a string user id (comment created_by_id / git identity user_id) into int64.
func parseUserID(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
