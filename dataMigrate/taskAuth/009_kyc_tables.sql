-- KYC identity tier + audit + AML screening + limit policy (owner: task-auth)
CREATE TABLE IF NOT EXISTS auth_kyc_profile (
  user_id VARCHAR(64) PRIMARY KEY,
  tier TEXT NOT NULL DEFAULT ('T0_unverified'),
  status TEXT NOT NULL DEFAULT ('none'),
  effective_at TEXT,
  expires_at TEXT,
  risk_flags TEXT NOT NULL DEFAULT (''),
  updated_at TEXT NOT NULL,
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_kyc_audit_log (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id TEXT NOT NULL,
  old_tier TEXT NOT NULL,
  new_tier TEXT NOT NULL,
  old_status TEXT NOT NULL,
  new_status TEXT NOT NULL,
  trigger_source TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT (''),
  reason_code TEXT NOT NULL DEFAULT (''),
  reason_detail TEXT NOT NULL DEFAULT (''),
  request_id TEXT NOT NULL DEFAULT (''),
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_kyc_audit_log_user_created
  ON auth_kyc_audit_log (user_id(255), created_at(255));

CREATE TABLE IF NOT EXISTS auth_aml_screening_record (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  screening_ref TEXT NOT NULL DEFAULT (''),
  result TEXT NOT NULL,
  checked_at TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT (''),
  notes TEXT NOT NULL DEFAULT (''),
  created_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_aml_screening_user_checked
  ON auth_aml_screening_record (user_id(255), checked_at(255));

CREATE TABLE IF NOT EXISTS auth_kyc_limit_policy (
  tier VARCHAR(64) PRIMARY KEY,
  max_single_yuan INTEGER NOT NULL,
  max_daily_yuan INTEGER NOT NULL,
  recharge_allowed INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO auth_kyc_limit_policy (
  tier, max_single_yuan, max_daily_yuan, recharge_allowed, updated_at
) VALUES
  ('T0_unverified', 1000, 5000, 1, NOW()),
  ('T1_basic', 4999, 20000, 1, NOW()),
  ('T2_enhanced', 50000, 200000, 1, NOW());

-- Fix existing T0_unverified rows that were seeded with recharge_allowed=0
-- before this migration was updated (safe: only touches T0_unverified).
UPDATE auth_kyc_limit_policy
   SET recharge_allowed = 1,
       max_single_yuan = GREATEST(max_single_yuan, 1000),
       max_daily_yuan = GREATEST(max_daily_yuan, 5000),
       updated_at = NOW()
 WHERE tier = 'T0_unverified';
