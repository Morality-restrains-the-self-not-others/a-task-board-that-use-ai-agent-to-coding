-- 信用批次有效期：billing_payment_ledger.expires_at（充值到账 + 12 个月）
-- 幂等设计：使用存储过程 guarded_add_column 防止列已存在导致迁移失败。

-- 幂等列添加辅助存储过程
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column;
CREATE PROCEDURE guarded_add_column(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD COLUMN `', col, '` ', col_def);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //

DELIMITER ;

CALL guarded_add_column('billing_payment_ledger', 'expires_at', 'TEXT NOT NULL DEFAULT ('''')');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- 已有台账回填：created_at + 12 months（SQLite datetime）
UPDATE billing_payment_ledger
-- MySQL compat: datetime()+|| converted to DATE_ADD()+CONCAT(); %i is MySQL minutes (not %M which is month name)
SET expires_at = CONCAT(DATE_FORMAT(DATE_ADD(SUBSTRING(created_at, 1, 19), INTERVAL 12 MONTH), '%Y-%m-%d %H:%i:%S'), '.000000')
WHERE expires_at = '';

-- 历史非付费充值（admin/grant/user_recharge）补写批次，便于 FEFO / 到期清零
INSERT INTO billing_payment_ledger (
  id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
  points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
)
SELECT
  bt.id,
  ba.tenant_id,
  bt.account_id,
  CASE
    WHEN bt.points_source_type = 'user_recharge_admin' THEN 'admin'
    WHEN bt.points_source_type = 'admin_grant' THEN 'grant'
    ELSE 'other'
  END,
  bt.transaction_id,
  '',
  bt.amount,
  bt.amount,
  bt.amount,
  'CNY',
  bt.id,
  bt.created_at,
-- MySQL compat: datetime()+|| converted to DATE_ADD()+CONCAT(); %i is MySQL minutes (not %M which is month name)
  CONCAT(DATE_FORMAT(DATE_ADD(SUBSTRING(bt.created_at, 1, 19), INTERVAL 12 MONTH), '%Y-%m-%d %H:%i:%S'), '.000000')
FROM billing_transaction bt
JOIN billing_account ba ON ba.id = bt.account_id
WHERE bt.transaction_type = 'recharge'
  AND bt.points_source_type IN ('user_recharge_admin', 'admin_grant', 'user_recharge')
  AND NOT EXISTS (
    SELECT 1 FROM billing_payment_ledger pl WHERE pl.billing_transaction_id = bt.id
  );

-- MySQL compat: MySQL does not support partial indexes (WHERE clause). Use functional index instead (8.0.13+).
-- NULL values in functional index are not indexed, achieving similar sparse-index effect.
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND INDEX_NAME = 'billing_payment_ledger_tenant_expires_active') = 0, 'CREATE INDEX billing_payment_ledger_tenant_expires_active ON billing_payment_ledger((CAST(CASE WHEN remaining_points > 0 THEN tenant_id ELSE NULL END AS CHAR(255))), (CAST(CASE WHEN remaining_points > 0 THEN expires_at ELSE NULL END AS CHAR(255))))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
