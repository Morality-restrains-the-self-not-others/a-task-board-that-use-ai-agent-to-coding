-- taskTaskService: Backfill created_by_id from legacy user_id column
-- Extracted from Go migrateCommentsLegacyUserID() pre-update step
-- Only runs when user_id column still exists in comments table.
-- Idempotent: UPDATE is a no-op when created_by_id is already populated.
-- Applied once via data_migrate_log tracking.

SET @has_user_id = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'user_id');

SET @stmt = IF(@has_user_id > 0, 'UPDATE comments SET created_by_id = CASE WHEN created_by_id IS NULL OR created_by_id = \'\' THEN COALESCE(user_id, \'\') ELSE created_by_id END', 'SELECT 1 \'skip: comments.user_id column not present\'');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
