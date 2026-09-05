-- 推荐佣金改为微信分账实现（2026-07-26）
-- 幂等设计：使用存储过程 guarded_add_column 防止列已存在导致迁移失败。

-- 幂等列添加辅助存储过程
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

-- 扩展推荐关系边：存储推荐人 openid（用于分账接收方）
CALL guarded_add_column('billing_referral_edge', 'referrer_openid', 'TEXT DEFAULT ('''')');

-- 扩展推荐关系边：存储被推荐人 openid
CALL guarded_add_column('billing_referral_edge', 'referred_openid', 'TEXT DEFAULT ('''')');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- 分账记录表
CREATE TABLE IF NOT EXISTS billing_profit_sharing (
    id INT AUTO_INCREMENT PRIMARY KEY,
    out_profit_sharing_no VARCHAR(255) NOT NULL UNIQUE,   -- 商户分账单号
    order_id INTEGER NOT NULL,                     -- 关联的资源订单ID
    order_number TEXT NOT NULL,                    -- 订单号（冗余，便于查询）
    tenant_id INTEGER NOT NULL,                    -- 租户ID
    referrer_user_id TEXT NOT NULL,                -- 推荐人用户ID
    referrer_openid TEXT NOT NULL,                 -- 推荐人微信openid（分账接收方）
    total_yuan_cents INTEGER NOT NULL,             -- 订单总额（分）
    commission_yuan_cents INTEGER NOT NULL,         -- 佣金金额（分）
    status TEXT NOT NULL DEFAULT ('pending'),         -- pending / processing / finished / failed
    wechat_profit_sharing_id TEXT DEFAULT (''),       -- 微信分账订单号（微信返回）
    settle_after TEXT NOT NULL,                     -- 最早可分账时间（订单完成+delay_days）
    settled_at TEXT,                                -- 实际分账完成时间
    fail_reason TEXT DEFAULT (''),                    -- 失败原因
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_profit_sharing' AND INDEX_NAME = 'idx_profit_sharing_status_settle') = 0, 'CREATE INDEX idx_profit_sharing_status_settle ON billing_profit_sharing (status(255), settle_after(255))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_profit_sharing' AND INDEX_NAME = 'idx_profit_sharing_order') = 0, 'CREATE INDEX idx_profit_sharing_order ON billing_profit_sharing (order_id)', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_profit_sharing' AND INDEX_NAME = 'idx_profit_sharing_referrer') = 0, 'CREATE INDEX idx_profit_sharing_referrer ON billing_profit_sharing (referrer_user_id(255))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
