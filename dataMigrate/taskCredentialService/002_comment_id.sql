-- Migration 002: comment-scoped container tokens (ADR-0005).
-- comment_id 与 tenant/workspace/task 一起唯一标识一行令牌。

SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS
              WHERE TABLE_SCHEMA = DATABASE()
                AND TABLE_NAME = 'credential_container_tokens'
                AND COLUMN_NAME = 'comment_id'),
  'ALTER TABLE `credential_container_tokens` ADD COLUMN `comment_id` VARCHAR(64) NOT NULL DEFAULT '''' AFTER `workspace_id`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

-- Keep the latest row per (company, workspace, task, comment); drop older duplicates
-- so UNIQUE can be added. IssueToken historically inserted extra empty-refresh rows.
DELETE t1 FROM credential_container_tokens t1
INNER JOIN credential_container_tokens t2
  ON t1.company_id = t2.company_id
 AND t1.workspace_id = t2.workspace_id
 AND t1.task_id = t2.task_id
 AND COALESCE(t1.comment_id, '') = COALESCE(t2.comment_id, '')
 AND (t1.updated_at < t2.updated_at OR (t1.updated_at = t2.updated_at AND t1.id > t2.id));

SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.STATISTICS
              WHERE TABLE_SCHEMA = DATABASE()
                AND TABLE_NAME = 'credential_container_tokens'
                AND INDEX_NAME = 'uk_credential_container_tokens_scope'),
  'ALTER TABLE `credential_container_tokens` ADD UNIQUE KEY `uk_credential_container_tokens_scope` (`company_id`, `workspace_id`, `task_id`, `comment_id`)',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS
              WHERE TABLE_SCHEMA = DATABASE()
                AND TABLE_NAME = 'credential_token_audit_events'
                AND COLUMN_NAME = 'comment_id'),
  'ALTER TABLE `credential_token_audit_events` ADD COLUMN `comment_id` VARCHAR(64) NOT NULL DEFAULT '''' AFTER `task_id`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
