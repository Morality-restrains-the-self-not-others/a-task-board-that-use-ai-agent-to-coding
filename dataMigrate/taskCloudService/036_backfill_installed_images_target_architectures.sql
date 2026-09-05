-- OPT-20260821-016 回填已安装镜像空的 target_architectures。
--
-- 背景：cloud_tenant_installed_images.target_architectures 部分行仍为 JSON 空数组 `[]`
--       或含 `"unknown"`（如 trae-agent `private_x86_64-latest`）。读路径已从
--       version/name/image_url 推断规范 ISA，但库内脏数据会让只读 DB 的脚本再次
--       当成「无架构」。
--
-- 处理：仅对 target_architectures 为 NULL / `[]` / 含 `unknown` 的行，从
--       version/name/image_url 提取首个 ISA token（x86_64|amd64|aarch64|arm64）
--       并回填为规范 JSON 数组；无法提取的行保持原样（由外部监控/清单兜底）。
-- 幂等：UPDATE 只在仍为脏值的行上生效，重复执行不产生副作用。
UPDATE `cloud_tenant_installed_images`
SET `target_architectures` = JSON_ARRAY(
  CASE REGEXP_SUBSTR(CONCAT_WS(' ', `name`, COALESCE(`version`, ''), `image_url`), '(x86_64|amd64|aarch64|arm64)')
    WHEN 'amd64'   THEN 'x86_64'
    WHEN 'aarch64' THEN 'arm64'
    WHEN 'arm64'   THEN 'arm64'
    ELSE 'x86_64'
  END
)
WHERE (`target_architectures` IS NULL OR `target_architectures` = '[]' OR `target_architectures` LIKE '%unknown%')
  AND REGEXP_SUBSTR(CONCAT_WS(' ', `name`, COALESCE(`version`, ''), `image_url`), '(x86_64|amd64|aarch64|arm64)') IS NOT NULL;
