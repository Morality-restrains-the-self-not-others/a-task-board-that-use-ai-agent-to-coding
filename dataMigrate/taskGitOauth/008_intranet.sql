-- 008: 自建 GitLab 内网标记。控制面探测失败时，intranet=1 视为预期而非故障。
-- 一行一租户，非时间累积表，无需分区。
-- 幂等：列已存在则跳过。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_gitoauth_008;
CREATE PROCEDURE guarded_add_column_gitoauth_008(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_gitoauth_008(
  'git_oauth_tenant_gitlab_oauth_connections',
  'intranet',
  'TINYINT(1) NOT NULL DEFAULT 0'
);

DROP PROCEDURE IF EXISTS guarded_add_column_gitoauth_008;
