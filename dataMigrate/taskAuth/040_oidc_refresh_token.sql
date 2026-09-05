-- OIDC refresh token storage (offline_access grant)
-- token 仅存 SHA-256 哈希（sha256Hash(tokenPlain)，格式 hex），绝不明文落库。
-- 轮换制：每次 refresh_token grant 消费后 revoked=1 并签发新 token（单次使用）。
-- 到期行由 storeRefreshToken / consumeRefreshToken 惰性清理（不额外建定时任务）。

CREATE TABLE IF NOT EXISTS auth_oidc_refresh_token (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  client_id VARCHAR(255) NOT NULL,
  user_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  last_used_at TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX idx_oidc_refresh_token_hash ON auth_oidc_refresh_token (token_hash);
CREATE INDEX idx_oidc_refresh_token_client ON auth_oidc_refresh_token (client_id, user_id(255));
CREATE INDEX idx_oidc_refresh_token_expires ON auth_oidc_refresh_token (expires_at(255));
