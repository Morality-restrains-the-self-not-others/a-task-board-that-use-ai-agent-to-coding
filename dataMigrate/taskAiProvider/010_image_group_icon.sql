-- Image group marketplace icon (required on create/update; existing rows stay empty until next edit).
SET @stmt = IF(NOT EXISTS(SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_provider_containerimagegroup' AND COLUMN_NAME='icon_file_key'),
  'ALTER TABLE `ai_provider_containerimagegroup` ADD COLUMN `icon_file_key` VARCHAR(512) NOT NULL DEFAULT '''' AFTER `description`',
  'SELECT 1');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
