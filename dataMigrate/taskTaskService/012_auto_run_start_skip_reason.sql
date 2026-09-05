-- taskTaskService: 持久化 auto_run 软跳过启服原因，供任务详情冷打开展示
-- Applied once via data_migrate_log tracking.

-- L2 稳定存在性检查: auto_run_start_skip_reason 列
SET @has_col = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'auto_run_start_skip_reason');

SET @stmt = IF(@has_col = 0, 'ALTER TABLE task_tasks ADD COLUMN auto_run_start_skip_reason VARCHAR(512) NOT NULL DEFAULT '''' COMMENT ''auto_run soft-skip start-vm reason''', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
