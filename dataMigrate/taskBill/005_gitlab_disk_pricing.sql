-- GitLab 磁盘价目：套餐「积分/GB/月」+ 账户锁价 + billing_unit 标签
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

CALL guarded_add_column('billing_pricing_package', 'gitlab_disk_points_per_gb_per_month', 'bigint NOT NULL DEFAULT 1');
CALL guarded_add_column('billing_account', 'locked_gitlab_disk_points_per_gb_per_month', 'bigint NOT NULL DEFAULT 1');

DROP PROCEDURE IF EXISTS guarded_add_column;

INSERT IGNORE INTO billing_unit (id, unit_type, name, price, unit, is_active, created_at, updated_at)
VALUES (1000000000000000005, 'gitlab_disk', 'GitLab 磁盘', 1, 'GB/月', 1, NOW(), NOW());
