-- taskTenantService: 邀请预授 page/region grants（v74）
-- 语义：[{"group_key":"people.access.subject_list","effect":"operate"}, ...]
-- NULL / [] = 无预授；role=admin 入职走 tenant_admin，忽略本列
-- 幂等: guarded_add_column + data_migrate_log（与 taskAuth/033 同模式）。
-- 若 data_migrate_log 被清且列已存在时重跑，裸 ADD COLUMN 会失败；守卫下为 no-op。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_009;
CREATE PROCEDURE guarded_add_column_009(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_009(
  'tenant_invitation',
  'pending_grants',
  'JSON NULL'
);

DROP PROCEDURE IF EXISTS guarded_add_column_009;
