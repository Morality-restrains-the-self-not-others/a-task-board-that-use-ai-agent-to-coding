-- taskProjectService: per-user per-gitsite usage grant for a project page
CREATE TABLE IF NOT EXISTS project_git_oauth_grant (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    task2app_user_id VARCHAR(36) NOT NULL,
    gitsite VARCHAR(255) NOT NULL,
    remote_user_id VARCHAR(64) NOT NULL DEFAULT '',
    granted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_project_git_oauth_grant_user_site (project_id, task2app_user_id, gitsite),
    INDEX idx_project_git_oauth_grant_company (company_id, project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
