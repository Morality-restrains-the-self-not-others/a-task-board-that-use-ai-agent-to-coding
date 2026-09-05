-- taskTaskService: 任务帖存续期 — task_tasks.post_expires_at
-- 定价模型重构: 创建帖消耗 1 创建帖次数 → 帖子获得 12 个月存续期
-- 单帖独立到期；续存 = 再消耗 1 次数 → max(now, 当前到期日) + 12M
-- Applied once via data_migrate_log tracking.

-- L2 稳定存在性检查: post_expires_at 列
SET @has_col = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND COLUMN_NAME = 'post_expires_at');

-- 新增列 (DATETIME NULL; NULL = 未启用存续期, 到期判定: post_expires_at IS NOT NULL AND post_expires_at < now)
SET @stmt = IF(@has_col = 0, 'ALTER TABLE task_tasks ADD COLUMN post_expires_at DATETIME NULL', 'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- 存量回填: 上线日起统一 +12 个月 (用户已确认「存量统一新规则」)
-- 存储约定: 与 Go driver 的 DSN loc=Local 一致, 存北京墙钟时间 (UTC+8)。
-- NOW() 在 UTC 时区容器中返回 UTC 墙钟, 故用 CONVERT_TZ 转北京墙钟。
UPDATE task_tasks SET post_expires_at = DATE_ADD(CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+08:00'), INTERVAL 12 MONTH) WHERE post_expires_at IS NULL;

-- 到期扫描索引 (taskEvents 每日到期扫描 + 惰性校验)
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'task_tasks' AND INDEX_NAME = 'idx_tasks_post_expires') = 0, 'CREATE INDEX idx_tasks_post_expires ON task_tasks(post_expires_at)', 'SELECT 1 ''skip: idx_tasks_post_expires exists''');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
