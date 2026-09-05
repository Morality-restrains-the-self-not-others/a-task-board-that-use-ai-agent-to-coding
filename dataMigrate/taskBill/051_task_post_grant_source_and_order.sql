-- 051: 任务帖批次区分赠送/购买，并支持消耗归属到订单
-- billing_resource_grant.source_kind + order_id
-- billing_transaction.source_grant_id

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
  'billing_resource_grant',
  'source_kind',
  'VARCHAR(16) NOT NULL DEFAULT ''gift'' COMMENT ''gift=后台赠送 purchase=订单购买'''
);

CALL guarded_add_column(
  'billing_resource_grant',
  'order_id',
  'BIGINT NULL COMMENT ''归属的资源订单；购买批次必填'''
);

CALL guarded_add_column(
  'billing_transaction',
  'source_grant_id',
  'BIGINT NULL COMMENT ''配额消耗命中的 billing_resource_grant.id'''
);

CALL guarded_add_index('billing_resource_grant', 'bill_res_grant_order', 'tenant_id, order_id');
CALL guarded_add_index('billing_resource_grant', 'bill_res_grant_source', 'tenant_id, resource_type, source_kind');
CALL guarded_add_index('billing_transaction', 'bill_tra_source_grant', 'source_grant_id');

UPDATE billing_resource_grant
SET source_kind = 'gift'
WHERE source_kind IS NULL OR source_kind = '';

DROP PROCEDURE IF EXISTS guarded_add_column;
DROP PROCEDURE IF EXISTS guarded_add_index;
