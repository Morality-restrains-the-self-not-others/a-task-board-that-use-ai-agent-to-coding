-- 008: 推荐资格可取消 + 操作审计表
-- Scale: 超管操作年增量 ≪ 10 万，合规保留不删；不分区。前缀 referral_。

DELIMITER //
DROP PROCEDURE IF EXISTS drop_referral_code_checks;
CREATE PROCEDURE drop_referral_code_checks()
BEGIN
  DECLARE done INT DEFAULT 0;
  DECLARE cname VARCHAR(64);
  DECLARE cur CURSOR FOR
    SELECT CONSTRAINT_NAME
    FROM information_schema.TABLE_CONSTRAINTS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'referral_code'
      AND CONSTRAINT_TYPE = 'CHECK';
  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;
  OPEN cur;
  read_loop: LOOP
    FETCH cur INTO cname;
    IF done THEN
      LEAVE read_loop;
    END IF;
    SET @ddl = CONCAT('ALTER TABLE `referral_code` DROP CHECK `', cname, '`');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END LOOP;
  CLOSE cur;
END //
DELIMITER ;

CALL drop_referral_code_checks();
DROP PROCEDURE IF EXISTS drop_referral_code_checks;

ALTER TABLE referral_code
  MODIFY COLUMN status VARCHAR(32) NOT NULL DEFAULT 'pending';

CREATE TABLE IF NOT EXISTS referral_qualification_audit (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  application_id BIGINT NOT NULL,
  user_id VARCHAR(255) NOT NULL,
  action VARCHAR(32) NOT NULL,
  reason VARCHAR(512) NOT NULL,
  operator_id VARCHAR(255) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  trace_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_referral_qualification_audit_app (application_id, created_at),
  INDEX idx_referral_qualification_audit_user (user_id, created_at),
  UNIQUE KEY uk_referral_qualification_audit_idem (idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
