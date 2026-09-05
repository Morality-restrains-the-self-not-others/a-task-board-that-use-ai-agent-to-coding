-- 065: billing_gitlab_region.access_mode — 发布/开发模式
-- 存量默认 release，生产租户无感；development 仅测试账号可在 SaaS 侧使用。

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
  'billing_gitlab_region',
  'access_mode',
  'VARCHAR(16) NOT NULL DEFAULT ''release'' COMMENT ''release=发布模式 development=开发模式'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
