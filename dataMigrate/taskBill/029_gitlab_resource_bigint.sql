-- 029: billing_tenant_gitlab_resource 列 INT→BIGINT（snowflake ID 超出 INT 范围）
-- 背景：tenant_id 是 snowflake 值（~8.7e17 > INT_MAX 2.1e9），原 INT AUTO_INCREMENT
-- 无法容纳，导致 INSERT 失败 → refreshTenantDiskUsage 静默失败 → 前端显示 0 GB / 0 GB。
-- disk_used_bytes 同样可能超过 INT_MAX（2.1 GB），一并升级为 BIGINT。
-- 参考：027_order_id_bigint.sql 修复了 billing_resource_order 同类问题。
-- 2026-08-01

-- tenant_id: 移除 AUTO_INCREMENT（由应用层分配 snowflake ID），INT→BIGINT
ALTER TABLE billing_tenant_gitlab_resource
  MODIFY COLUMN tenant_id BIGINT NOT NULL;

-- disk_used_bytes: GitLab 仓库磁盘用量可能超过 2.1 GB，INT→BIGINT
ALTER TABLE billing_tenant_gitlab_resource
  MODIFY COLUMN disk_used_bytes BIGINT NOT NULL DEFAULT 0;
