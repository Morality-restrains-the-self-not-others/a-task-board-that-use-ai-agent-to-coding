-- OIDC Provider tables for SSO (gitService OmniAuth OIDC)
-- [MySQL compat] removed: PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS auth_oidc_client (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  client_id VARCHAR(255) NOT NULL UNIQUE,
  client_secret_hash VARCHAR(255) NOT NULL,
  name TEXT NOT NULL,
  redirect_uris TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_oidc_authorization (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  code VARCHAR(255) NOT NULL UNIQUE,
  client_id VARCHAR(255) NOT NULL REFERENCES auth_oidc_client (client_id),
  user_id TEXT NOT NULL,
  redirect_uri TEXT NOT NULL,
  scope TEXT NOT NULL,
  nonce TEXT,
  expires_at TEXT NOT NULL,
  used INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX idx_oidc_authorization_code ON auth_oidc_authorization (code);
CREATE INDEX idx_oidc_authorization_expires ON auth_oidc_authorization (expires_at(255));
