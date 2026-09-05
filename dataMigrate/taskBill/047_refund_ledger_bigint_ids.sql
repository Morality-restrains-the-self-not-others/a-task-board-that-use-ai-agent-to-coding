-- Migration 047: Widen refund/ledger ID columns to BIGINT for Snowflake IDs.
-- Root cause: billing_payment_ledger / billing_refund_application used INT,
-- so insertPaymentLedger / applyRefundApplication always failed with
-- "Out of range value for column 'id'" under STRICT_TRANS_TABLES.
-- Tables are empty in production as of 2026-08-19; ALTER is safe.
-- Idempotent: only MODIFY when COLUMN_TYPE is not already bigint.

-- billing_payment_ledger
SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND COLUMN_NAME = 'id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_payment_ledger` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT',
  'SELECT 1 AS skipped_ledger_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND COLUMN_NAME = 'tenant_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_payment_ledger` MODIFY COLUMN `tenant_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_ledger_tenant_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND COLUMN_NAME = 'account_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_payment_ledger` MODIFY COLUMN `account_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_ledger_account_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_payment_ledger' AND COLUMN_NAME = 'billing_transaction_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_payment_ledger` MODIFY COLUMN `billing_transaction_id` BIGINT NULL',
  'SELECT 1 AS skipped_ledger_txn_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- billing_refund_application
SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_refund_application` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT',
  'SELECT 1 AS skipped_refund_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'tenant_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_refund_application` MODIFY COLUMN `tenant_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_refund_tenant_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'account_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_refund_application` MODIFY COLUMN `account_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_refund_account_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @t = (SELECT COLUMN_TYPE FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'order_id');
SET @stmt = IF(@t IS NOT NULL AND @t NOT LIKE 'bigint%',
  'ALTER TABLE `billing_refund_application` MODIFY COLUMN `order_id` BIGINT NOT NULL',
  'SELECT 1 AS skipped_refund_order_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Recreate pending-tenant unique index after BIGINT widen (stale INT functional index → Error 3752)
SET @idx = (SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application'
    AND INDEX_NAME = 'uq_refund_pending_tenant');
SET @stmt = IF(@idx > 0,
  'DROP INDEX uq_refund_pending_tenant ON billing_refund_application',
  'SELECT 1 AS skipped_drop_uq_refund_pending_tenant');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = 'CREATE UNIQUE INDEX uq_refund_pending_tenant ON billing_refund_application((CASE WHEN status = ''pending'' THEN tenant_id ELSE NULL END))';
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Unique: one non-rejected refund application per order (pending or approved)
SET @idx = (SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application'
    AND INDEX_NAME = 'uq_refund_order_active');
SET @stmt = IF(@idx = 0,
  'CREATE UNIQUE INDEX uq_refund_order_active ON billing_refund_application((CASE WHEN status IN (''pending'',''approved'') THEN order_id ELSE NULL END))',
  'SELECT 1 AS skipped_uq_refund_order_active');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
