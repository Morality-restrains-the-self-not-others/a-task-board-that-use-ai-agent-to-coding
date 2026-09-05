-- 全局退款申请开关（平台级，单行）
CREATE TABLE IF NOT EXISTS billing_refund_policy (
  id INT AUTO_INCREMENT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO billing_refund_policy (id, enabled, updated_at)
VALUES (1, 1, NOW());
