-- 邮箱邀请增强：邀请原因、账号有效期、指定角色
-- 使用幂等存储过程 guarded_add_column 防止重复执行
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_019;
CREATE PROCEDURE guarded_add_column_019(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_019('auth_email_registration_invite', 'invite_reason',
  'varchar(500) DEFAULT NULL COMMENT ''邀请原因''');
CALL guarded_add_column_019('auth_email_registration_invite', 'account_expires_at',
  'datetime DEFAULT NULL COMMENT ''受邀账号有效期（NULL=永久）''');
CALL guarded_add_column_019('auth_email_registration_invite', 'assigned_role',
  'varchar(50) DEFAULT NULL COMMENT ''预设角色: superuser/staff/tenant/member(NULL=默认member)''');

DROP PROCEDURE IF EXISTS guarded_add_column_019;
