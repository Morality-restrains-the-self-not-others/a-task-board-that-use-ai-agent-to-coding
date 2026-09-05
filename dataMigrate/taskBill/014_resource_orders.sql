-- 014: 资源订单系统 — 用户创建订单选择具体资源数量，支付后发放到配额
-- 替代旧的「充值→积分余额→扣减」模型，改为「订单→支付→资源配额→扣减」

-- 资源订单主表
CREATE TABLE IF NOT EXISTS billing_resource_order (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  order_number VARCHAR(64) UNIQUE NOT NULL,
  status TEXT NOT NULL DEFAULT ('pending'),  -- pending | paid | cancelled | expired
  total_yuan_cents INTEGER NOT NULL,       -- 总价，单位：分
  payment_method TEXT,                     -- wechat | paypal
  payment_ref TEXT,                        -- 支付网关返回的流水号
  created_at TEXT NOT NULL,
  paid_at TEXT,
  cancelled_at TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 订单行项
CREATE TABLE IF NOT EXISTS billing_resource_order_item (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES billing_resource_order(id),
  resource_type TEXT NOT NULL,             -- task_post | gitlab_disk | gitlab_traffic
  quantity INTEGER NOT NULL,               -- 数量
  unit_price_yuan_cents INTEGER NOT NULL,  -- 单价，单位：分
  subtotal_yuan_cents INTEGER NOT NULL,    -- 小计，单位：分
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- billing_account 增加任务帖预购配额（幂等：使用 guarded_add_column 防止列已存在导致迁移失败）
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

CALL guarded_add_column('billing_account', 'task_post_quota', 'INTEGER NOT NULL DEFAULT 0');

DROP PROCEDURE IF EXISTS guarded_add_column;
