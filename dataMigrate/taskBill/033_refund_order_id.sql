-- Migration 033: Add order_id column to billing_refund_application for per-order refund tracking.
-- Idempotent: checks information_schema before ALTER to prevent duplicate column errors.

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'order_id');

SET @stmt = IF(@col_exists = 0,
  'ALTER TABLE `billing_refund_application` ADD COLUMN `order_id` INTEGER DEFAULT NULL',
  'SELECT 1 AS skipped_add_order_id');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

-- Add index for looking up refund applications by order (if not already present)
SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND INDEX_NAME = 'idx_refund_order');

SET @stmt = IF(@idx_exists = 0,
  'CREATE INDEX idx_refund_order ON billing_refund_application(order_id)',
  'SELECT 1 AS skipped_idx_refund_order');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
