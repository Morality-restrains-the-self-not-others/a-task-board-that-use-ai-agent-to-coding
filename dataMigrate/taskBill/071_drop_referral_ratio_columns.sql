-- 071: 分成比例常量化后删除 billing_referral_config 比例列（OPT-20260825-012）
-- 读写路径已不再使用 profit_sharing_ratio_percent / referral_rate_percent：
--   getReferralConfig 只 SELECT settle_delay_days, updated_at；
--   updateReferralConfig / updateReferralSettleConfig 的 INSERT 已去掉这两列。
-- 必须先部署 taskBill/taskReferral 新包（INSERT 不再引用），再应用本迁移。
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_drop_column_071;
CREATE PROCEDURE guarded_drop_column_071(IN tbl VARCHAR(64), IN col VARCHAR(64))
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` DROP COLUMN `', col, '`');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_drop_column_071('billing_referral_config', 'profit_sharing_ratio_percent');
CALL guarded_drop_column_071('billing_referral_config', 'referral_rate_percent');

DROP PROCEDURE IF EXISTS guarded_drop_column_071;
