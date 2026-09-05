-- 003: 用户归档标记 + 搜索索引
-- 为已归档账号禁止登录提供数据库支持
-- 幂等设计：使用存储过程 guarded_add_column 防止列已存在导致迁移失败。

-- 幂等列添加辅助存储过程
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

CALL guarded_add_column('auth_user', 'is_archived', 'TINYINT(1) NOT NULL DEFAULT 0');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- 搜索索引：加速邮箱/手机号搜索
-- MySQL compat: CREATE INDEX is not supported in MySQL 8.0; use functional index syntax (MySQL 8.0.13+)
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'auth_login_method' AND INDEX_NAME = 'auth_login_method_identifier_search') = 0, 'CREATE INDEX auth_login_method_identifier_search ON auth_login_method ((LOWER(identifier)), method_type)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
