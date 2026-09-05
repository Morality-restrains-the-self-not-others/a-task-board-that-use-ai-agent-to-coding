-- 055: billing_resource_order.payment_ref 等值查询索引
-- listOrdersByTradeNo 对 payment_ref = ? 做精确匹配（管理端按交易单号查单）；
-- 订单累积后无索引会退化成全表扫描，补前缀索引避免锁表迁移重演（OPT-20260821-038）。

DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_index;
CREATE PROCEDURE guarded_add_index(IN tbl VARCHAR(64), IN idx VARCHAR(64), IN cols VARCHAR(255))
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD INDEX `', idx, '` (', cols, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

-- payment_ref 为 TEXT，需前缀长度；utf8mb4 下 191 字符索引键字节安全，
-- 微信 "wechat:WX..." / PayPal 等真实单号均远短于此。
CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_payment_ref', 'payment_ref(191)');

DROP PROCEDURE IF EXISTS guarded_add_index;
