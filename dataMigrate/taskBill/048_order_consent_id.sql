-- 048: 资源订单关联支付服务条款签署记录（consent_id），供管理端审计与支付门禁回写
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
  'consent_id',
  'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''支付服务条款签署记录 ID；用户付费下单时写入'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
