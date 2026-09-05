-- Registration invite policy + codes (owner: task-auth)
CREATE TABLE IF NOT EXISTS auth_registration_invite_policy (
  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  daily_quota INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  updated_by TEXT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_registration_invite_code (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  code VARCHAR(255) NOT NULL UNIQUE,
  issuer_user_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('unused', 'used', 'revoked')),
  issued_day TEXT NOT NULL,
  created_at TEXT NOT NULL,
  redeemed_by_user_id TEXT NULL,
  redeemed_at TEXT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_registration_invite_code_issued_day
  ON auth_registration_invite_code (issued_day(255));

CREATE INDEX idx_registration_invite_code_issuer_created
  ON auth_registration_invite_code (issuer_user_id(255), created_at(255));

CREATE INDEX idx_registration_invite_code_status
  ON auth_registration_invite_code (status(255));

INSERT IGNORE INTO auth_registration_invite_policy (
  singleton_key, enabled, daily_quota, updated_at, updated_by
) VALUES ('global', 0, 0, NOW(), NULL);
