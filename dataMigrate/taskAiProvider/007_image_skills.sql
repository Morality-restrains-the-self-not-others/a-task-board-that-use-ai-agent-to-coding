-- 镜像内 /app/imageSkills.yaml 抽取结果（ADR-0026）。
-- 非时间累积型目录表，无需分区。空字符串表示尚未抽取。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'ai_provider_vendorcontainerimage'
      AND COLUMN_NAME = 'image_skills_json'
  ),
  'ALTER TABLE `ai_provider_vendorcontainerimage`
     ADD COLUMN `image_skills_json` TEXT NULL COMMENT ''parsed /app/imageSkills.yaml JSON'' AFTER `auto_run_steps_digest`,
     ADD COLUMN `image_skills_extract_status` VARCHAR(64) NOT NULL DEFAULT '''' AFTER `image_skills_json`,
     ADD COLUMN `image_skills_digest` VARCHAR(255) NOT NULL DEFAULT '''' AFTER `image_skills_extract_status`',
  'SELECT 1'
);
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
