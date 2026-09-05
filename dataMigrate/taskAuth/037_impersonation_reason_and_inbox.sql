-- ADR-0038: 模拟登录必填理由 + 用户收信箱。
-- 年增量预估 < 10 万：热表 + created_at 索引；归档见 OPT。

ALTER TABLE auth_impersonation_session
  ADD COLUMN reason VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '代登理由，不进访问日志全文' AFTER idempotency_key;

CREATE TABLE IF NOT EXISTS auth_user_inbox_message (
  id BIGINT NOT NULL,
  recipient_user_id VARCHAR(36) NOT NULL,
  kind VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  body TEXT NOT NULL,
  reason TEXT NOT NULL,
  actor_user_id VARCHAR(36) NOT NULL DEFAULT '',
  impersonation_session_id BIGINT NULL,
  created_at DATETIME NOT NULL,
  read_at DATETIME NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_auth_inbox_impersonation_session (impersonation_session_id),
  KEY idx_auth_inbox_recipient_created (recipient_user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
