-- 意见与建议链接组（平台配置，按租户累计消耗可见）
-- 年增量远小于 10 万，不分区。主键 Snowflake 字符串。无物理外键。

CREATE TABLE IF NOT EXISTS billing_feedback_link_group (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  enabled TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_feedback_link_threshold (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  group_id VARCHAR(64) NOT NULL,
  resource_kind VARCHAR(64) NOT NULL,
  min_quantity DECIMAL(20,6) NOT NULL,
  UNIQUE KEY uk_feedback_threshold_group_kind (group_id, resource_kind),
  KEY idx_feedback_threshold_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_feedback_link (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  group_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  url VARCHAR(2048) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  enabled TINYINT NOT NULL DEFAULT 1,
  KEY idx_feedback_link_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_feedback_resource_kind (
  kind VARCHAR(64) NOT NULL PRIMARY KEY,
  display_name VARCHAR(255) NOT NULL,
  unit VARCHAR(32) NOT NULL,
  enabled TINYINT NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS billing_feedback_idempotency (
  idempotency_key VARCHAR(200) NOT NULL PRIMARY KEY,
  method VARCHAR(8) NOT NULL,
  group_id VARCHAR(64) NOT NULL DEFAULT '',
  status_code INT NOT NULL,
  response_body MEDIUMTEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO billing_feedback_resource_kind (kind, display_name, unit, enabled) VALUES
('task_post', '任务帖消耗', '帖', 1),
('gitlab_traffic', 'GitLab 流量', 'GB', 1),
('gitlab_disk', 'GitLab 磁盘已用', 'GB', 1),
('consumed_amount', '已消费金额', '分', 1);
