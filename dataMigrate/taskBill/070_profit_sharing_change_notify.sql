-- 070: 微信分账动账通知收件箱（notify.id 去重）
CREATE TABLE IF NOT EXISTS billing_profit_sharing_change_notify (
  notify_id VARCHAR(64) NOT NULL PRIMARY KEY,
  out_order_no VARCHAR(64) NOT NULL DEFAULT '',
  wechat_transaction_id VARCHAR(64) NOT NULL DEFAULT '',
  wechat_order_id VARCHAR(64) NOT NULL DEFAULT '',
  event_type VARCHAR(32) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  KEY idx_billing_ps_change_notify_out (out_order_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
