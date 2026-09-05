-- OPT-20260823-017: 模拟登录会话与收信箱归档表（冷热分离）。
--
-- auth_impersonation_session 与 auth_user_inbox_message 为时间累积型审计表（年增量 <10 万），
-- 本期只建热表。归档表用 `CREATE TABLE ... LIKE` 复制热表结构，热表后续加列会自动同步；
-- 去掉唯一键（token_key / impersonation_session_id）允许归档重放（崩溃后重复归档不冲突）；
-- 加 archived_at 记录归档时间。归档由 db/scripts/archive_impersonation_sessions.py 分批执行。

CREATE TABLE IF NOT EXISTS auth_impersonation_session_archive LIKE auth_impersonation_session;
ALTER TABLE auth_impersonation_session_archive
  DROP KEY uk_auth_impersonation_token,
  DROP KEY idx_auth_impersonation_idem;
ALTER TABLE auth_impersonation_session_archive
  ADD COLUMN archived_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '归档时间' AFTER reason,
  ADD KEY idx_archive_archived_at (archived_at);

CREATE TABLE IF NOT EXISTS auth_user_inbox_message_archive LIKE auth_user_inbox_message;
ALTER TABLE auth_user_inbox_message_archive
  DROP KEY uk_auth_inbox_impersonation_session;
ALTER TABLE auth_user_inbox_message_archive
  ADD COLUMN archived_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '归档时间' AFTER read_at,
  ADD KEY idx_archive_archived_at (archived_at);
