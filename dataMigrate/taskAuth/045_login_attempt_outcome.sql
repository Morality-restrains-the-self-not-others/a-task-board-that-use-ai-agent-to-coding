-- 登录结果留痕（OPT-20260825-035）：给 auth_login_history 加 outcome 列。
-- 成功认证沿用默认 'success'；失败尝试（错密/邮箱未验证/账号未激活/入口不匹配）
-- 写入对应 outcome，账号中心默认列表只展示成功，可选含失败。
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

CALL guarded_add_column('auth_login_history', 'outcome', 'varchar(24) NOT NULL DEFAULT ''success'' COMMENT ''登录结果: success/password_mismatch/email_unverified/not_active/admin_entry_mismatch/not_admin_staff''');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- 幂等索引添加：默认列表查询 user_id + outcome + logged_in_at DESC 走该索引
SET @has_idx := (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'auth_login_history'
    AND INDEX_NAME = 'idx_auth_login_history_user_outcome_time'
);
SET @ddl_idx := IF(@has_idx = 0,
  'ALTER TABLE auth_login_history ADD KEY idx_auth_login_history_user_outcome_time (user_id, outcome, logged_in_at)',
  'SELECT 1'
);
PREPARE stmt_idx FROM @ddl_idx;
EXECUTE stmt_idx;
DEALLOCATE PREPARE stmt_idx;
