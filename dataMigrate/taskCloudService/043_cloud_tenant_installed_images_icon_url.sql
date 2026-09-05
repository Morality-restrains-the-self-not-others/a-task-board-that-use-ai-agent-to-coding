-- 已安装镜像快照目录 icon_url（镜像组图标）。
--
-- 背景：镜像市场「已发布 / 开发中」卡片已展示组图标，但「已安装镜像」列表
--       未带 icon_url——安装路径未从目录快照该字段，存量行也无列可存。
--       前端同页可用目录回退，但其它消费方（以及目录未加载时）仍无图标。
--       本迁移补列，由安装路径快照；存量行保持空串，由前端目录回退补齐。
--
-- 幂等：information_schema 列存在性守卫，重复执行无害。
-- 伸缩：元数据列，非时间累积型宽表膨胀；无需分区。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'cloud_tenant_installed_images'
      AND COLUMN_NAME = 'icon_url'
  ),
  'ALTER TABLE `cloud_tenant_installed_images`
     ADD COLUMN `icon_url` VARCHAR(512) NOT NULL DEFAULT ''''
     COMMENT ''镜像组图标 URL（目录 icon_url 安装时快照）'' AFTER `image_url`',
  'SELECT 1'
);
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
