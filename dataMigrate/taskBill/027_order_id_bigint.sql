-- 027: billing_resource_order 相关列 INT→BIGINT（snowflake ID 超出 INT 范围）
-- 背景：createOrder / adminGrantResources 使用 generateSnowflakeID() 生成 ID，
-- tenant_id 也是 snowflake 值（~8.7e17 > INT_MAX 2.1e9）。
-- 原 INT 列无法容纳 snowflake 值。
-- 2026-07-31

ALTER TABLE billing_resource_order MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE billing_resource_order MODIFY COLUMN tenant_id BIGINT NOT NULL;
ALTER TABLE billing_resource_order_item MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;
ALTER TABLE billing_resource_order_item MODIFY COLUMN order_id BIGINT NOT NULL;
