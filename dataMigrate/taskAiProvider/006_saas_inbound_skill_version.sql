-- 容器镜像声明所实现的「容器→SaaS 接口」契约版本（ADR-0024）。
-- 存量行回填 v1（当前 skill SSOT）。非时间累积型目录表，无需分区。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'ai_provider_vendorcontainerimage'
      AND COLUMN_NAME = 'saas_inbound_skill_version'
  ),
  'ALTER TABLE `ai_provider_vendorcontainerimage` ADD COLUMN `saas_inbound_skill_version` VARCHAR(32) NOT NULL DEFAULT ''1'' COMMENT ''published inbound skill contract version'' AFTER `version`',
  'SELECT 1'
);
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

UPDATE ai_provider_vendorcontainerimage
SET saas_inbound_skill_version = '1'
WHERE saas_inbound_skill_version IS NULL OR saas_inbound_skill_version = '';
