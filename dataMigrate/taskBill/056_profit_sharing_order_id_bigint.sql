-- 056: billing_profit_sharing.order_id / tenant_id INT→BIGINT
-- markOrderForProfitSharing 写入 snowflake 订单/租户 ID（> INT_MAX）。
-- 021 建表时为 INTEGER，STRICT 下插入失败，管理员无法按真实订单记录分账。
-- 幂等：已是 bigint 则跳过。

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_profit_sharing' AND COLUMN_NAME = 'order_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_profit_sharing` MODIFY COLUMN `order_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_ps_order_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_profit_sharing' AND COLUMN_NAME = 'tenant_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_profit_sharing` MODIFY COLUMN `tenant_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_ps_tenant_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
