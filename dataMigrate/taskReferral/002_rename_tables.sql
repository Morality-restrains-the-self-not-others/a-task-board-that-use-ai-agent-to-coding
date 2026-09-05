-- Migration 002: Rename tables to conform to service prefix convention.
-- Rule: taskReferral → referral_ prefix.
-- Guarded: skips if old table gone or new table already exists (idempotent across restarts).

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'accounts_referral_code')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'referral_code'),
  'ALTER TABLE `accounts_referral_code` RENAME TO `referral_code`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;

SET @stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'accounts_referral_policy')
  AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'referral_policy'),
  'ALTER TABLE `accounts_referral_policy` RENAME TO `referral_policy`',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
