-- 访问令牌审计日志表：记录令牌的创建、使用、吊销事件
CREATE TABLE IF NOT EXISTS auth_user_access_token_audit (
  id bigint NOT NULL PRIMARY KEY,
  user_id varchar(36) NOT NULL,
  token_id varchar(36) NOT NULL DEFAULT '',
  action varchar(20) NOT NULL,
  detail varchar(255) NOT NULL DEFAULT '',
  created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX auth_user_access_token_audit_user_id ON auth_user_access_token_audit (user_id);
CREATE INDEX auth_user_access_token_audit_token_id ON auth_user_access_token_audit (token_id);
