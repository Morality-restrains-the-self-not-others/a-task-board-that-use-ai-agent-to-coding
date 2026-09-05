-- taskTenantService: 邀请可复用角色预绑（v75 角色优先）
-- 语义：JSON 数组 ["部署工程师", "只读"]，join 落权时直接绑定 tenant_member_role；
-- pending_grants 保留为兼容（deprecated-to-roles）。role=admin 入职走 tenant_admin，忽略本列。
-- 幂等: guarded_add_column + data_migrate_log（与 taskTenantService/009 同模式）。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_010;
CREATE PROCEDURE guarded_add_column_010(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_010(
  'tenant_invitation',
  'pending_role_names',
  'JSON NULL'
);

DROP PROCEDURE IF EXISTS guarded_add_column_010;
