-- taskGitOauth: one-shot L2 grant tickets (comment/auto-run before comment_id exists)
CREATE TABLE IF NOT EXISTS git_oauth_grant_ticket (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    task2app_user_id VARCHAR(36) NOT NULL,
    gitsite VARCHAR(255) NOT NULL,
    remote_user_id VARCHAR(64) NOT NULL DEFAULT '',
    expires_at DATETIME NOT NULL,
    consumed_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_git_oauth_grant_ticket_user (task2app_user_id, gitsite),
    INDEX idx_git_oauth_grant_ticket_exp (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
