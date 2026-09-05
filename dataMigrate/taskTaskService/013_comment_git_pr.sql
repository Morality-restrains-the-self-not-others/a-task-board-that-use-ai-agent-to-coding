-- taskTaskService: 评论可嵌套回复并记录 git PR/MR 链接（幂等键 git_pr_html_url）
-- Applied once via data_migrate_log tracking.

SET @has_parent = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND COLUMN_NAME = 'parent_comment_id');
SET @stmt = IF(@has_parent = 0, 'ALTER TABLE task_comments ADD COLUMN parent_comment_id VARCHAR(64) NOT NULL DEFAULT ''''', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @has_pr_url = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND COLUMN_NAME = 'git_pr_html_url');
SET @stmt = IF(@has_pr_url = 0, 'ALTER TABLE task_comments ADD COLUMN git_pr_html_url VARCHAR(512) NOT NULL DEFAULT ''''', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @has_pr_json = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND COLUMN_NAME = 'git_pr_json');
SET @stmt = IF(@has_pr_json = 0, 'ALTER TABLE task_comments ADD COLUMN git_pr_json TEXT NULL', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @has_idx = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND INDEX_NAME = 'idx_task_comments_parent');
SET @stmt = IF(@has_idx = 0, 'CREATE INDEX idx_task_comments_parent ON task_comments(task_id, parent_comment_id)', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
