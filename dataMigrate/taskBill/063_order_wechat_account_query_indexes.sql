-- 063: 管理端按微信关联账号查单 — user_id / pay_openid / pay_unionid 等值索引
-- listOrdersByWechatAccount 对这三列做 OR 等值匹配；无索引会全表扫描。

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

CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_user_id', 'user_id');
CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_pay_openid', 'pay_openid');
CALL guarded_add_index('billing_resource_order', 'idx_billing_resource_order_pay_unionid', 'pay_unionid');

DROP PROCEDURE IF EXISTS guarded_add_index;
