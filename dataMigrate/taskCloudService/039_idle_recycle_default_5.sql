-- 039: 工作空间闲置自动回收默认由 30 分钟改为 5 分钟
-- 仅改列 DEFAULT；已保存的策略行保持原值。无策略行时应用层 defaultIdleRecycleMinutes=5。
-- 幂等：information_schema 检查 COLUMN_DEFAULT 仍为 30 才 ALTER。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_workspace_machine_policies' AND COLUMN_NAME = 'idle_recycle_minutes' AND COLUMN_DEFAULT = '30') > 0, 'ALTER TABLE cloud_workspace_machine_policies ALTER idle_recycle_minutes SET DEFAULT 5', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
