-- OPT-20260820-026 已安装镜像快照 SaaS inbound 契约版本。
--
-- 背景：ADR-0024 一期只在厂商镜像目录（ai_provider_vendorcontainerimage.
--       saas_inbound_skill_version）登记契约版本，未写入已安装镜像记录，容器进程
--       启动时无法自证实现的 inbound 版本。安装路径已能从公开目录/厂商镜像字段
--       取到 saas_inbound_skill_version，本迁移在已安装镜像表补列，供 start-vm
--       UserData 注入 SAAS_INBOUND_SKILL_VERSION。
--
-- 幂等：information_schema 列存在性守卫，重复执行无害。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'cloud_tenant_installed_images'
      AND COLUMN_NAME = 'saas_inbound_skill_version'
  ),
  'ALTER TABLE `cloud_tenant_installed_images`
     ADD COLUMN `saas_inbound_skill_version` VARCHAR(64) NULL DEFAULT NULL
     COMMENT ''SaaS inbound skill contract version (ADR-0024); populated from catalog at install time'''
   ,
  'SELECT 1'
);
PREPARE stmt FROM @stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
