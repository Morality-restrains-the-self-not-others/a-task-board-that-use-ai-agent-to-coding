-- Migration 034: Make order_id NOT NULL in billing_refund_application.
-- Migration 033 added the column as nullable; this tightens it now that the API
-- always requires an order_id on submission.
-- Idempotent: checks is_nullable before ALTER.

SET @nullable = (SELECT IS_NULLABLE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_refund_application' AND COLUMN_NAME = 'order_id');

SET @stmt = IF(@nullable = 'YES',
  'ALTER TABLE `billing_refund_application` MODIFY COLUMN `order_id` INTEGER NOT NULL',
  'SELECT 1 AS skipped_modify_order_id_not_null');
PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
