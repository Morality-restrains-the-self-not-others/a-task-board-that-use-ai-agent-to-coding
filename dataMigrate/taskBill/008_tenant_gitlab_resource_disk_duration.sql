-- 磁盘配额购买时长：月数 + 到期时间（按「积分/GB/月」预付）
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

CALL guarded_add_column('billing_tenant_gitlab_resource', 'disk_months', 'INTEGER NOT NULL DEFAULT 0');
CALL guarded_add_column('billing_tenant_gitlab_resource', 'disk_expires_at', 'TEXT NOT NULL DEFAULT ('''')');

DROP PROCEDURE IF EXISTS guarded_add_column;
