-- 066: 开票申请区分专票/普票；手动开具的发票文件挂到发票行
-- 低频合规表，年增量远低于 100 万，不分分区。
-- 幂等：guarded_add_column；MySQL 8 加列可 INPLACE（含默认值）。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_066;
CREATE PROCEDURE guarded_add_column_066(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_066(
  'billing_invoice_application',
  'invoice_type',
  'VARCHAR(16) NOT NULL DEFAULT ''general'' AFTER buyer_type'
);
CALL guarded_add_column_066(
  'billing_invoice_application',
  'invoice_file_path',
  'VARCHAR(512) NOT NULL DEFAULT '''' AFTER bank_account'
);
CALL guarded_add_column_066(
  'billing_invoice',
  'invoice_file_path',
  'VARCHAR(512) NOT NULL DEFAULT '''' AFTER wechat_fapiao_number'
);

DROP PROCEDURE IF EXISTS guarded_add_column_066;
