-- Migration 003: Rename tables to conform to service prefix convention.
-- Rule: taskAIComment → ai_comment_ prefix.
-- Guarded: skips if old table gone or new table already exists (idempotent across restarts).

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'container_agent_comments')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_comment_container_agent_comments'),
  'ALTER TABLE `container_agent_comments` RENAME TO `ai_comment_container_agent_comments`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_task_comments')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ai_comment_task_comments'),
  'ALTER TABLE `ai_task_comments` RENAME TO `ai_comment_task_comments`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
