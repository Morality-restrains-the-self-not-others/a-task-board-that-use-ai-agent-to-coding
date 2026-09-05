-- taskTaskService: record who enabled queued auto-run (OAuth actor; not task owner)
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_queued_auto_run_memberships' AND COLUMN_NAME = 'enabler_user_id') = 0,
  'ALTER TABLE task_queued_auto_run_memberships ADD COLUMN enabler_user_id VARCHAR(36) NOT NULL DEFAULT ''''',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
