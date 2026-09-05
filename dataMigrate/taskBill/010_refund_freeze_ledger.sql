-- 退款申请：冻结余额、支付台账、退款申请表
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

CALL guarded_add_column('billing_account', 'frozen_balance', 'INTEGER NOT NULL DEFAULT 0');

DROP PROCEDURE IF EXISTS guarded_add_column;

CREATE TABLE IF NOT EXISTS billing_payment_ledger (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  account_id INTEGER NOT NULL,
  channel TEXT NOT NULL,
  provider_ref TEXT NOT NULL,
  provider_capture_id TEXT NOT NULL DEFAULT (''),
  points INTEGER NOT NULL,
  remaining_points INTEGER NOT NULL,
  amount_minor INTEGER NOT NULL,
  currency TEXT NOT NULL,
  billing_transaction_id INTEGER,
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND INDEX_NAME = 'billing_payment_ledger_tenant_created') = 0, 'CREATE INDEX billing_payment_ledger_tenant_created ON billing_payment_ledger(tenant_id, created_at(255))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS billing_refund_application (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  account_id INTEGER NOT NULL,
  applicant_user_id TEXT NOT NULL,
  frozen_points INTEGER NOT NULL,
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT (''),
  reviewer_user_id TEXT NOT NULL DEFAULT (''),
  review_note TEXT NOT NULL DEFAULT (''),
  payment_refund_refs TEXT NOT NULL DEFAULT ('[]'),
  reviewed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- MySQL compat: MySQL does not support partial unique indexes (WHERE clause). Use functional unique index (8.0.13+).
-- NULLs are not considered duplicate in unique indexes, so only pending rows with non-null tenant_id are constrained.
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND INDEX_NAME = 'uq_refund_pending_tenant') = 0, 'CREATE UNIQUE INDEX uq_refund_pending_tenant ON billing_refund_application((CASE WHEN status = ''pending'' THEN tenant_id ELSE NULL END))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- 回填历史付费充值台账
INSERT INTO billing_payment_ledger (
  id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
  points, remaining_points, amount_minor, currency, billing_transaction_id, created_at
)
SELECT
  bt.id,
  ba.tenant_id,
  bt.account_id,
  CASE
    WHEN bt.points_source_type = 'user_recharge_paypal' THEN 'paypal'
    WHEN bt.points_source_type = 'user_recharge_wechat' THEN 'wechat'
    ELSE 'unknown'
  END,
  CASE
    WHEN bt.transaction_id LIKE 'paypal:%' THEN SUBSTRING(bt.transaction_id, 8)
    WHEN bt.transaction_id LIKE 'wechat:%' THEN SUBSTRING(bt.transaction_id, 8)
    ELSE bt.transaction_id
  END,
  '',
  bt.amount,
  bt.amount,
  bt.amount,
  CASE
    WHEN bt.points_source_type = 'user_recharge_wechat' THEN 'CNY'
    WHEN bt.points_source_type = 'user_recharge_paypal' THEN 'USD'
    ELSE 'CNY'
  END,
  bt.id,
  bt.created_at
FROM billing_transaction bt
JOIN billing_account ba ON ba.id = bt.account_id
WHERE bt.points_source_type IN ('user_recharge_paypal', 'user_recharge_wechat')
  AND bt.transaction_type = 'recharge'
  AND NOT EXISTS (
    SELECT 1 FROM billing_payment_ledger pl WHERE pl.billing_transaction_id = bt.id
  );
