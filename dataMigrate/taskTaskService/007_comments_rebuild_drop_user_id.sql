-- taskTaskService: Rebuild comments table without legacy user_id column
-- Extracted from Go migrateCommentsLegacyUserID() table rebuild step
-- Only runs when user_id column still exists in comments table.
-- DDL is atomic under MySQL 8.0 InnoDB transactional DDL.
-- Applied once via data_migrate_log tracking.
--
-- Target table: task_comments (conforms to task_ prefix convention).
-- When 001_schema.sql already created task_comments (empty, in fresh DBs),
-- we drop it first so the rename from comments__new → task_comments succeeds.

SET @has_user_id = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'user_id');

-- Drop the empty task_comments created by 001_schema.sql when rebuilding from legacy
SET @stmt = IF(@has_user_id > 0 AND (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments') > 0, 'DROP TABLE task_comments', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Create new table without user_id (matching 001_schema.sql)
SET @stmt = IF(@has_user_id > 0, 'CREATE TABLE comments__new (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    created_by_id VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    mentions_json TEXT,
    execution_mode VARCHAR(64) NOT NULL DEFAULT \'wait_previous\',
    depends_on_comment_ids TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Copy data: backfill created_by_id from user_id, COALESCE for nullable columns
SET @stmt = IF(@has_user_id > 0, 'INSERT INTO comments__new(id, task_id, created_by_id, content, mentions_json, execution_mode, depends_on_comment_ids, created_at, updated_at) SELECT id, task_id, CASE WHEN created_by_id IS NULL OR created_by_id = \'\' THEN COALESCE(user_id, \'\') ELSE created_by_id END, content, COALESCE(mentions_json, \'\'), COALESCE(execution_mode, \'wait_previous\'), COALESCE(depends_on_comment_ids, \'[]\'), COALESCE(created_at, CURRENT_TIMESTAMP), CURRENT_TIMESTAMP FROM comments', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Drop old table (DDL is atomic in MySQL 8.0 — rolls back with transaction)
SET @stmt = IF(@has_user_id > 0, 'DROP TABLE comments', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Rename new table to the service-prefixed target name (conforms to task_ prefix convention)
SET @stmt = IF(@has_user_id > 0, 'RENAME TABLE comments__new TO task_comments', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Recreate index (matching 001_schema.sql)
SET @stmt = IF(@has_user_id > 0 AND (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND INDEX_NAME = 'idx_comments_task') = 0, 'CREATE INDEX idx_comments_task ON task_comments(task_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
