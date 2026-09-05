-- SuperAdmin 表：user_id 存在即超管（auth.db 真源）
-- [MySQL compat] removed: PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS auth_super_admin (
  user_id varchar(36) NOT NULL PRIMARY KEY REFERENCES auth_user (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 回填已有 is_superuser=1 的用户
INSERT IGNORE INTO auth_super_admin (user_id)
SELECT id FROM auth_user WHERE is_superuser = 1;
