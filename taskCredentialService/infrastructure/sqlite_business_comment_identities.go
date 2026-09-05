package infrastructure

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"taskCredentialService/domain"
)

// FetchRepoIdentities returns comment-level git identities when commentID is set
// and task_comments.repo_identities_json has entries; otherwise task_repo_identities.
func (r *SQLiteBusinessRepository) FetchRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	start := time.Now()
	commentID = strings.TrimSpace(commentID)
	if commentID != "" && commentID != "-" {
		fromComment, err := r.fetchCommentRepoIdentities(taskID, commentID)
		if err != nil {
			return nil, err
		}
		if len(fromComment) > 0 {
			log.Printf("[task-credential-service] repo identities fetched: task=%s comment=%s source=comment count=%d duration=%dms",
				taskID, commentID, len(fromComment), time.Since(start).Milliseconds())
			return fromComment, nil
		}
	}
	fromTask, err := r.fetchTaskTableRepoIdentities(taskID)
	if err != nil {
		return nil, err
	}
	log.Printf("[task-credential-service] repo identities fetched: task=%s comment=%s source=task_table count=%d duration=%dms",
		taskID, commentID, len(fromTask), time.Since(start).Milliseconds())
	return fromTask, nil
}

func (r *SQLiteBusinessRepository) fetchCommentRepoIdentities(taskID, commentID string) ([]domain.GitIdentitySnapshot, error) {
	if r == nil || r.taskDB == nil {
		return nil, nil
	}
	var raw string
	err := r.taskDB.QueryRow(
		`SELECT COALESCE(repo_identities_json, '') FROM task_comments WHERE id = ? AND task_id = ? LIMIT 1`,
		commentID, taskID,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetch comment repo identities: %w", err)
	}
	return r.snapshotsFromRepoIdentitiesJSON(raw, true)
}

func (r *SQLiteBusinessRepository) fetchTaskTableRepoIdentities(taskID string) ([]domain.GitIdentitySnapshot, error) {
	rows, err := r.taskDB.Query(`
		SELECT repo_url, COALESCE(git_identity_id, '')
		FROM task_repo_identities
		WHERE task_id = ?
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch repo identities: %w", err)
	}
	defer rows.Close()

	var result []domain.GitIdentitySnapshot
	for rows.Next() {
		var s domain.GitIdentitySnapshot
		var identityID string
		if err := rows.Scan(&s.RepoURL, &identityID); err != nil {
			return nil, fmt.Errorf("scan repo identity: %w", err)
		}
		s.GitIdentityID = identityID
		s.UserID, s.UserName, s.UserEmail = r.lookupGitIdentityDetails(identityID)
		result = append(result, s)
	}
	return result, nil
}

func (r *SQLiteBusinessRepository) snapshotsFromRepoIdentitiesJSON(raw string, fromComment bool) ([]domain.GitIdentitySnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var items []struct {
		RepoURL           string `json:"repo_url"`
		GitIdentityID     string `json:"git_identity_id"`
		OauthGitsite      string `json:"oauth_gitsite"`
		OauthRemoteUserID string `json:"oauth_remote_user_id"`
		OauthGrantedAt    string `json:"oauth_granted_at"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		log.Printf("[task-credential-service] invalid repo_identities_json: %v", err)
		return nil, nil
	}
	var result []domain.GitIdentitySnapshot
	for _, item := range items {
		url := strings.TrimSpace(item.RepoURL)
		gid := strings.TrimSpace(item.GitIdentityID)
		oauthSite := strings.TrimSpace(item.OauthGitsite)
		if url == "" && gid == "" && oauthSite == "" {
			continue
		}
		s := domain.GitIdentitySnapshot{
			RepoURL:           url,
			GitIdentityID:     gid,
			OauthGitsite:      oauthSite,
			OauthRemoteUserID: strings.TrimSpace(item.OauthRemoteUserID),
			OauthGrantedAt:    strings.TrimSpace(item.OauthGrantedAt),
			FromComment:       fromComment,
		}
		s.UserID, s.UserName, s.UserEmail = r.lookupGitIdentityDetails(gid)
		result = append(result, s)
	}
	return result, nil
}
