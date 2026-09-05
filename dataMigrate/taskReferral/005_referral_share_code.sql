-- Per-user shareable access code (opaque, not derived from user_id).
-- Scale: 1 row per user (not time-accumulating). No RANGE partition.
-- PK is the code itself (globally unique); no AUTO_INCREMENT.

CREATE TABLE IF NOT EXISTS referral_share_code (
  code VARCHAR(32) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (code),
  UNIQUE KEY uk_referral_share_code_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
