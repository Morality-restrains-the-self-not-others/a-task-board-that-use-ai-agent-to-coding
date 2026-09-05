-- 077: 资源订单行项记录购买月数（gitlab_disk 专用）
-- 背景：createOrder 计价忽略 disk_months，导致展示金额(单价×GB×月数)与实扣(单价×GB)分叉；
-- markOrderPaid 也无从得知订单行月数，只能硬编码 disk_months=1。
-- 本迁移为 billing_resource_order_item 补 disk_months 列，使后续发放/开通可按订单行月数记账
-- 与计算到期日（OPT-20260903-010 / OPT-20260903-012）。
-- 旧行默认 0，代码层按 1 兜底，语义与旧行为一致（仅收 1 个月款则按 1 个月发放）。

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

CALL guarded_add_column('billing_resource_order_item', 'disk_months', 'INTEGER NOT NULL DEFAULT 0');

DROP PROCEDURE IF EXISTS guarded_add_column;
