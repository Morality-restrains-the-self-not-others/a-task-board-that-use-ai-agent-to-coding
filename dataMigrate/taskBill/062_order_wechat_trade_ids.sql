-- 062: 资源订单保存微信支付商户订单号与微信支付单号，供管理端按微信账单字段查单。
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

CALL guarded_add_column(
  'billing_resource_order',
  'out_trade_no',
  'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''微信支付商户订单号 out_trade_no'''
);

CALL guarded_add_column(
  'billing_resource_order',
  'wechat_transaction_id',
  'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''微信支付单号 transaction_id'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_index;
CREATE PROCEDURE guarded_add_index(IN tbl VARCHAR(64), IN idx VARCHAR(64), IN cols VARCHAR(255))
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD INDEX `', idx, '` (', cols, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_out_trade_no', 'out_trade_no');
CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_wechat_txn_id', 'wechat_transaction_id');

DROP PROCEDURE IF EXISTS guarded_add_index;

UPDATE billing_resource_order
SET out_trade_no = SUBSTRING(payment_ref, 8)
WHERE payment_method = 'wechat'
  AND payment_ref LIKE 'wechat:%'
  AND out_trade_no = '';

UPDATE billing_resource_order o
INNER JOIN billing_payment_ledger l
  ON l.channel = 'wechat'
 AND l.provider_ref IN (
      o.payment_ref,
      CASE WHEN o.payment_ref LIKE 'wechat:%' THEN SUBSTRING(o.payment_ref, 8) ELSE o.payment_ref END
    )
SET o.wechat_transaction_id = l.provider_capture_id
WHERE o.wechat_transaction_id = ''
  AND COALESCE(l.provider_capture_id, '') <> '';
