-- Migration 032: Rename tables to conform to service prefix convention.
-- Rule: taskBill → billing_ prefix.
-- Guarded: skips if old table gone or new table already exists (idempotent across restarts).
-- Note: legal tables (license_agreements, privacy_policies, etc.) are now renamed in 022_legal_agreements.sql
--       to guarantee rename-before-CREATE ordering — data migration happens before DDL in a single step.

SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='referral_edge') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_referral_edge'),'ALTER TABLE `referral_edge` RENAME TO `billing_referral_edge`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='referral_commission_accrual') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_referral_commission_accrual'),'ALTER TABLE `referral_commission_accrual` RENAME TO `billing_referral_commission_accrual`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='referral_config') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_referral_config'),'ALTER TABLE `referral_config` RENAME TO `billing_referral_config`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
SET @stmt = IF(EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='gitlab_region') AND NOT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='billing_gitlab_region'),'ALTER TABLE `gitlab_region` RENAME TO `billing_gitlab_region`','SELECT 1');PREPARE s FROM @stmt;EXECUTE s;DEALLOCATE PREPARE s;
