-- OPT-20260806-065 审核流：厂商申请单复用 ai_provider_vendor 表，
-- 新增审核字段（is_active=0 且 review_note 空 = 待审核；review_note 非空 = 已驳回）。
SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='review_note'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `review_note` TEXT NULL AFTER `contact_name`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='reviewed_at'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `reviewed_at` DATETIME NULL AFTER `review_note`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='reviewed_by'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `reviewed_by` BIGINT NULL AFTER `reviewed_at`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
