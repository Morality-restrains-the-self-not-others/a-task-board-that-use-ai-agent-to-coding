-- Referral code applications + policy (owner: taskReferral)
-- Users apply for referral codes; admin can approve/reject or set open-quota mode.

CREATE TABLE IF NOT EXISTS referral_code (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,
  status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'expired')) DEFAULT 'pending',
  applied_at VARCHAR(255) NOT NULL,
  approved_at VARCHAR(255) NULL,
  expires_at VARCHAR(255) NULL,
  rejected_at VARCHAR(255) NULL,
  reject_reason VARCHAR(512) NOT NULL DEFAULT '',
  reviewed_by VARCHAR(512) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_referral_code_user_status
  ON referral_code (user_id, status);

CREATE INDEX idx_referral_code_status_applied
  ON referral_code (status, applied_at);

CREATE TABLE IF NOT EXISTS referral_policy (
  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
  mode TEXT NOT NULL CHECK (mode IN ('open', 'approval')) DEFAULT ('approval'),
  message TEXT NOT NULL DEFAULT (''),
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO referral_policy (
  singleton_key, mode, message, updated_at
) VALUES ('global', 'approval', '', NOW());
