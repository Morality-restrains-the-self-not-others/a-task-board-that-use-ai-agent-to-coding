-- Recharge SMS gate + pending phone (owner: task-auth)
CREATE TABLE IF NOT EXISTS auth_recharge_sms_gate (
  user_id VARCHAR(64) NOT NULL PRIMARY KEY,
  sms_ok_until TEXT NULL,
  pending_phone TEXT NULL,
  pending_until TEXT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
