-- taskTaskService: 工作空间人读序号 workspace_seq
-- 仅当列尚不存在且 task_tasks 已有存量行时清空任务帖域（不清 task_git_identities）。
-- Applied once via data_migrate_log.

SET @has_col = (SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'workspace_seq');
SET @task_n = (SELECT COUNT(*) FROM information_schema.TABLES t
  WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_NAME = 'task_tasks');
SET @task_n = IF(@task_n = 0, 0, (SELECT COUNT(*) FROM task_tasks));
-- 仅存量任务帖库首次加列时清空；全新 001→010 或仅有 007 回填评论时不删行。
SET @do_wipe = IF(@has_col = 0 AND @task_n > 0, 1, 0);

SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_queued_machine_slots', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_queued_auto_run_memberships', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_top_deliverable_schedule_rhythm_windows', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_top_deliverable_schedule_rhythms', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_feature_params_snapshots', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_feature_params', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_repo_identities', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_branch_strategies', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_projects', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_assignees', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_comments', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF(@do_wipe = 1, 'DELETE FROM task_tasks', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF(@has_col = 0,
  'ALTER TABLE task_tasks ADD COLUMN workspace_seq INT UNSIGNED NULL',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS task_workspace_seq (
  tenant_id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  next_val INT UNSIGNED NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (tenant_id, workspace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND INDEX_NAME = 'uk_task_workspace_seq') = 0,
  'CREATE UNIQUE INDEX uk_task_workspace_seq ON task_tasks(tenant_id, workspace_id, workspace_seq)',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
