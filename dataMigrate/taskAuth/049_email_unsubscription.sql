-- 邮件邀请退订列表：规范化邮箱唯一。集合型（非时间流水），无需分区。
CREATE TABLE IF NOT EXISTS auth_email_unsubscription (
  id BIGINT NOT NULL PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'invite_email',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_auth_email_unsubscription_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
