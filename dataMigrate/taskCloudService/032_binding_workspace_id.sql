-- OPT-20260821-003：binding 行持久化 workspace_id，供日志分片在无任务级 CSC 时也能落库。
-- 应用前需确认日志分片（030）已回填；本列允许空串（存量行保留，读路径回退 lookupWorkspaceIDByTask）。
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_comment_container_bindings'
      AND COLUMN_NAME = 'workspace_id') = 0,
    'ALTER TABLE cloud_comment_container_bindings ADD COLUMN workspace_id VARCHAR(64) NOT NULL DEFAULT \'\' AFTER company_id',
    'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
