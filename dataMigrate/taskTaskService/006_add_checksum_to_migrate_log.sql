-- Add checksum column to data_migrate_log for schema parity with taskAuth/taskBill/taskReferral.
-- The tracking table is managed by the Go service (not by dataMigrate itself),
-- but this migration ensures existing DBs get the column added idempotently.
-- Safe to re-run: checks information_schema before ALTER.

SET @stmt = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'data_migrate_log' AND COLUMN_NAME = 'checksum') = 0,
  'ALTER TABLE data_migrate_log ADD COLUMN checksum VARCHAR(64) NOT NULL DEFAULT \'\'',
  'SELECT 1 \'skip: checksum column exists\''
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
