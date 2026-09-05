-- 邮箱注册邀请表：管理员可通过邮箱邀请用户注册
-- token 有效期为 7 天，过期后自动失效
CREATE TABLE IF NOT EXISTS auth_email_registration_invite (
  id bigint NOT NULL PRIMARY KEY,
  email varchar(255) NOT NULL,
  token varchar(128) NOT NULL UNIQUE,
  inviter_user_id varchar(36) NOT NULL,
  status varchar(20) NOT NULL DEFAULT 'pending',  -- pending / accepted / expired
  expires_at datetime NOT NULL,
  created_at datetime NOT NULL,
  accepted_at datetime NULL,
  accepted_user_id varchar(36) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX email_invite_email ON auth_email_registration_invite (email);
CREATE INDEX email_invite_token ON auth_email_registration_invite (token);
CREATE INDEX email_invite_status ON auth_email_registration_invite (status);
