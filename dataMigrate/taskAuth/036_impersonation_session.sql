-- 独立模拟登录会话（ADR-0037）。不复用 auth_customtoken。
-- 年增量预估 < 10 万：热表 + ended_at/expires_at 索引；归档见 OPT。

CREATE TABLE IF NOT EXISTS auth_impersonation_session (
  id BIGINT NOT NULL,
  actor_user_id VARCHAR(36) NOT NULL,
  target_user_id VARCHAR(36) NOT NULL,
  token_key VARCHAR(64) NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL DEFAULT '',
  started_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  ended_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_auth_impersonation_token (token_key),
  KEY idx_auth_impersonation_actor_open (actor_user_id, ended_at, expires_at),
  KEY idx_auth_impersonation_idem (actor_user_id, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
