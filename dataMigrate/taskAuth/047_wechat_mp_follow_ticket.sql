-- 047_wechat_mp_follow_ticket.sql
-- 推荐页动态 scene 二维码票据。PK = Snowflake = 微信 QR_STR_SCENE scene_str。
-- 伸缩：每用户热路径一行 pending；年增量远低于百万。过期 pending 由 timer worker 清理（禁止 taskAuth ticker）。

CREATE TABLE IF NOT EXISTS auth_wechat_mp_follow_ticket (
  id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  wechat_ticket VARCHAR(512) NOT NULL DEFAULT '',
  expire_at DATETIME NOT NULL,
  conflict_owner_user_id BIGINT NULL,
  conflict_code VARCHAR(64) NOT NULL DEFAULT '',
  mp_openid VARCHAR(128) NOT NULL DEFAULT '',
  unionid VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (id),
  KEY idx_auth_mp_follow_ticket_user_status_expire (user_id, status, expire_at),
  KEY idx_auth_mp_follow_ticket_expire (expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
