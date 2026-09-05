-- 043: 租户自建 GitLab OIDC SSO client（ADR-0043）
-- auth_oidc_client 增加 owner_company_id / purpose；managed_by 可取 tenant。
-- bootstrap seed 不得 UPDATE managed_by=tenant 行。
-- 幂等表记录 enable/rotate/disable 的 Idempotency-Key（密钥不明文落库）。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column_043;
CREATE PROCEDURE guarded_add_column_043(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_043('auth_oidc_client', 'owner_company_id', 'VARCHAR(64) NULL');
CALL guarded_add_column_043('auth_oidc_client', 'purpose', 'VARCHAR(64) NOT NULL DEFAULT ''''');

DROP PROCEDURE IF EXISTS guarded_add_column_043;

CREATE TABLE IF NOT EXISTS auth_oidc_sso_idempotency (
  company_id VARCHAR(64) NOT NULL,
  operation VARCHAR(32) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (company_id, operation, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-gitlab-sso-get', 'rg-reg-settings-gitlab-main', 'api', 'GET /api/tenant/{tid}/gitlab-oidc-sso/'),
('rm-gitlab-sso-put', 'rg-reg-settings-gitlab-main', 'api', 'PUT /api/tenant/{tid}/gitlab-oidc-sso/'),
('rm-gitlab-sso-rotate', 'rg-reg-settings-gitlab-main', 'api', 'POST /api/tenant/{tid}/gitlab-oidc-sso/rotate/'),
('rm-gitlab-sso-delete', 'rg-reg-settings-gitlab-main', 'api', 'DELETE /api/tenant/{tid}/gitlab-oidc-sso/');
