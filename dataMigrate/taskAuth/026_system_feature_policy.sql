-- auth_system_feature_policy: platform-wide feature toggles (login methods, SMS verification, region whitelist).
-- OPT-049: Migrated from Django saas-backend to taskAuth 2026-07-30.
-- Previously Django's auth_system_feature_policy model; now owned by taskAuth Go service.
-- 默认策略（2026-08-09 产品调整）：邮箱注册默认关闭、微信登录默认开启、手机号区域默认仅 +86（中国）。
CREATE TABLE IF NOT EXISTS auth_system_feature_policy (
  id INT NOT NULL PRIMARY KEY AUTO_INCREMENT,
  enable_phone_login TINYINT(1) NOT NULL DEFAULT 1,
  enable_recharge_phone_verification TINYINT(1) NOT NULL DEFAULT 0,
  enable_email_register TINYINT(1) NOT NULL DEFAULT 0,
  enable_wechat_login TINYINT(1) NOT NULL DEFAULT 1,
  allowed_phone_country_codes JSON NOT NULL DEFAULT ('["+86"]'),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed default row so feature policy queries always return something.
INSERT IGNORE INTO auth_system_feature_policy (id, enable_phone_login, enable_email_register, enable_wechat_login, enable_recharge_phone_verification, allowed_phone_country_codes)
VALUES (1, 1, 0, 1, 0, '["+86"]');
