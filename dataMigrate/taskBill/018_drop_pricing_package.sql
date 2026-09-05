-- 018: 删除价格套餐表及相关废弃列
-- billing_unit 已成为资源定价的唯一来源
-- 注意：依赖 SQLite >= 3.35.0（DROP COLUMN 支持）
-- 幂等设计：每个 DROP 均通过 information_schema 检查，列/索引不存在时跳过，防止迁移失败。

-- 先删除依赖索引（否则 DROP COLUMN 会因外键/索引约束失败）
-- MySQL 不支持 DROP INDEX IF EXISTS; 改用 ALTER TABLE DROP INDEX
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND INDEX_NAME = 'billing_account_pricing_package_id') > 0, 'ALTER TABLE billing_account DROP INDEX billing_account_pricing_package_id', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- 1. 删除 billing_account 中不再需要的定价锁列
--    这些列在旧架构中用于"开户时锁价"；新架构中订单直接从 billing_unit 取当前价格
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'pricing_package_id') > 0, 'ALTER TABLE billing_account DROP COLUMN pricing_package_id', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_post_creation_points') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_post_creation_points', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_server_start_points') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_server_start_points', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_normal_renewal_points_per_month') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_normal_renewal_points_per_month', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_programming_renewal_points_per_month') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_programming_renewal_points_per_month', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_gitlab_disk_points_per_gb_per_month') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_gitlab_disk_points_per_gb_per_month', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'locked_gitlab_traffic_points_per_gb') > 0, 'ALTER TABLE billing_account DROP COLUMN locked_gitlab_traffic_points_per_gb', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_account' AND COLUMN_NAME = 'excluded_pricing_package_ids') > 0, 'ALTER TABLE billing_account DROP COLUMN excluded_pricing_package_ids', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- 2. 删除整张价格套餐表
DROP TABLE IF EXISTS billing_pricing_package;
