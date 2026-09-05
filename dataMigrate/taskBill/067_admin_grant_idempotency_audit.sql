-- 067: admin_grant 幂等键审计——记录操作者与目标租户
-- 管理端可追溯「哪次操作产生了哪张赠送订单」；老行/用户侧操作这些列保持 NULL。
-- 低频审计列，直接 ALTER。

ALTER TABLE billing_idempotency_key
  ADD COLUMN operator_user_id VARCHAR(64) NULL DEFAULT NULL
    COMMENT '管理员操作者 user_id（admin_grant 等管理路径）' AFTER transaction_id,
  ADD COLUMN tenant_id BIGINT NULL DEFAULT NULL
    COMMENT '目标租户（管理路径）' AFTER operator_user_id,
  ADD COLUMN op_type VARCHAR(32) NULL DEFAULT NULL
    COMMENT '操作类型：admin_grant / user 等' AFTER tenant_id;
