-- 042: cloud_comment_startup_log_object — 评论级启动日志 COS 指针
-- 伸缩：每评论一行（非时间累积热表）；访问键 (workspace_id, task_id, comment_id)
-- 主键 Snowflake 字符串；禁止 AUTO_INCREMENT
-- backend=local 时 payload_json 存全文；COS 时仅指针 + object_key
-- 热路径仍写 ADR-0023 分片表；本表供冷打开 hydrate

CREATE TABLE IF NOT EXISTS cloud_comment_startup_log_object (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
  workspace_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  etag VARCHAR(128) NOT NULL DEFAULT '',
  bytes INT NOT NULL DEFAULT 0,
  source VARCHAR(16) NOT NULL DEFAULT 'local',
  payload_json MEDIUMTEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ws_task_comment (workspace_id, task_id, comment_id),
  KEY idx_task (workspace_id, company_id, task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
