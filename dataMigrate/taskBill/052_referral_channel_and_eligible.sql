-- 052: 推荐边快照渠道码与分成资格；计提冗余 channel_code
-- 旧边 fail-closed：commission_eligible 默认 0；已有计提的边回填为 1

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column;
CREATE PROCEDURE guarded_add_column(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
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

DROP PROCEDURE IF EXISTS guarded_add_index;
CREATE PROCEDURE guarded_add_index(IN tbl VARCHAR(64), IN idx VARCHAR(64), IN cols VARCHAR(255))
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD INDEX `', idx, '` (', cols, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_add_column('billing_referral_edge', 'channel_code', 'VARCHAR(32) NOT NULL DEFAULT ''''');
CALL guarded_add_column('billing_referral_edge', 'commission_eligible', 'TINYINT(1) NOT NULL DEFAULT 0');
CALL guarded_add_index('billing_referral_edge', 'idx_referral_edge_referrer_channel', 'referrer_user_id, channel_code, bound_at');

CALL guarded_add_column('billing_referral_commission_accrual', 'channel_code', 'VARCHAR(32) NOT NULL DEFAULT ''''');
CALL guarded_add_index('billing_referral_commission_accrual', 'idx_referral_accrual_referrer_channel', 'referrer_user_id, channel_code, consumed_at');

UPDATE billing_referral_edge e
INNER JOIN billing_referral_commission_accrual a
  ON a.referred_user_id = e.referred_user_id
SET e.commission_eligible = 1
WHERE e.commission_eligible = 0;

UPDATE billing_referral_commission_accrual a
INNER JOIN billing_referral_edge e
  ON e.referred_user_id = a.referred_user_id
SET a.channel_code = e.channel_code
WHERE a.channel_code = '' AND e.channel_code <> '';

DROP PROCEDURE IF EXISTS guarded_add_column;
DROP PROCEDURE IF EXISTS guarded_add_index;
