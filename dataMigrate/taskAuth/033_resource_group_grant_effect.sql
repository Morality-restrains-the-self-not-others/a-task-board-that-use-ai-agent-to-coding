-- ═══════════════════════════════════════════════════════════════
-- taskAuth: 资源组授予效果层 (v73 / ADR-0004)
-- view=可访问；operate=可编辑执行（蕴含 view）
-- 存量行 DEFAULT operate → 与 v72 全权授予行为连续
-- 幂等: guarded_add_column + data_migrate_log
-- ═══════════════════════════════════════════════════════════════

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_033;
CREATE PROCEDURE guarded_add_column_033(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_033(
  'auth_role_resource_group',
  'effect',
  'ENUM(''view'',''operate'') NOT NULL DEFAULT ''operate'''
);

DROP PROCEDURE IF EXISTS guarded_add_column_033;
