-- 邮件邀请送达追踪：记录实际发送时间和发送尝试次数
-- 用于诊断"邀请已创建但邮件未送达"的问题
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

CALL guarded_add_column('auth_email_registration_invite', 'email_sent_at', 'datetime NULL COMMENT ''最后一次邮件发送时间''');
CALL guarded_add_column('auth_email_registration_invite', 'email_send_attempts', 'int NOT NULL DEFAULT 0 COMMENT ''邮件发送尝试次数''');

DROP PROCEDURE IF EXISTS guarded_add_column;
