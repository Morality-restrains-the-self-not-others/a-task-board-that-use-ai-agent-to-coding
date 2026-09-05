-- 069: 超管带审计缘由发起分账（低频审计表，年增量远低于分区阈值）
CREATE TABLE IF NOT EXISTS billing_profit_sharing_admin_action (
  id BIGINT NOT NULL PRIMARY KEY,
  profit_sharing_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,
  actor_user_id VARCHAR(64) NOT NULL,
  impersonator_user_id VARCHAR(64) NOT NULL DEFAULT '',
  impersonation_session_id VARCHAR(64) NOT NULL DEFAULT '',
  reason VARCHAR(512) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  outcome VARCHAR(32) NOT NULL DEFAULT 'accepted',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE KEY uk_billing_ps_admin_action_idem (idempotency_key),
  KEY idx_billing_ps_admin_action_ps (profit_sharing_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
