-- 046_wechat_mp_subscribe_pending.sql
-- 服务号关注事件在 unionid 尚未对应平台用户时的挂起行。
-- 伸缩：关注量远低于百万/年；unionid PK；绑定后删除。TTL 清理见 OPT。

CREATE TABLE IF NOT EXISTS auth_wechat_mp_subscribe_pending (
  unionid VARCHAR(128) NOT NULL,
  openid VARCHAR(128) NOT NULL DEFAULT '',
  app_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  PRIMARY KEY (unionid),
  KEY idx_openid (openid),
  KEY idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
