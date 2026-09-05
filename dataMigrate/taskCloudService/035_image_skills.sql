-- 已安装镜像快照：/app/imageSkills.yaml 抽取结果（ADR-0026）。
-- 非时间累积型目录表，无需分区。空字符串表示尚未抽取或列表为空。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'cloud_tenant_installed_images'
      AND COLUMN_NAME = 'image_skills_json'
  ),
  'ALTER TABLE `cloud_tenant_installed_images`
     ADD COLUMN `image_skills_json` TEXT NULL COMMENT ''parsed /app/imageSkills.yaml JSON'' AFTER `auto_run_steps_digest`,
     ADD COLUMN `image_skills_extract_status` VARCHAR(64) NOT NULL DEFAULT '''' AFTER `image_skills_json`,
     ADD COLUMN `image_skills_digest` VARCHAR(255) NOT NULL DEFAULT '''' AFTER `image_skills_extract_status`',
  'SELECT 1'
);
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
