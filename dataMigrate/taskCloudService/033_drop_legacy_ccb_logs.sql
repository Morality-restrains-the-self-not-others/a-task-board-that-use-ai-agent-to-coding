-- OPT-20260820-023: 遗留单表 cloud_comment_container_binding_logs 回填已完成
-- （030 已把存量迁到 16 张分片，生产核对分片行数 ≥ 遗留表行数）。
-- 应用已停止写入遗留表（隔离回归测保障）。RENAME 过渡保留数据可回滚，
-- 且让「误查旧表名」直接失败，防止排障走错表。
-- 幂等：information_schema 检查遗留表存在才 RENAME，已迁移时跳过。
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.tables
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_comment_container_binding_logs') = 1,
    'RENAME TABLE cloud_comment_container_binding_logs TO cloud_comment_container_binding_logs_deprecated_20260821',
    'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
