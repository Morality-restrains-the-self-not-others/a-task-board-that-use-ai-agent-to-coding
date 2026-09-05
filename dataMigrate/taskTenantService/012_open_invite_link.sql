-- taskTenantService: 开放式邀请链接
-- link_kind=single（默认，一链一人）| open（一链多人直至 max_uses 或过期）
-- max_uses: single 强制 1；open 时 0=不限人数
-- 幂等: guarded_add_column + CREATE TABLE IF NOT EXISTS

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_012;
CREATE PROCEDURE guarded_add_column_012(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_012(
  'tenant_invitation',
  'link_kind',
  'VARCHAR(16) NOT NULL DEFAULT ''single'''
);
CALL guarded_add_column_012(
  'tenant_invitation',
  'max_uses',
  'INT NOT NULL DEFAULT 1'
);
CALL guarded_add_column_012(
  'tenant_invitation',
  'use_count',
  'INT NOT NULL DEFAULT 0'
);

DROP PROCEDURE IF EXISTS guarded_add_column_012;

CREATE TABLE IF NOT EXISTS tenant_invitation_redemption (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    invitation_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    member_id VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_tenant_invitation_redemption_invite_user (invitation_id, user_id),
    INDEX idx_tir_company (company_id),
    INDEX idx_tir_invitation (invitation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
