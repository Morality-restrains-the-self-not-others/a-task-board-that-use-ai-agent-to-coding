-- 厂商申请证照 + 联系方式（vendor-application-kyc-docs）
-- id_card / business_license 存本地 file_key；contact_phone 为 E.164
SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='id_card_file_key'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `id_card_file_key` VARCHAR(512) NULL AFTER `contact_name`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='business_license_file_key'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `business_license_file_key` VARCHAR(512) NULL AFTER `id_card_file_key`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_vendor' AND COLUMN_NAME='contact_phone'),
  'ALTER TABLE `ai_provider_vendor` ADD COLUMN `contact_phone` VARCHAR(32) NULL AFTER `business_license_file_key`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
