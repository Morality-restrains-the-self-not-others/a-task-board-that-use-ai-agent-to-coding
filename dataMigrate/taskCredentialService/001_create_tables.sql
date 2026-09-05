-- Migration 001: Create container tokens table.
-- Owned by task-credential-service (Go). Django CloudServerConfig equivalent.
-- MySQL 8 compatible (no TEXT DEFAULT / no functional prefix indexes on TEXT).

-- Prefix-compliance rename: container_tokens → credential_container_tokens.
-- Guarded: skips if already done (fresh DB or post-rename restart).
SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'container_tokens')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'credential_container_tokens'),
  'ALTER TABLE `container_tokens` RENAME TO `credential_container_tokens`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

-- Prefix-compliance rename: token_audit_events → credential_token_audit_events.
SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'token_audit_events')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'credential_token_audit_events'),
  'ALTER TABLE `token_audit_events` RENAME TO `credential_token_audit_events`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS credential_container_tokens (
    id                              VARCHAR(64) PRIMARY KEY,
    task_id                         VARCHAR(64) NOT NULL,
    company_id                      VARCHAR(64) NOT NULL DEFAULT '',
    workspace_id                    VARCHAR(64) NOT NULL DEFAULT '',
    container_access_token          VARCHAR(512) NOT NULL DEFAULT '',
    container_access_token_expires_at VARCHAR(64) NULL,
    container_refresh_token         VARCHAR(512) NOT NULL DEFAULT '',
    server_url                      VARCHAR(1024) NOT NULL DEFAULT '',
    business_api_endpoint           VARCHAR(1024) NOT NULL DEFAULT '',
    container_vscode_url            VARCHAR(1024) NOT NULL DEFAULT '',
    authorization_id                VARCHAR(64) NOT NULL DEFAULT '',
    image_id                        VARCHAR(64) NOT NULL DEFAULT '',
    instance_type                   VARCHAR(128) NOT NULL DEFAULT '',
    region_id                       VARCHAR(64) NOT NULL DEFAULT '',
    zone_id                         VARCHAR(64) NOT NULL DEFAULT '',
    created_at                      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Idempotent CREATE INDEX (MySQL 8 does not support IF NOT EXISTS for indexes).
SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.STATISTICS
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'credential_container_tokens' AND INDEX_NAME = 'idx_container_tokens_access'),
  'CREATE INDEX idx_container_tokens_access ON credential_container_tokens(container_access_token)',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.STATISTICS
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'credential_container_tokens' AND INDEX_NAME = 'idx_container_tokens_task'),
  'CREATE INDEX idx_container_tokens_task ON credential_container_tokens(company_id, task_id)',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

-- Migration 002: Create token audit events table.
CREATE TABLE IF NOT EXISTS credential_token_audit_events (
    id                      VARCHAR(64) PRIMARY KEY,
    task_id                 VARCHAR(64) NOT NULL DEFAULT '',
    event_type              VARCHAR(64) NOT NULL DEFAULT '',
    access_token_sha256     VARCHAR(64) NOT NULL DEFAULT '',
    prev_access_token_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    new_access_token_sha256  VARCHAR(64) NOT NULL DEFAULT '',
    refresh_token_sha256    VARCHAR(64) NOT NULL DEFAULT '',
    source_component        VARCHAR(128) NOT NULL DEFAULT '',
    trace_id                VARCHAR(64) NOT NULL DEFAULT '',
    error_code              VARCHAR(64) NOT NULL DEFAULT '',
    error_detail            TEXT,
    seq                     INTEGER NOT NULL DEFAULT 0,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Idempotent CREATE INDEX (MySQL 8 does not support IF NOT EXISTS for indexes).
SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.STATISTICS
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'credential_token_audit_events' AND INDEX_NAME = 'idx_token_audit_task'),
  'CREATE INDEX idx_token_audit_task ON credential_token_audit_events(task_id, created_at)',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
