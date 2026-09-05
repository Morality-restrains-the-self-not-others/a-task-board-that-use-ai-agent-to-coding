-- 038: 指令闲置回收 — CSC 闲置起点 + CPA 可选 STS Role
-- instruction_idle_since：交付成功后心跳 Mark 写入；新指令 Clear；L2 到期扫描即使 server_url 非空也释放。
-- sts_release_role_arn：可选 RAM Role；空则 task-detail 不带 machine_release_sts（ADR-0031）。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'instruction_idle_since') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN instruction_idle_since DATETIME NULL', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_platform_authorizations' AND COLUMN_NAME = 'sts_release_role_arn') = 0, 'ALTER TABLE cloud_platform_authorizations ADD COLUMN sts_release_role_arn VARCHAR(255) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND INDEX_NAME = 'idx_csc_instruction_idle_since') = 0, 'ALTER TABLE cloud_server_configs ADD INDEX idx_csc_instruction_idle_since (instruction_idle_since)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
