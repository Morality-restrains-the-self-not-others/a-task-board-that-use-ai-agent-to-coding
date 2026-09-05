-- 039: auth_user.is_tester — 测试角色（产品权限等同租户，非平台 RBAC）
-- 勾选测试时应用层强制 is_tenant=1；禁止写入 X-User-Roles。

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
  'auth_user',
  'is_tester',
  'TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''测试角色：等同租户产品权限，非平台员工'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
