-- Migration 005: Rename tables to conform to service prefix convention.
-- Rule: taskTaskService → task_ prefix.
-- Guarded: skips if old table gone or new table already exists (idempotent across restarts).

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'git_identities')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_git_identities'),
  'ALTER TABLE `git_identities` RENAME TO `task_git_identities`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'top_deliverable_schedule_rhythms')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_top_deliverable_schedule_rhythms'),
  'ALTER TABLE `top_deliverable_schedule_rhythms` RENAME TO `task_top_deliverable_schedule_rhythms`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'top_deliverable_schedule_rhythm_windows')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_top_deliverable_schedule_rhythm_windows'),
  'ALTER TABLE `top_deliverable_schedule_rhythm_windows` RENAME TO `task_top_deliverable_schedule_rhythm_windows`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'queued_auto_run_memberships')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_queued_auto_run_memberships'),
  'ALTER TABLE `queued_auto_run_memberships` RENAME TO `task_queued_auto_run_memberships`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'queued_machine_slots')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_queued_machine_slots'),
  'ALTER TABLE `queued_machine_slots` RENAME TO `task_queued_machine_slots`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tasks')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks'),
  'ALTER TABLE `tasks` RENAME TO `task_tasks`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments'),
  'ALTER TABLE `comments` RENAME TO `task_comments`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
