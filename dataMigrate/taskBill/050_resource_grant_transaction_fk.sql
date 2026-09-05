-- 050: admin_grant 汇总流水与资源赠送/订单建立稳定外键
-- 替代 enrich 的 created_at 秒级匹配：新写入的 billing_resource_grant 带
-- billing_transaction_id，billing_transaction 带 related_order_id；旧行保留
-- 秒级兜底（见 transaction_change_enrich.go）。

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

CALL guarded_add_column(
  'billing_transaction',
  'related_order_id',
  'BIGINT NULL COMMENT ''资源订单 id；admin_grant 汇总流水关联赠送订单'''
);

CALL guarded_add_column(
  'billing_resource_grant',
  'billing_transaction_id',
  'BIGINT NULL COMMENT ''admin_grant 汇总流水 id；替代 created_at 秒级匹配'''
);

CALL guarded_add_index('billing_transaction', 'bill_tra_related_order', 'related_order_id');
CALL guarded_add_index('billing_resource_grant', 'bill_res_grant_txn', 'billing_transaction_id');

DROP PROCEDURE IF EXISTS guarded_add_column;
DROP PROCEDURE IF EXISTS guarded_add_index;
