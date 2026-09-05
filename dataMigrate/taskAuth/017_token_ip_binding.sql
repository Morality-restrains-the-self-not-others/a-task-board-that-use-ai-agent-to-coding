-- 017_token_ip_binding: authToken IP binding for XSS theft mitigation
-- Binds each auth_customtoken to the client IP at creation time,
-- and tracks the last IP used for successful auth.
-- 幂等: guarded_add_column + data_migrate_log（与 taskAuth/033 同模式）。
-- 若日志被清且列已存在时重跑，裸 ADD COLUMN 会失败；守卫下为 no-op。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_017;
CREATE PROCEDURE guarded_add_column_017(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_017('auth_customtoken', 'client_ip', 'VARCHAR(45) NOT NULL DEFAULT '''' COMMENT ''Client IP at token creation; used for IP-binding verification''');
CALL guarded_add_column_017('auth_customtoken', 'last_ip', 'VARCHAR(45) NOT NULL DEFAULT '''' COMMENT ''Last client IP that successfully authenticated with this token''');

DROP PROCEDURE IF EXISTS guarded_add_column_017;

-- Existing tokens with empty IP are treated as grandfathered:
-- resolveTokenUserIDWithIP skips IP check when stored IP is empty.
