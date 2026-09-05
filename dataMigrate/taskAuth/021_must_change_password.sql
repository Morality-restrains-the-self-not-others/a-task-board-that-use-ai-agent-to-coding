-- 021: 用户密码强制修改标记
-- 新增 must_change_password 列，标记需要强制修改密码的用户（如初始超级管理员）。
-- 登录后如该标记为 1，前端应引导用户到修改密码页面。
-- 幂等：使用 guarded_add_column 存储过程防止列已存在导致迁移失败。

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

CALL guarded_add_column('auth_user', 'must_change_password', 'TINYINT(1) NOT NULL DEFAULT 0');

DROP PROCEDURE IF EXISTS guarded_add_column;
