-- 072: persist the outbound WeChat profit-sharing request trace id with fail_reason
-- so admin fail-reason cells can mount data-traceId (constraint 24).
-- Scale: one row per paid referred order; VARCHAR(64) only; no partition.

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
DELIMITER ;

CALL guarded_add_column(
  'billing_profit_sharing',
  'fail_trace_id',
  'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''trace_id of the request that last wrote fail_reason'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;
