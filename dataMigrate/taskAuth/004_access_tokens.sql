-- 访问令牌表：用户可创建个人访问令牌，用于替代密码登录或 API 认证
CREATE TABLE IF NOT EXISTS auth_user_access_token (
  id bigint NOT NULL PRIMARY KEY,
  user_id varchar(36) NOT NULL,
  name varchar(255) NOT NULL,
  token_hash varchar(128) NOT NULL UNIQUE,
  last_4 varchar(4) NOT NULL,
  created_at datetime NOT NULL,
  last_used_at datetime NULL,
  expires_at datetime NULL,
  is_revoked TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX auth_user_access_token_user_id ON auth_user_access_token (user_id, is_revoked);
