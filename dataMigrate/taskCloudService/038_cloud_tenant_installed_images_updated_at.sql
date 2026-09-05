-- OPT-20260824-XXX 已安装镜像快照目录 updated_at（镜像更新时间）。
--
-- 背景：创建任务弹层需要同时展示镜像的版本、更新时间与说明。版本/说明安装时
--       已快照，但目录镜像的 updated_at 未入库——列表接口为返回该字段每次都要
--       额外请求镜像服务目录，且开发模式镜像不在公开目录中。本迁移补列，由
--       安装路径在安装时快照；存量行以 installed_at 回填兜底。
--
-- 幂等：information_schema 列存在性守卫 + WHERE updated_at IS NULL，重复执行无害。
SET @stmt = IF(
  NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'cloud_tenant_installed_images'
      AND COLUMN_NAME = 'updated_at'
  ),
  'ALTER TABLE `cloud_tenant_installed_images`
     ADD COLUMN `updated_at` DATETIME NULL DEFAULT NULL
     COMMENT ''镜像更新时间（目录 updated_at 安装时快照；无则回填安装时间）'' AFTER `installed_at`',
  'SELECT 1'
);
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- 存量行回填：旧记录无目录时间时以安装时间兜底，保证前端始终有值可展示
UPDATE cloud_tenant_installed_images SET updated_at = installed_at WHERE updated_at IS NULL;
