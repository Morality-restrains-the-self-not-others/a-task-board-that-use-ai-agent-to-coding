-- Referral commission config — settable by admin (owner: task-bill)
CREATE TABLE IF NOT EXISTS billing_referral_config (
  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
  settle_delay_days INTEGER NOT NULL DEFAULT 15,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO billing_referral_config (
  singleton_key, settle_delay_days, updated_at
) VALUES ('global', 15, NOW());
