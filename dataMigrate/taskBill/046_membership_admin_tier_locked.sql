-- 046: 管理员在赠送页设定会员等级后锁定，避免累计消费自动升级覆盖运维决定
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
  'billing_membership',
  'admin_tier_locked',
  'TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''1=管理员锁定等级，跳过累计消费自动升级'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
