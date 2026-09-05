-- 061: 订单电子发票（申请 + 蓝/红/重开票面）
-- 低频合规表，年增量远低于 100 万，不分分区（冷热清单：无时间累积压力）。
-- 主键 Snowflake；无物理外键；字符集 utf8mb4。

CREATE TABLE IF NOT EXISTS billing_invoice_application (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,
  applicant_user_id VARCHAR(64) NOT NULL DEFAULT '',
  buyer_type VARCHAR(16) NOT NULL,
  buyer_name VARCHAR(256) NOT NULL,
  taxpayer_id VARCHAR(32) NOT NULL DEFAULT '',
  address VARCHAR(128) NOT NULL DEFAULT '',
  telephone VARCHAR(32) NOT NULL DEFAULT '',
  bank_name VARCHAR(128) NOT NULL DEFAULT '',
  bank_account VARCHAR(32) NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL,
  reviewer_user_id VARCHAR(64) NOT NULL DEFAULT '',
  review_note VARCHAR(512) NOT NULL DEFAULT '',
  reviewed_at VARCHAR(64) NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_billing_invoice_app_order (order_id),
  INDEX idx_billing_invoice_app_tenant (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 一单一 pending 申请（MySQL 8 函数唯一索引；非 pending 的 order_id 表达式为 NULL，不冲突）
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_invoice_application' AND INDEX_NAME = 'uq_invoice_app_order_pending') = 0, 'CREATE UNIQUE INDEX uq_invoice_app_order_pending ON billing_invoice_application((CASE WHEN status = ''pending'' THEN order_id ELSE NULL END))', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

CREATE TABLE IF NOT EXISTS billing_invoice (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,
  application_id BIGINT NOT NULL DEFAULT 0,
  related_invoice_id BIGINT NULL,
  kind VARCHAR(8) NOT NULL,
  purpose VARCHAR(16) NOT NULL,
  status VARCHAR(24) NOT NULL,
  amount_yuan_cents BIGINT NOT NULL,
  fapiao_id VARCHAR(32) NOT NULL,
  wechat_apply_id VARCHAR(64) NOT NULL DEFAULT '',
  wechat_fapiao_number VARCHAR(32) NOT NULL DEFAULT '',
  buyer_snapshot TEXT NOT NULL,
  fail_reason VARCHAR(512) NOT NULL DEFAULT '',
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_billing_invoice_order (order_id),
  INDEX idx_billing_invoice_tenant (tenant_id),
  INDEX idx_billing_invoice_apply_id (wechat_apply_id),
  UNIQUE INDEX uq_billing_invoice_fapiao_id (fapiao_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
