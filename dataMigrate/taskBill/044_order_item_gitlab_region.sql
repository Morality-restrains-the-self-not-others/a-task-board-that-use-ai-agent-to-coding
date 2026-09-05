-- 044: 订单行项记录 GitLab 区域（可插拔多区域；禁止支付发放时静默落到 tencent-shanghai-5）

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

CALL guarded_add_column('billing_resource_order_item', 'region', 'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''GitLab 区域 slug；非 GitLab 行项为空''');

DROP PROCEDURE IF EXISTS guarded_add_column;
