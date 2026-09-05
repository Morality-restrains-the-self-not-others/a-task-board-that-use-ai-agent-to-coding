-- 060: billing_gitlab_region 带宽共享与总/剩余带宽
-- 系统管理 GitLab 区域卡片需展示是否带宽共享分区、分区总带宽与剩余带宽（Mbps）。
-- 剩余为运维录入的包余量，不是租户配额汇总。

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
  'bandwidth_shared',
  'TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''是否带宽共享分区'''
);
CALL guarded_add_column(
  'billing_gitlab_region',
  'total_bandwidth_mbps',
  'BIGINT NOT NULL DEFAULT 0 COMMENT ''分区总带宽 (Mbps)'''
);
CALL guarded_add_column(
  'billing_gitlab_region',
  'remaining_bandwidth_mbps',
  'BIGINT NOT NULL DEFAULT 0 COMMENT ''剩余带宽 (Mbps)'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
