-- taskGitOauth: Core schema
-- Tables: git_oauth_tenant_gitlab_oauth_connections, git_oauth_appusercredential,
--          git_oauth_appaccesstokenuseaudit, git_oauth_taskcredentialaudit
-- Tracking: data_migrate_log

CREATE TABLE IF NOT EXISTS data_migrate_log (
    step_key VARCHAR(255) PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS git_oauth_tenant_gitlab_oauth_connections (
    id VARCHAR(64) PRIMARY KEY NOT NULL,
    company_id VARCHAR(255) NOT NULL UNIQUE,
    base_url TEXT NOT NULL,
    client_id TEXT NOT NULL,
    client_secret_enc TEXT NOT NULL,
    remark TEXT NOT NULL DEFAULT (''),
    redirect_uri TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT (''),
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS git_oauth_appusercredential (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    provider varchar(64) NOT NULL,
    task2app_user_id bigint NOT NULL,
    refresh_token_cipher text NOT NULL,
    remote_user_id varchar(32) NOT NULL,
    remote_login varchar(255) NOT NULL DEFAULT '',
    scope varchar(512) NOT NULL DEFAULT '',
    bind_status varchar(32) NOT NULL DEFAULT 'active',
    bind_error varchar(512) NOT NULL DEFAULT '',
    created_at datetime NOT NULL,
    updated_at datetime NOT NULL,
    UNIQUE(provider, task2app_user_id, remote_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS git_oauth_appaccesstokenuseaudit (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    provider varchar(64) NOT NULL,
    task2app_user_id bigint NOT NULL,
    company_id bigint NULL,
    workspace_id bigint NULL,
    action varchar(64) NOT NULL,
    access_token_fingerprint varchar(64) NOT NULL DEFAULT '',
    detail text NOT NULL,
    created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS git_oauth_taskcredentialaudit (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    provider varchar(64) NOT NULL,
    task2app_task_id varchar(64) NOT NULL,
    task2app_workspace_id varchar(64) NOT NULL DEFAULT '',
    task2app_company_id bigint NULL,
    task2app_user_id bigint NULL,
    action varchar(64) NOT NULL,
    detail text NOT NULL,
    created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
