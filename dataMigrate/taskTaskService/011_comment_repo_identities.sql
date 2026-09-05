-- taskTaskService: 评论级仓库身份快照 — task_comments.repo_identities_json
-- 运行评论（@镜像）写入本次选用的 git_identity_id / github_user_id，避免并行评论覆盖任务级身份表。
-- Applied once via data_migrate_log tracking.

-- L2 稳定存在性检查: repo_identities_json 列
SET @has_col = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_comments' AND COLUMN_NAME = 'repo_identities_json');

SET @stmt = IF(@has_col = 0, 'ALTER TABLE task_comments ADD COLUMN repo_identities_json TEXT NULL', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
