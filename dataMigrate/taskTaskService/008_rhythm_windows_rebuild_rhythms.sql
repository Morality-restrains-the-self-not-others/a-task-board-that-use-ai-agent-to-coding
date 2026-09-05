-- taskTaskService: Rebuild top_deliverable_schedule_rhythms without legacy window columns
-- Extracted from Go migrateLegacyRhythmWindows() table rebuild step
-- Guard: daily_start column still exists in top_deliverable_schedule_rhythms
-- DDL is atomic under MySQL 8.0 InnoDB transactional DDL.
-- Applied once via data_migrate_log tracking.

SET @has_legacy = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'top_deliverable_schedule_rhythms' AND COLUMN_NAME = 'daily_start');

-- Create new table without legacy columns (matching 001_schema.sql)
SET @stmt = IF(@has_legacy > 0, 'CREATE TABLE top_deliverable_schedule_rhythms__new (
    task_id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    enabled TINYINT NOT NULL DEFAULT 0,
    timezone VARCHAR(64) NOT NULL DEFAULT \'Asia/Shanghai\',
    updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Copy data (only modern columns)
SET @stmt = IF(@has_legacy > 0, 'INSERT INTO top_deliverable_schedule_rhythms__new(task_id, tenant_id, workspace_id, enabled, timezone, updated_at) SELECT task_id, tenant_id, workspace_id, enabled, timezone, updated_at FROM top_deliverable_schedule_rhythms', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Drop old table with legacy columns
SET @stmt = IF(@has_legacy > 0, 'DROP TABLE top_deliverable_schedule_rhythms', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Rename new table to original name
SET @stmt = IF(@has_legacy > 0, 'RENAME TABLE top_deliverable_schedule_rhythms__new TO top_deliverable_schedule_rhythms', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
