package infrastructure

import (
	"testing"
)

func TestSQLiteBusinessRepository_FetchCommentAuthorAndUserIdentities(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE task_comments (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL DEFAULT '',
			created_by_id VARCHAR(64) NOT NULL DEFAULT '',
			content TEXT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE task_git_identities (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL DEFAULT '',
			git_user_name VARCHAR(255) NOT NULL DEFAULT '',
			git_user_email VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO task_comments(id, task_id, created_by_id, content)
		VALUES ('cmt_ann', 'task_1', '99', 'hello');
		INSERT INTO task_git_identities(id, user_id, git_user_name, git_user_email)
		VALUES ('gid_ann', '99', 'Ann', 'ann@example.com');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewSQLiteBusinessRepository(db, nil)
	uid, err := repo.FetchCommentCreatedByUserID("cmt_ann")
	if err != nil {
		t.Fatalf("FetchCommentCreatedByUserID: %v", err)
	}
	if uid != 99 {
		t.Fatalf("author=%d want 99", uid)
	}
	missing, err := repo.FetchCommentCreatedByUserID("cmt_missing")
	if err != nil {
		t.Fatalf("missing comment should not error: %v", err)
	}
	if missing != 0 {
		t.Fatalf("missing author=%d want 0", missing)
	}

	idents, err := repo.FetchUserGitIdentities(99)
	if err != nil {
		t.Fatalf("FetchUserGitIdentities: %v", err)
	}
	if len(idents) != 1 || idents[0].UserID != 99 || idents[0].UserName != "Ann" {
		t.Fatalf("idents=%+v", idents)
	}
	none, err := repo.FetchUserGitIdentities(1)
	if err != nil {
		t.Fatalf("other user: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("owner identities must not be returned for another user, got %+v", none)
	}
}

func TestSQLiteBusinessRepository_FetchRepoIdentitiesPrefersCommentJSON(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE task_comments (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL DEFAULT '',
			created_by_id VARCHAR(64) NOT NULL DEFAULT '',
			repo_identities_json TEXT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE task_git_identities (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL DEFAULT '',
			git_user_name VARCHAR(255) NOT NULL DEFAULT '',
			git_user_email VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE task_repo_identities (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL DEFAULT '',
			repo_url VARCHAR(512) NOT NULL DEFAULT '',
			git_identity_id VARCHAR(64) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO task_git_identities(id, user_id, git_user_name, git_user_email) VALUES
			('gid-task', '1', 'Owner', 'owner@example.com'),
			('gid-comment', '99', 'Ann', 'ann@example.com');
		INSERT INTO task_repo_identities(id, task_id, repo_url, git_identity_id)
			VALUES ('ri-1', 'task_1', 'https://git.example/a.git', 'gid-task');
		INSERT INTO task_comments(id, task_id, created_by_id, repo_identities_json)
			VALUES ('cmt_ann', 'task_1', '99',
			'[{"repo_url":"https://git.example/a.git","git_identity_id":"gid-comment"}]');
		INSERT INTO task_comments(id, task_id, created_by_id, repo_identities_json)
			VALUES ('cmt_legacy', 'task_1', '99', '[]');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewSQLiteBusinessRepository(db, nil)
	got, err := repo.FetchRepoIdentities("task_1", "cmt_ann")
	if err != nil {
		t.Fatalf("comment identities: %v", err)
	}
	if len(got) != 1 || got[0].GitIdentityID != "gid-comment" || got[0].UserName != "Ann" || !got[0].FromComment {
		t.Fatalf("want comment identity, got %+v", got)
	}

	fallback, err := repo.FetchRepoIdentities("task_1", "cmt_legacy")
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if len(fallback) != 1 || fallback[0].GitIdentityID != "gid-task" || fallback[0].FromComment {
		t.Fatalf("empty JSON must fall back to task table, got %+v", fallback)
	}

	taskOnly, err := repo.FetchRepoIdentities("task_1", "")
	if err != nil {
		t.Fatalf("task only: %v", err)
	}
	if len(taskOnly) != 1 || taskOnly[0].GitIdentityID != "gid-task" {
		t.Fatalf("no comment_id must use task table, got %+v", taskOnly)
	}
}

func TestSQLiteBusinessRepository_FetchRepoIdentitiesKeepsSiteLevelOAuthGrant(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE task_comments (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL DEFAULT '',
			created_by_id VARCHAR(64) NOT NULL DEFAULT '',
			repo_identities_json TEXT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE task_git_identities (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL DEFAULT '',
			git_user_name VARCHAR(255) NOT NULL DEFAULT '',
			git_user_email VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE task_repo_identities (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL DEFAULT '',
			repo_url VARCHAR(512) NOT NULL DEFAULT '',
			git_identity_id VARCHAR(64) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO task_repo_identities(id, task_id, repo_url, git_identity_id)
			VALUES ('ri-1', 'task_1', 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work.git', 'gid-task');
		INSERT INTO task_comments(id, task_id, created_by_id, repo_identities_json)
			VALUES ('cmt_grant', 'task_1', '99',
			'[{"repo_url":"","oauth_gitsite":"gitlab-tencent-sh-1.daydaymoney.com","oauth_remote_user_id":"gl-user","oauth_granted_at":"2026-09-02T00:00:00Z"}]');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewSQLiteBusinessRepository(db, nil)
	got, err := repo.FetchRepoIdentities("task_1", "cmt_grant")
	if err != nil {
		t.Fatalf("comment identities: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("site-level L2 must not be dropped, got %+v", got)
	}
	if got[0].OauthGitsite != "gitlab-tencent-sh-1.daydaymoney.com" || got[0].RepoURL != "" || !got[0].FromComment {
		t.Fatalf("want site-level comment grant, got %+v", got[0])
	}
}
