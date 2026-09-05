-- 031_oidc_client_managed_by.sql — OIDC client 管理权属落 DB（OPT-20260808-025）
-- 用途: auth_oidc_client 新增 managed_by 列（'bootstrap' | 'admin'），seed 按行判定：
--       admin 行 INSERT-only 不 UPDATE（管理员托管，以 DB 为准）；bootstrap 行维持
--       conf 自愈（行为不变）。chrome-extension 行置 admin（OPT-024 固定 ID 运维接管）。
-- 幂等: guarded_add_column 存储过程防护列已存在；UPDATE 无匹配行零影响。
--       data_migrate_log 按文件名去重，每库仅执行一次（重跑由存储过程幂等兜底）。

-- 幂等列添加辅助存储过程（与 024_pkce.sql 同模式）
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_031;
CREATE PROCEDURE guarded_add_column_031(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_031('auth_oidc_client', 'managed_by', 'VARCHAR(32) NOT NULL DEFAULT ''bootstrap''');

-- 固定 ID 扩展（chrome-extension，OPT-024）置 admin 托管：管理员改库后 seed 不再自愈覆盖
UPDATE auth_oidc_client SET managed_by = 'admin' WHERE client_id = 'chrome-extension';

DROP PROCEDURE IF EXISTS guarded_add_column_031;
