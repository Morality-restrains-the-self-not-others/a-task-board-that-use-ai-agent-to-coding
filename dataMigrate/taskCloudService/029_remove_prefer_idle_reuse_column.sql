-- 029: 删除 prefer_idle_reuse 列（ADR-0013 / OPT-20260818-064）
-- 跨任务闲置复用已下线（ADR-0013），客户端与 API 不再读/写该列；列保留会造成
-- 「开关还在」的误解。024 已把存量值清零并默认 0，本迁移删除列。
-- 幂等：information_schema 检查列存在才 DROP，已删除时跳过，防止迁移失败。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_workspace_machine_policies' AND COLUMN_NAME = 'prefer_idle_reuse') > 0, 'ALTER TABLE cloud_workspace_machine_policies DROP COLUMN prefer_idle_reuse', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
