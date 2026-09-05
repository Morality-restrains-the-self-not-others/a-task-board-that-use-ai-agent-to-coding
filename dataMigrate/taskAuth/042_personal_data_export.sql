-- 042: 个人信息导出快照（PIPL 合规「导出权」，与 035 注销流程同族）
-- 每次生成覆盖同用户旧行（惰性清理，天然每用户最多 1 行）；7 天后过期（下载 410）。
CREATE TABLE IF NOT EXISTS auth_personal_data_export (
  id BIGINT NOT NULL PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'ready',   -- ready | partial（远端 section 有失败）
  content MEDIUMTEXT NOT NULL,                   -- JSON 快照（不含任何凭证原文）
  generated_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_auth_pde_user (user_id),
  INDEX idx_auth_pde_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
