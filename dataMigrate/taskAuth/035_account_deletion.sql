-- 035: 用户账号注销申请与审计（PIPL 合规生命周期）
CREATE TABLE IF NOT EXISTS auth_account_deletion_request (
  id BIGINT NOT NULL PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending_cooldown',
  requested_at DATETIME NOT NULL,
  effective_at DATETIME NOT NULL,
  cancelled_at DATETIME NULL,
  executed_at DATETIME NULL,
  last_precheck_snapshot JSON NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_auth_account_deletion_effective (status, effective_at),
  INDEX idx_auth_account_deletion_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_account_deletion_audit (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  request_id BIGINT NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  step VARCHAR(64) NOT NULL,
  detail_redacted VARCHAR(512) NOT NULL DEFAULT '',
  trace_id VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_auth_account_deletion_audit_request (request_id),
  INDEX idx_auth_account_deletion_audit_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 幂等列添加：注销完成时间戳
DROP PROCEDURE IF EXISTS guarded_add_column_auth_deletion;
DELIMITER //
CREATE PROCEDURE guarded_add_column_auth_deletion(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

CALL guarded_add_column_auth_deletion('auth_user', 'deletion_completed_at', 'DATETIME NULL DEFAULT NULL');
DROP PROCEDURE IF EXISTS guarded_add_column_auth_deletion;
