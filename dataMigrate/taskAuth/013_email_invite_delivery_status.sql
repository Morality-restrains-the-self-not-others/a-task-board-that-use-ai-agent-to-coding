-- 邮件投递状态追踪：记录邀请邮件是否成功送达
-- 状态值: delivered (已送达), failed (投递失败), pending (尚未发送)
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

CALL guarded_add_column('auth_email_registration_invite', 'delivery_status', 'varchar(20) NOT NULL DEFAULT ''pending'' COMMENT ''邮件投递状态: delivered/failed/pending''');

DROP PROCEDURE IF EXISTS guarded_add_column;
