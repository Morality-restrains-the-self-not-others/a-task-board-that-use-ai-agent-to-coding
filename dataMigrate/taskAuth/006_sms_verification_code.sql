-- SMS / OTP verification codes (owner: task-auth)
CREATE TABLE IF NOT EXISTS auth_sms_verification_code (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  phone VARCHAR(24) NULL,
  country_calling_code VARCHAR(8) NULL,
  user_id VARCHAR(36) NULL,
  email VARCHAR(254) NULL,
  code VARCHAR(6) NOT NULL,
  created_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  is_used BOOL NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX auth_sms_vc_phone_cc_code
  ON auth_sms_verification_code (phone, country_calling_code, code);
CREATE INDEX auth_sms_vc_email_code
  ON auth_sms_verification_code (email, code);
