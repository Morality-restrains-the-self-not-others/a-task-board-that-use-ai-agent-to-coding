-- 013: User profile sync table (migrated from Django saas auth_user_profile)
-- owner: taskAuth
-- Stores user avatar + display name synced from USER_CREATED events.

CREATE TABLE IF NOT EXISTS auth_user_profile (
  user_id VARCHAR(64) NOT NULL PRIMARY KEY,
  username VARCHAR(255) NOT NULL DEFAULT '',
  avatar TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
