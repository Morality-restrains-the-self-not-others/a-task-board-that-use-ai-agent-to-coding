-- 040: cloud_job_step_full_object — 评论/job 级 step_full.json COS 指针
-- 伸缩：每 job 一行（非时间累积热表）；访问键 (workspace_id, task_id, comment_id, job_id)
-- 主键 Snowflake 字符串；禁止 AUTO_INCREMENT
-- backend=local 时 payload_json 存全文；COS 时仅指针 + object_key

CREATE TABLE IF NOT EXISTS cloud_job_step_full_object (
  id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64) NOT NULL DEFAULT '',
  workspace_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  comment_id VARCHAR(64) NOT NULL,
  job_id VARCHAR(128) NOT NULL,
  layer_id VARCHAR(128) NOT NULL DEFAULT '',
  object_key VARCHAR(512) NOT NULL,
  etag VARCHAR(128) NOT NULL DEFAULT '',
  bytes INT NOT NULL DEFAULT 0,
  source VARCHAR(16) NOT NULL DEFAULT 'local',
  payload_json MEDIUMTEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ws_task_comment_job (workspace_id, task_id, comment_id, job_id),
  KEY idx_comment (workspace_id, task_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
