-- 029_wechat_identity.sql — 微信身份统一表（v64 wechat-login-openid-unionid）
-- 用途: (app_key, openid) → user_id 与 unionid → user_id 双映射，支持跨应用登录/绑定/解绑
-- 幂等: CREATE TABLE IF NOT EXISTS + INSERT IGNORE 回填

CREATE TABLE IF NOT EXISTS wechat_identity (
  id           BIGINT PRIMARY KEY,
  user_id      BIGINT       NOT NULL,
  app_key      VARCHAR(64)  NOT NULL,            -- 应用逻辑名: 'web' / 'inapp' / 'miniapp'
  app_id       VARCHAR(64)  NOT NULL DEFAULT '', -- 微信 AppID (wx...)
  openid       VARCHAR(128) NOT NULL DEFAULT '', -- 应用维度 openid（空表示仅 unionid 映射）
  unionid      VARCHAR(128) NOT NULL DEFAULT '', -- 跨应用唯一标识
  nickname     VARCHAR(255) NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL,
  updated_at   DATETIME     NOT NULL,
  UNIQUE KEY uk_app_openid (app_key, openid),    -- MySQL UNIQUE 允许多 NULL，空 openid 行不冲突
  KEY idx_unionid (unionid),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 存量回填（幂等，INSERT IGNORE）
-- 规则:
--   1) 同一 user 有两行 wechat 登录方式（unionid 行 + openid 行）:
--      id 较小的一行视为 unionid 行（代码创建顺序: 先 unionid 后 openid；
--      snowflake id 递增；created_at 同事务相同不可用）
--      → 生成两条 wechat_identity: (openid='', unionid=先建行) + (openid=后建行, unionid=先建行)
--   2) 仅一行: 视为 unionid 行（qrconnect 扫码几乎总是返回 unionid；openid-only 的罕见
--      场景由登录路径的 auth_login_method 兜底查找收敛）
-- 存量 openid 行无 app 维度 → 全部归入 'web'（当前生产仅此一个微信应用，安全）

INSERT IGNORE INTO wechat_identity
  (id, user_id, app_key, app_id, openid, unionid, nickname, avatar_url, created_at, updated_at)
SELECT
  lm.id,
  lm.object_id,
  'web',
  '',
  '',
  lm.identifier,
  '',
  '',
  lm.created_at,
  lm.updated_at
FROM auth_login_method lm
WHERE lm.method_type = 'wechat'
  AND lm.binding_voided_at IS NULL
  AND NOT EXISTS (
    -- 该 user 有 id 更小的 wechat 登录方式 → 本行是 openid 行，不按 unionid 方式插入
    SELECT 1 FROM auth_login_method other
    WHERE other.object_id = lm.object_id
      AND other.method_type = 'wechat'
      AND other.binding_voided_at IS NULL
      AND other.id < lm.id
  );

INSERT IGNORE INTO wechat_identity
  (id, user_id, app_key, app_id, openid, unionid, nickname, avatar_url, created_at, updated_at)
SELECT
  lm.id,
  lm.object_id,
  'web',
  '',
  lm.identifier,
  COALESCE(u.identifier, ''),
  '',
  '',
  lm.created_at,
  lm.updated_at
FROM auth_login_method lm
JOIN auth_login_method u
  ON u.object_id = lm.object_id
 AND u.method_type = 'wechat'
 AND u.binding_voided_at IS NULL
 AND u.id < lm.id
WHERE lm.method_type = 'wechat'
  AND lm.binding_voided_at IS NULL;
