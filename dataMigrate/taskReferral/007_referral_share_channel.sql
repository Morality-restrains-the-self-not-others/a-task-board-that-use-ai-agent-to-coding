-- 007: 一用户多渠道分享码（去掉 UNIQUE user_id，增加渠道名/默认/状态）
-- Scale: 每用户 ≤20 行，非时间累积。无 RANGE 分区。

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

DROP PROCEDURE IF EXISTS guarded_drop_index;
CREATE PROCEDURE guarded_drop_index(IN tbl VARCHAR(64), IN idx VARCHAR(64))
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` DROP INDEX `', idx, '`');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //

DROP PROCEDURE IF EXISTS guarded_add_unique;
CREATE PROCEDURE guarded_add_unique(IN tbl VARCHAR(64), IN idx VARCHAR(64), IN cols VARCHAR(255))
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD UNIQUE KEY `', idx, '` (', cols, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_drop_index('referral_share_code', 'uk_referral_share_code_user');
CALL guarded_add_column('referral_share_code', 'channel_name', 'VARCHAR(32) NOT NULL DEFAULT ''默认''');
CALL guarded_add_column('referral_share_code', 'is_default', 'TINYINT(1) NOT NULL DEFAULT 1');
CALL guarded_add_column('referral_share_code', 'status', 'VARCHAR(16) NOT NULL DEFAULT ''active''');
CALL guarded_add_unique('referral_share_code', 'uk_referral_share_code_user_channel', 'user_id, channel_name');

UPDATE referral_share_code
SET channel_name = '默认', is_default = 1, status = 'active'
WHERE channel_name = '' OR channel_name IS NULL;

DROP PROCEDURE IF EXISTS guarded_add_column;
DROP PROCEDURE IF EXISTS guarded_drop_index;
DROP PROCEDURE IF EXISTS guarded_add_unique;
