-- 邮件投递失败原因记录：存储投递失败的具体错误信息
-- 用于前端 tooltip 展示，帮助管理员了解失败原因
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

CALL guarded_add_column('auth_email_registration_invite', 'delivery_error', 'TEXT DEFAULT NULL COMMENT ''投递失败原因：Kafka/SMTP 错误详情''');

DROP PROCEDURE IF EXISTS guarded_add_column;
