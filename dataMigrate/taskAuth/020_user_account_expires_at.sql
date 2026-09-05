-- 邮箱邀请增强配套：auth_user 增加账号有效期字段
-- account_expires_at: 注册时从邀请的 account_expires_at 复制过来，NULL=永久有效
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_020;
CREATE PROCEDURE guarded_add_column_020(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_020('auth_user', 'account_expires_at',
  'datetime DEFAULT NULL COMMENT ''账号有效期（NULL=永久有效，由邀请注册时设定）''');

DROP PROCEDURE IF EXISTS guarded_add_column_020;
