-- taskCloudService: cloud_server_configs multi-comment migration
-- Extracted from Go migrateCloudServerConfigMultiComment() + dedupeCloudServerConfigsForWorkspaceTaskComment()
-- Each statement is idempotent. The dedup DELETE is inherently idempotent (no-op when no duplicates exist).
-- Applied once via data_migrate_log tracking.

-- Add comment_id column if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'comment_id') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN comment_id VARCHAR(64) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Dedup: delete rows that have a newer duplicate (same workspace_id / task_id / comment_id).
-- Keeps the row with the latest updated_at (tiebreak on id DESC).
DELETE t1 FROM cloud_server_configs t1
INNER JOIN cloud_server_configs t2 ON
  t1.workspace_id = t2.workspace_id
  AND t1.task_id = t2.task_id
  AND COALESCE(t1.comment_id,'') = COALESCE(t2.comment_id,'')
  AND (t1.updated_at < t2.updated_at OR (t1.updated_at = t2.updated_at AND t1.id < t2.id));

-- Drop legacy index idx_csc_company_task (from pre-multi-comment schema)
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND INDEX_NAME = 'idx_csc_company_task') > 0, 'DROP INDEX idx_csc_company_task ON cloud_server_configs', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Create idx_csc_workspace_task if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND INDEX_NAME = 'idx_csc_workspace_task') = 0, 'CREATE INDEX idx_csc_workspace_task ON cloud_server_configs(workspace_id, task_id)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Create unique idx_csc_workspace_task_comment if missing
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND INDEX_NAME = 'idx_csc_workspace_task_comment') = 0, 'CREATE UNIQUE INDEX idx_csc_workspace_task_comment ON cloud_server_configs(workspace_id, task_id, comment_id)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
